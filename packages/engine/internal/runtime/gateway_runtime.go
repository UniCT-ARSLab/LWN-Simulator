package runtime

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/protocol"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

// External network servers use wall-clock receive windows and cannot follow
// an accelerated or slowed simulation clock.
func requiresRealtimeTransport(gateways []contracts.Gateway, factory types.GatewayAdapterFactory) bool {
	if factory == nil {
		return false
	}
	if virtualFactory, ok := factory.(types.VirtualGatewayAdapterFactory); ok && virtualFactory.SupportsVirtualGateways() {
		return true
	}
	for _, gateway := range gateways {
		if gateway.Type == contracts.Real {
			return true
		}
	}
	return false
}

func (e *Engine) startGatewayAdapters(ctx context.Context) {
	if e.gatewayAdapterFactory == nil {
		return
	}

	for _, gateway := range e.registry.ActiveGateways() {
		if gateway.Type == contracts.Virtual {
			virtualFactory, ok := e.gatewayAdapterFactory.(types.VirtualGatewayAdapterFactory)
			if !ok || !virtualFactory.SupportsVirtualGateways() {
				continue
			}
		} else if gateway.Type != contracts.Real {
			continue
		}
		e.startGatewayAdapter(ctx, gateway)
	}
}

func (e *Engine) startGatewayAdapter(ctx context.Context, gateway contracts.Gateway) {
	if e.gatewayAdapterFactory == nil || ctx == nil {
		return
	}
	adapter, err := e.gatewayAdapterFactory.NewGatewayAdapter(gateway)
	if err != nil {
		e.publishGatewayError(gateway.ID, fmt.Errorf("create gateway adapter: %w", err), false)
		return
	}

	e.mu.Lock()
	e.gatewayAdapters[gateway.ID] = adapter
	connecting := e.updateGatewayRuntimeLocked(gateway.ID, contracts.Connecting, "", true)
	e.mu.Unlock()
	e.publish(connecting)

	e.gatewayWG.Add(1)
	go func() {
		defer e.gatewayWG.Done()
		e.consumeGatewayAdapter(ctx, gateway.ID, adapter)
	}()
	if err := adapter.Start(ctx); err != nil {
		e.publishGatewayError(gateway.ID, fmt.Errorf("start gateway adapter: %w", err), false)
		_ = adapter.Close()
	}
}

func (e *Engine) consumeGatewayAdapter(ctx context.Context, gatewayID string, adapter types.GatewayAdapter) {
	packets := adapter.Packets()
	adapterEvents := adapter.Events()

	for packets != nil || adapterEvents != nil {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-adapterEvents:
			if !ok {
				adapterEvents = nil
				continue
			}
			if event.GatewayID == "" {
				event.GatewayID = gatewayID
			}
			e.handleGatewayAdapterEvent(event)
		case packet, ok := <-packets:
			if !ok {
				packets = nil
				continue
			}
			if packet.GatewayID == "" {
				packet.GatewayID = gatewayID
			}
			e.handleGatewayPacket(packet)
		}
	}
}

func (e *Engine) handleGatewayAdapterEvent(adapterEvent types.GatewayAdapterEvent) {
	e.mu.Lock()

	runtime, ok := e.gatewayRuntime[adapterEvent.GatewayID]
	if !ok {
		e.mu.Unlock()
		return
	}

	if adapterEvent.Heartbeat {
		runtime.State = contracts.Connected
		runtime.LastHeartbeat = e.clock.Now()
		e.metrics.GatewayHeartbeats++
		event := e.newGatewayEventLocked(
			contracts.GatewayHeartbeat,
			"gateway heartbeat received",
			adapterEvent.GatewayID,
			contracts.Connected,
			nil,
			"",
			"",
		)
		e.mu.Unlock()
		e.publish(event)
		return
	}

	wasConnected := runtime.State == contracts.Connected
	var pendingPackets []types.GatewayPacket
	eventType := contracts.GatewayNetworkError
	message := "gateway network error"
	events := make([]contracts.SimulationEvent, 0, 2)
	switch adapterEvent.State {
	case contracts.Connected:
		runtime.State = contracts.Connected
		runtime.LastNetworkError = ""
		e.metrics.GatewayConnections++
		eventType = contracts.GatewayConnected
		message = "gateway connected"
		if !wasConnected {
			pendingPackets = e.takeGatewayPacketsLocked(adapterEvent.GatewayID)
		}
	case contracts.Disconnected:
		runtime.State = contracts.Disconnected
		eventType = contracts.GatewayDisconnected
		message = "gateway disconnected"
	case contracts.Reconnecting:
		runtime.State = contracts.Reconnecting
		runtime.ConnectionAttempts++
		e.metrics.GatewayReconnects++
		eventType = contracts.GatewayReconnecting
		message = "gateway reconnecting"
	case contracts.Error:
		runtime.State = contracts.Error
		runtime.LastNetworkError = adapterEvent.Error
		e.metrics.GatewayNetworkErrors++
		if adapterEvent.Timeout {
			e.metrics.GatewayTimeouts++
		}
		if adapterEvent.Packet != nil {
			copyPacket := *adapterEvent.Packet
			copyPacket.Payload = append([]byte(nil), adapterEvent.Packet.Payload...)
			e.gatewayPackets = append(e.gatewayPackets, copyPacket)
		}
	default:
		e.mu.Unlock()
		return
	}

	events = append(events, e.newGatewayEventLocked(
		eventType,
		message,
		adapterEvent.GatewayID,
		adapterEvent.State,
		nil,
		"",
		adapterEvent.Error,
	))
	if adapterEvent.State == contracts.Connected && !wasConnected {
		if beacon := e.rescheduleRealGatewayBeaconLocked(adapterEvent.GatewayID); beacon != nil {
			events = append(events, *beacon)
		}
	}
	e.mu.Unlock()
	for _, event := range events {
		e.publish(event)
	}
	if len(pendingPackets) > 0 {
		e.dispatchGatewayPackets(pendingPackets)
	}
}

func (e *Engine) handleGatewayPacket(packet types.GatewayPacket) {
	e.mu.Lock()
	runtime, ok := e.gatewayRuntime[packet.GatewayID]
	if !ok {
		e.mu.Unlock()
		return
	}
	runtime.IngressPackets++
	e.metrics.GatewayIngressPackets++
	payloadSize := int64(len(packet.Payload))
	event := e.newGatewayEventLocked(
		contracts.GatewayPacketIngress,
		"packet received from gateway transport",
		packet.GatewayID,
		contracts.Connected,
		&payloadSize,
		"",
		"",
	)
	if packet.Frequency > 0 {
		event.ChannelFrequency = &packet.Frequency
	}
	if packet.Bandwidth > 0 {
		event.Bandwidth = &packet.Bandwidth
	}
	if packet.SpreadingFactor > 0 {
		spreadingFactor := int32(packet.SpreadingFactor)
		event.SpreadingFactor = &spreadingFactor
	}
	if packet.RSSI != 0 {
		rssi := packet.RSSI
		event.RSSI = &rssi
	}
	if packet.SNR != 0 {
		snr := packet.SNR
		event.SNR = &snr
	}
	classified, classificationErr := classifyGatewayPacket(packet)
	if packet.Protocol == "" && packet.DeviceID != "" {
		if device, exists := e.registry.Device(packet.DeviceID); exists {
			classified, classificationErr = protocol.Classify(packet.Payload, deviceReceiveMode(device))
		}
	}
	if classificationErr == nil {
		packet.Protocol = classified.Protocol
		event.RadioProtocol = &classified.Protocol
		if classified.Protocol == contracts.RawLoRa {
			senderID := packet.DeviceID
			if senderID == "" {
				senderID = packet.GatewayID
			}
			event.Message = "raw LoRa frame received from gateway transport"
			event.RadioFrame = radioFrameFromGatewayPacket(packet, senderID, event.TimestampMilliseconds)
		}
	}
	e.eventLog[len(e.eventLog)-1] = event
	events := []contracts.SimulationEvent{event}
	if packet.Kind == types.GatewayPacketDownlink || packet.Kind == types.GatewayPacketClassBDownlink {
		if classificationErr != nil {
			events = append(events, e.newPacketEventLocked(
				contracts.DeviceDownlinkDropped,
				fmt.Sprintf("downlink received from gateway transport could not be classified: %v", classificationErr),
				packet.DeviceID, packet.GatewayID, "", uuid.NewString(),
			))
		} else if classified.Protocol == contracts.RawLoRa && packet.DeviceID != "" {
			events = append(events, e.processRawDownlinkLocked(packet)...)
		} else if classified.Protocol == contracts.LoRaWAN {
			events = append(events, e.processExternalDownlinkLocked(packet, classified.Packet)...)
		}
	} else if classificationErr != nil {
		events = append(events, e.newPacketEventLocked(
			contracts.PacketDropped,
			fmt.Sprintf("uplink received from gateway transport could not be classified: %v", classificationErr),
			packet.DeviceID, packet.GatewayID, "", uuid.NewString(),
		))
	} else if classified.Protocol == contracts.RawLoRa && packet.DeviceID != "" {
		events = append(events, e.processRawUplinkLocked(packet)...)
	} else if classified.Protocol == contracts.LoRaWAN {
		events = append(events, e.processRealUplinkLocked(packet, classified.Packet)...)
	}
	pendingPackets := e.takeGatewayPacketsLocked(packet.GatewayID)
	e.mu.Unlock()
	for _, event := range events {
		e.publish(event)
	}
	if len(pendingPackets) > 0 {
		e.dispatchGatewayPackets(pendingPackets)
	}
}

func classifyGatewayPacket(packet types.GatewayPacket) (protocol.ClassifiedFrame, error) {
	switch packet.Protocol {
	case contracts.LoRaWAN:
		return protocol.Classify(packet.Payload, contracts.ReceiveLoRaWAN)
	case contracts.RawLoRa:
		return protocol.Classify(packet.Payload, contracts.ReceiveRawLoRa)
	case "":
		return protocol.Classify(packet.Payload, contracts.ReceiveAuto)
	default:
		return protocol.ClassifiedFrame{}, fmt.Errorf("unsupported radio protocol %q", packet.Protocol)
	}
}

func (e *Engine) processRawUplinkLocked(packet types.GatewayPacket) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(packet.DeviceID)
	if !ok || session == nil || !device.Active || !acceptsRadioProtocol(device, contracts.RawLoRa) {
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.PacketDropped,
			"raw LoRa uplink was rejected because the device does not accept raw LoRa",
			packet.DeviceID, packet.GatewayID, "", uuid.NewString(),
		)}
	}
	payloadSize := int64(len(packet.Payload))
	e.metrics.TotalUplinks++
	e.metrics.SuccessfulUplinks++
	e.metrics.TotalPacketsReceived++
	e.metrics.PacketSuccessRate = e.packetSuccessRate()
	if packet.RSSI != 0 {
		session.LastRSSI = packet.RSSI
	}
	if packet.SNR != 0 {
		session.LastSNR = packet.SNR
	}
	if packet.Frequency > 0 {
		session.LastChannel = packet.Frequency
	}
	session.LastPayloadSize = len(packet.Payload)
	packetID := uuid.NewString()
	event := e.newProtocolEventLocked(
		contracts.DeviceUplinkTransmitted,
		"raw LoRa frame received from gateway",
		device.ID, packet.GatewayID, "", packetID, -1, 1, false, nil,
	)
	event.PayloadSize = &payloadSize
	event.RadioFrame = radioFrameFromGatewayPacket(packet, device.ID, event.TimestampMilliseconds)
	e.setUplinkOutcomeLocked(&event, contracts.UplinkAccepted)
	return []contracts.SimulationEvent{event}
}

func (e *Engine) processRawDownlinkLocked(packet types.GatewayPacket) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(packet.DeviceID)
	if !ok || session == nil {
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.DeviceDownlinkDropped,
			"raw LoRa downlink has no known target device",
			packet.DeviceID, packet.GatewayID, "", uuid.NewString(),
		)}
	}
	if !device.Active || !acceptsRadioProtocol(device, contracts.RawLoRa) {
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.DeviceDownlinkDropped,
			"raw LoRa downlink was rejected because the device does not accept raw LoRa",
			device.ID, packet.GatewayID, "", uuid.NewString(),
		)}
	}
	downlink := types.Downlink{ID: uuid.NewString(), DeviceID: device.ID, Payload: append([]byte(nil), packet.Payload...), DataRate: gatewayPacketDataRate(device, packet)}
	event := e.newDownlinkEventLocked(
		contracts.DeviceDownlinkTransmitted,
		"raw LoRa downlink received from gateway transport",
		device, session, downlink, "",
	)
	event.GatewayID = &packet.GatewayID
	event.FPort = nil
	event.FrameCounter = nil
	event.Confirmed = nil
	event.RadioFrame = radioFrameFromGatewayPacket(packet, packet.GatewayID, event.TimestampMilliseconds)
	e.eventLog[len(e.eventLog)-1] = event
	return []contracts.SimulationEvent{event}
}

func radioFrameFromGatewayPacket(packet types.GatewayPacket, senderID string, timestamp int64) *contracts.RadioFrame {
	frequency := packet.Frequency
	if frequency <= 0 {
		frequency = 1
	}
	bandwidth := packet.Bandwidth
	if bandwidth <= 0 {
		bandwidth = radioBandwidthHz
	}
	spreadingFactor := packet.SpreadingFactor
	if spreadingFactor < 5 {
		spreadingFactor = 7
	}
	frame := &contracts.RadioFrame{
		Protocol:              packet.Protocol,
		Payload:               append([]byte(nil), packet.Payload...),
		Bandwidth:             bandwidth,
		CodingRate:            "4/5",
		Frequency:             frequency,
		SenderID:              senderID,
		SpreadingFactor:       int32(spreadingFactor),
		TimestampMilliseconds: timestamp,
		TransmissionPower:     int32(packet.Power),
	}
	if packet.GatewayID != "" {
		frame.GatewayID = &packet.GatewayID
	}
	return frame
}

func (e *Engine) processExternalDownlinkLocked(packet types.GatewayPacket, decoded lorawan.Packet) []contracts.SimulationEvent {
	if decoded.MType == lorawan.MTypeJoinAccept {
		return e.processExternalJoinAcceptLocked(packet, decoded)
	}
	if decoded.MType != lorawan.MTypeUnconfirmedDataDown && decoded.MType != lorawan.MTypeConfirmedDataDown {
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.DeviceDownlinkDropped,
			"external gateway downlink is not a data frame",
			"", packet.GatewayID, "", uuid.NewString(),
		)}
	}
	device, session := e.realDeviceSessionLocked(decoded.DevAddr)
	if session == nil {
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.DeviceDownlinkDropped,
			"external gateway downlink targets an unknown device address",
			"", packet.GatewayID, "", uuid.NewString(),
		)}
	}
	if !acceptsRadioProtocol(device, contracts.LoRaWAN) {
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.DeviceDownlinkDropped,
			"LoRaWAN downlink was rejected because the device does not accept LoRaWAN",
			device.ID, packet.GatewayID, "", uuid.NewString(),
		)}
	}
	if len(session.NwkSKey) != 16 || !lorawanHandler.VerifyDataMIC(decoded, session.NwkSKey, 1) {
		e.metrics.FrameCounterErrors++
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.DeviceDownlinkDropped,
			"external gateway downlink has an invalid MIC",
			device.ID, packet.GatewayID, "", uuid.NewString(),
		)}
	}
	if decoded.ACK && !e.externalACKMatchesUplink(device, session, packet) {
		return []contracts.SimulationEvent{e.newPacketEventLocked(contracts.DeviceDownlinkDropped,
			"external ACK is outside the pending uplink receive windows", device.ID, packet.GatewayID, "", uuid.NewString())}
	}
	if session.ExternalDownlinkFrameCounterSet && decoded.FCnt <= session.LastExternalDownlinkFrameCounter {
		e.metrics.FrameCounterErrors++
		return []contracts.SimulationEvent{e.newProtocolEventLocked(
			contracts.FrameCounterRejected,
			"external gateway downlink frame counter is not newer",
			device.ID, packet.GatewayID, "", uuid.NewString(), int64(decoded.FCnt), 1, decoded.Confirmed, nil,
		)}
	}
	downlink := types.Downlink{
		ID:        uuid.NewString(),
		DeviceID:  device.ID,
		DataRate:  gatewayPacketDataRate(device, packet),
		Confirmed: decoded.Confirmed,
		FPending:  decoded.FPending,
		ACK:       decoded.ACK,
	}
	if decoded.FPort != nil {
		downlink.FPort = int(*decoded.FPort)
		key := session.AppSKey
		if *decoded.FPort == 0 {
			key = session.NwkSKey
		}
		payload, err := lorawanHandler.DecryptPayload(decoded, key, 1)
		if err != nil {
			return []contracts.SimulationEvent{e.newPacketEventLocked(
				contracts.DeviceDownlinkDropped,
				"external gateway downlink payload could not be decrypted",
				device.ID, packet.GatewayID, "", uuid.NewString(),
			)}
		}
		downlink.Payload = payload
	}
	commands, err := externalMACCommands(decoded, downlink.Payload)
	if err != nil {
		return []contracts.SimulationEvent{e.newPacketEventLocked(contracts.DeviceDownlinkDropped, "invalid external MAC commands: "+err.Error(), device.ID, packet.GatewayID, "", uuid.NewString())}
	}
	e.applyExternalMACLocked(device, session, commands)
	if decoded.Confirmed {
		session.PendingDownlinkACK = true
	}
	session.LastExternalDownlinkFrameCounter = decoded.FCnt
	session.ExternalDownlinkFrameCounterSet = true
	session.FrameCounterDown = int64(decoded.FCnt)
	event := e.newDownlinkEventLocked(
		contracts.DeviceDownlinkTransmitted,
		"downlink received from external gateway transport",
		device, session, downlink, "",
	)
	event.GatewayID = &packet.GatewayID
	if packet.Frequency > 0 {
		event.ChannelFrequency = &packet.Frequency
	}
	if packet.SpreadingFactor > 0 {
		spreadingFactor := int32(packet.SpreadingFactor)
		event.SpreadingFactor = &spreadingFactor
	}
	if packet.Bandwidth > 0 {
		event.Bandwidth = &packet.Bandwidth
	}
	e.eventLog[len(e.eventLog)-1] = event
	events := []contracts.SimulationEvent{event}
	if decoded.ACK {
		events = append(events, e.completeExternalUplinkACKLocked(device, session, packet.GatewayID)...)
	}
	events = append(events, e.applyDownlinkEffectsLocked(device, session, downlink, e.clock.Now())...)
	return events
}

func (e *Engine) completeExternalUplinkACKLocked(device contracts.Device, session *types.DeviceSession, gatewayID string) []contracts.SimulationEvent {
	pending := session.PendingUplink
	if pending == nil || !pending.Confirmed {
		return nil
	}
	e.metrics.AcknowledgedUplinks++
	events := []contracts.SimulationEvent{e.newProtocolEventLocked(
		contracts.UplinkACKReceived,
		"uplink acknowledgement received from external network server",
		device.ID,
		gatewayID,
		"",
		pending.PacketID,
		pending.FrameCounter,
		pending.Attempt,
		true,
		nil,
	)}
	if pending.FragmentIndex+1 < pending.FragmentCount {
		pending.FragmentIndex++
		pending.WirePayload = nil
		pending.Attempt = 1
		pending.GatewayIDs = nil
		next := types.ScheduledEvent{
			ID: uuid.NewString(), At: e.clock.Now(), Type: contracts.DeviceUplinkTransmitted,
			Message: "next application payload fragment scheduled", DeviceID: device.ID,
			Kind: types.ScheduledEventDeviceUplink, Attempt: 1, PacketID: pending.PacketID,
			FrameCounter: pending.FrameCounter, Confirmed: true,
			FragmentIndex: pending.FragmentIndex, FragmentCount: pending.FragmentCount,
		}
		if err := e.scheduler.Schedule(next); err != nil {
			session.PendingUplink = nil
			return append(events, e.newEventLocked(contracts.PacketDropped, "next payload fragment could not be scheduled", device.ID, "", next.ID))
		}
		return append(events, e.newEventLocked(contracts.DeviceUplinkScheduled, "next application payload fragment scheduled", device.ID, "", next.ID))
	}
	session.PendingUplink = nil
	e.metrics.SuccessfulUplinks++
	e.metrics.PacketSuccessRate = e.packetSuccessRate()
	events = append(events, e.newUplinkOutcomeEventLocked(contracts.MetricsUpdated, "simulation metrics updated", device.ID, gatewayID, "", pending.PacketID, contracts.UplinkSucceeded))
	return append(events, e.scheduleUplinkLocked(device, pending.TransmissionAt)...)
}

func (e *Engine) processExternalJoinAcceptLocked(packet types.GatewayPacket, decoded lorawan.Packet) []contracts.SimulationEvent {
	matchedCandidate := false
	for _, device := range e.registry.ActiveDevices() {
		if device.OOTAConfig == nil || !acceptsRadioProtocol(device, contracts.LoRaWAN) {
			continue
		}
		session := e.sessions[device.ID]
		if session == nil || session.PendingJoinRequest == nil ||
			!pendingJoinGateway(session.PendingJoinRequest, packet.GatewayID) {
			continue
		}
		appKey, err := lorawan.DecodeHex(device.OOTAConfig.AppKey)
		if err != nil {
			return []contracts.SimulationEvent{e.newPacketEventLocked(
				contracts.DeviceDownlinkDropped,
				"external Join-Accept could not be decoded because the device AppKey is invalid",
				device.ID, packet.GatewayID, "", uuid.NewString(),
			)}
		}
		matchedCandidate = true
		joinAccept, err := lorawan.DecodeJoinAccept(decoded.Raw, appKey)
		if err != nil {
			continue
		}
		if !packet.TransmitAt.IsZero() && !session.PendingJoinRequest.RequestedAt.IsZero() && packet.TransmitAt.Before(session.PendingJoinRequest.RequestedAt) {
			return []contracts.SimulationEvent{e.newPacketEventLocked(
				contracts.DeviceDownlinkDropped,
				"external Join-Accept belongs to an expired join attempt",
				device.ID, packet.GatewayID, "", uuid.NewString(),
			)}
		}
		nwkSKey, appSKey, err := lorawan.DeriveSessionKeys(appKey, joinAccept.AppNonce, joinAccept.NetID, session.DevNonce)
		if err != nil {
			return []contracts.SimulationEvent{e.newPacketEventLocked(
				contracts.DeviceDownlinkDropped,
				"external Join-Accept could not derive the device session keys",
				device.ID, packet.GatewayID, "", uuid.NewString(),
			)}
		}
		session.Joined = true
		session.PendingJoinRequest = nil
		session.DeviceAddress = fmt.Sprintf("%08X", joinAccept.DevAddr)
		session.NwkSKey = nwkSKey
		session.AppSKey = appSKey
		session.RX1DataRateOffset = joinAccept.RX1DataRateOffset
		session.RX2DataRate = joinAccept.RX2DataRate
		session.ReceiveDelay = joinAcceptReceiveDelay(joinAccept.RXDelay)
		if len(joinAccept.CFList) == 16 && joinAccept.CFList[15] == 0 && device.LocationConfig.Region != contracts.US915 && device.LocationConfig.Region != contracts.AU915 {
			session.NetworkChannels = make(map[int]types.NetworkChannel)
			start := len(regionalChannels(device.LocationConfig.Region, 0))
			for i := 0; i < 5; i++ {
				b := joinAccept.CFList[i*3:]
				frequency := int64(uint32(b[0])|uint32(b[1])<<8|uint32(b[2])<<16) * 100
				if frequency > 0 && externalFrequencyValid(device.LocationConfig.Region, frequency) {
					session.NetworkChannels[start+i] = types.NetworkChannel{Frequency: frequency, MinimumDataRate: 0, MaximumDataRate: 5}
				}
			}
		}
		session.FrameCounterDown = 0
		packetID := uuid.NewString()
		e.metrics.JoinAccepts++
		events := []contracts.SimulationEvent{e.newProtocolEventLocked(
			contracts.JoinAcceptReceived,
			fmt.Sprintf("OTAA Join-Accept received from external network server for DevAddr %s", session.DeviceAddress),
			device.ID, packet.GatewayID, "", packetID, -1, 1, false, nil,
		)}
		return append(events, e.scheduleUplinkLocked(device, e.clock.Now())...)
	}
	if matchedCandidate {
		return []contracts.SimulationEvent{e.newPacketEventLocked(contracts.DeviceDownlinkDropped,
			"external Join-Accept was rejected because its MIC is invalid", "", packet.GatewayID, "", uuid.NewString())}
	}
	return []contracts.SimulationEvent{e.newPacketEventLocked(
		contracts.DeviceDownlinkDropped,
		"external Join-Accept targets a device without a pending join request",
		"", packet.GatewayID, "", uuid.NewString(),
	)}
}

func pendingJoinGateway(pending *types.PendingJoinRequest, gatewayID string) bool {
	if pending == nil {
		return false
	}
	if len(pending.GatewayIDs) == 0 {
		return true
	}
	for _, id := range pending.GatewayIDs {
		if id == gatewayID {
			return true
		}
	}
	return false
}

func (e *Engine) processRealUplinkLocked(packet types.GatewayPacket, decoded lorawan.Packet) []contracts.SimulationEvent {
	if decoded.MType == lorawan.MTypeJoinRequest {
		for _, device := range e.registry.ActiveDevices() {
			if !acceptsRadioProtocol(device, contracts.LoRaWAN) || device.OOTAConfig == nil || !sameEUI(device.OOTAConfig.JoinEUI, decoded.JoinEUI) || !sameEUI(device.DevEUI, decoded.DevEUI) {
				continue
			}
			appKey, keyErr := lorawan.DecodeHex(device.OOTAConfig.AppKey)
			if keyErr != nil || !lorawan.VerifyJoinRequestMIC(decoded, appKey) {
				return []contracts.SimulationEvent{e.newPacketEventLocked(
					contracts.PacketDropped,
					"OTAA join request from real gateway has an invalid MIC",
					"", packet.GatewayID, "", uuid.NewString(),
				)}
			}
			session := e.sessions[device.ID]
			if session == nil {
				return nil
			}
			e.metrics.JoinRequests++
			if session.Joined {
				return []contracts.SimulationEvent{e.newProtocolEventLocked(
					contracts.JoinRequestReceived,
					"OTAA join request received from real gateway",
					device.ID, packet.GatewayID, "", uuid.NewString(), -1, 1, false, nil,
				)}
			}
			session.DevNonce = decoded.DevNonce
			session.Joined = true
			session.JoinEUI = device.OOTAConfig.JoinEUI
			session.DeviceAddress = joinedDeviceAddress(session)
			deriveOTAAKeys(device, session)
			joinAcceptQueued := e.queueRealJoinAcceptLocked(packet, device, session, appKey)
			e.metrics.JoinAccepts++
			packetID := uuid.NewString()
			events := []contracts.SimulationEvent{
				e.newProtocolEventLocked(contracts.JoinRequestReceived, "OTAA join request received from real gateway", device.ID, packet.GatewayID, "", packetID, -1, 1, false, nil),
				e.newProtocolEventLocked(contracts.JoinAcceptReceived, "OTAA join accepted for real gateway uplink", device.ID, packet.GatewayID, "", packetID, -1, 1, false, nil),
			}
			if !joinAcceptQueued {
				return append(events, e.newPacketEventLocked(contracts.PacketDropped, "OTAA join accept could not be queued for the real gateway", device.ID, packet.GatewayID, "", packetID))
			}
			return events
		}
		return nil
	}
	if decoded.MType != lorawan.MTypeUnconfirmedDataUp && decoded.MType != lorawan.MTypeConfirmedDataUp {
		return nil
	}
	device, session := e.realDeviceSessionLocked(decoded.DevAddr)
	if session == nil {
		return nil
	}
	if !device.Active || !acceptsRadioProtocol(device, contracts.LoRaWAN) {
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.PacketDropped, "LoRaWAN uplink was rejected because the device is inactive or does not accept LoRaWAN",
			device.ID, packet.GatewayID, "", uuid.NewString(),
		)}
	}
	if len(session.NwkSKey) == 16 && !lorawanHandler.VerifyDataMIC(decoded, session.NwkSKey, 0) {
		return []contracts.SimulationEvent{e.newPacketEventLocked(
			contracts.PacketDropped,
			"real gateway uplink has an invalid MIC",
			device.ID, packet.GatewayID, "", uuid.NewString(),
		)}
	}
	if decoded.FPort != nil && len(decoded.FRMPayload) > 0 {
		key := session.AppSKey
		if *decoded.FPort == 0 {
			key = session.NwkSKey
		}
		if _, err := lorawanHandler.DecryptPayload(decoded, key, 0); err != nil {
			return []contracts.SimulationEvent{e.newPacketEventLocked(
				contracts.PacketDropped,
				"real gateway uplink payload could not be decrypted",
				device.ID, packet.GatewayID, "", uuid.NewString(),
			)}
		}
		if dataRate := gatewayPacketDataRate(device, packet); dataRate >= 0 {
			maximum := maximumPayloadSize(device.LocationConfig.Region, dataRate)
			if len(decoded.FRMPayload) > maximum {
				return []contracts.SimulationEvent{e.newPacketEventLocked(
					contracts.PacketDropped,
					fmt.Sprintf("real gateway uplink payload exceeds the regional maximum of %d bytes", maximum),
					device.ID, packet.GatewayID, "", uuid.NewString(),
				)}
			}
		}
	}
	if !device.FrameConfig.DisableFrameCounterValidation && decoded.FCnt <= session.LastExternalFrameCounter {
		frameCounter := int64(decoded.FCnt)
		e.metrics.FrameCounterErrors++
		return []contracts.SimulationEvent{e.newProtocolEventLocked(
			contracts.FrameCounterRejected,
			"real gateway uplink rejected because its frame counter is not newer",
			device.ID, packet.GatewayID, "", uuid.NewString(), frameCounter, 1, decoded.Confirmed, nil,
		)}
	}
	frameCounter := decoded.FCnt
	session.LastExternalFrameCounter = decoded.FCnt
	if int64(frameCounter) > session.FrameCounterUp {
		session.FrameCounterUp = int64(frameCounter)
	}
	e.metrics.TotalUplinks++
	e.metrics.SuccessfulUplinks++
	e.metrics.TotalPacketsReceived++
	e.metrics.PacketSuccessRate = e.packetSuccessRate()
	if packet.RSSI != 0 {
		session.LastRSSI = packet.RSSI
	}
	if packet.SNR != 0 {
		session.LastSNR = packet.SNR
	}
	if packet.Frequency > 0 {
		session.LastChannel = packet.Frequency
	}
	session.LastPayloadSize = len(decoded.FRMPayload)
	if decoded.FPort != nil {
		session.LastFPort = int(*decoded.FPort)
	}
	if dataRate := gatewayPacketDataRate(device, packet); dataRate >= 0 {
		session.CurrentDataRate = dataRate
		session.CurrentSpreadingFactor = spreadingFactorForDataRate(device.LocationConfig.Region, dataRate)
	}
	packetID := uuid.NewString()
	events := []contracts.SimulationEvent{e.newProtocolEventLocked(
		contracts.DeviceUplinkTransmitted,
		"uplink received from real gateway",
		device.ID, packet.GatewayID, "", packetID, int64(frameCounter), 1, decoded.Confirmed, nil,
	)}
	e.setUplinkOutcomeLocked(&events[0], contracts.UplinkAccepted)
	decodeKey := session.AppSKey
	if decoded.FPort != nil && *decoded.FPort == 0 {
		decodeKey = session.NwkSKey
	}
	decodedPayload, decodeErr := lorawanHandler.DecryptPayload(decoded, decodeKey, 0)
	if decoded.FPort != nil && len(decoded.FRMPayload) > 0 && decodeErr == nil {
		decorateDecodedPayload(&events[0], device, decodedPayload, int(*decoded.FPort))
		e.eventLog[len(e.eventLog)-1] = events[0]
	}
	if len(session.PendingClassADownlinks) > 0 {
		downlink := session.PendingClassADownlinks[0]
		session.PendingClassADownlinks = session.PendingClassADownlinks[1:]
		downlink.ACK = decoded.Confirmed
		if len(session.PendingClassADownlinks) > 0 {
			downlink.FPending = true
		}
		dataRate := gatewayPacketDataRate(device, packet)
		if dataRate < 0 {
			dataRate = currentDeviceDataRate(session, device)
		}
		dataRate = regional.ClampDataRate(device.LocationConfig.Region, dataRate-session.RX1DataRateOffset)
		downlink.DataRate = dataRate
		session.FrameCounterDown++
		if gateway, ok := e.registry.Gateway(packet.GatewayID); ok {
			e.queueDataDownlinkLocked(gateway, device, session, downlink, packet.Frequency, dataRate, e.clock.Now()+session.ReceiveDelay)
		}
		downlinkEvent := e.newDownlinkEventLocked(
			contracts.DeviceDownlinkTransmitted,
			"Class A downlink scheduled through real gateway",
			device,
			session,
			downlink,
			"",
		)
		downlinkEvent.GatewayID = &packet.GatewayID
		downlinkEvent.ChannelFrequency = &packet.Frequency
		e.eventLog[len(e.eventLog)-1] = downlinkEvent
		events = append(events, downlinkEvent)
		events = append(events, e.applyDownlinkEffectsLocked(device, session, downlink, e.clock.Now()+session.ReceiveDelay)...)
		return events
	}
	if decoded.Confirmed {
		if ack := e.queueRealACKLocked(packet, device, session, decoded); ack {
			events = append(events, e.newProtocolEventLocked(
				contracts.UplinkACKReceived,
				"ACK scheduled for real gateway uplink",
				device.ID, packet.GatewayID, "", packetID, int64(frameCounter), 1, true, nil,
			))
		}
	}
	return events
}

func (e *Engine) realDeviceSessionLocked(devAddr uint32) (contracts.Device, *types.DeviceSession) {
	for _, device := range e.registry.ActiveDevices() {
		session := e.sessions[device.ID]
		if session == nil {
			continue
		}
		if parsed, err := hex.DecodeString(session.DeviceAddress); err == nil && len(parsed) == 4 {
			value := binary.BigEndian.Uint32(parsed)
			if value == devAddr {
				return device, session
			}
		}
	}
	return contracts.Device{}, nil
}

func (e *Engine) queueRealACKLocked(packet types.GatewayPacket, device contracts.Device, session *types.DeviceSession, uplink lorawan.Packet) bool {
	if len(session.NwkSKey) != 16 || len(session.AppSKey) != 16 {
		return false
	}
	devAddrBytes, err := hex.DecodeString(session.DeviceAddress)
	if err != nil || len(devAddrBytes) != 4 {
		return false
	}
	devAddr := binary.BigEndian.Uint32(devAddrBytes)
	payload, err := lorawanHandler.EncodeDataFrame(protocol.DataFrame{
		DevAddr:   devAddr,
		FCnt:      uint32(session.FrameCounterDown + 1),
		Direction: 1,
		ACK:       true,
		NwkSKey:   session.NwkSKey,
		AppSKey:   session.AppSKey,
	})
	if err != nil {
		return false
	}
	delay := durationMilliseconds(device.RX1Config.Delay)
	transmitAt, timestamp := gatewayReplyTiming(packet, delay)
	dataRate := gatewayPacketDataRate(device, packet)
	if dataRate < 0 {
		dataRate = currentDeviceDataRate(session, device)
	}
	spreadingFactor := spreadingFactorForDataRate(device.LocationConfig.Region, dataRate)
	bandwidth := gatewayPacketBandwidth(packet)
	e.gatewayPackets = append(e.gatewayPackets, types.GatewayPacket{
		GatewayID:             packet.GatewayID,
		Kind:                  types.GatewayPacketDownlink,
		Payload:               payload,
		Frequency:             packet.Frequency,
		Bandwidth:             bandwidth,
		SpreadingFactor:       spreadingFactor,
		Power:                 classBBeaconPower,
		DataRate:              fmt.Sprintf("SF%dBW%d", spreadingFactor, bandwidth/1000),
		TransmitAt:            transmitAt,
		ConcentratorTimestamp: timestamp,
	})
	session.FrameCounterDown++
	_ = uplink
	return true
}

func (e *Engine) queueRealJoinAcceptLocked(packet types.GatewayPacket, device contracts.Device, session *types.DeviceSession, appKey []byte) bool {
	devAddrBytes, err := hex.DecodeString(session.DeviceAddress)
	if err != nil || len(devAddrBytes) != 4 {
		return false
	}
	devAddr := binary.BigEndian.Uint32(devAddrBytes)
	payload, err := lorawanHandler.EncodeJoinAccept(protocol.JoinAccept{
		AppNonce:          0,
		NetID:             0,
		DevAddr:           devAddr,
		RX2DataRate:       intValue(device.RX2Config.DataRate),
		RX1DataRateOffset: intValue(device.RX1Config.DataRateOffset),
		RXDelay:           millisecondsAsWholeSeconds(device.RX1Config.Delay),
		AppKey:            appKey,
	})
	if err != nil {
		return false
	}
	delay := 5 * time.Second
	transmitAt, timestamp := gatewayReplyTiming(packet, delay)
	dataRate := gatewayPacketDataRate(device, packet)
	if dataRate < 0 {
		dataRate = currentDeviceDataRate(session, device)
	}
	bandwidth := gatewayPacketBandwidth(packet)
	e.gatewayPackets = append(e.gatewayPackets, types.GatewayPacket{
		GatewayID:             packet.GatewayID,
		Kind:                  types.GatewayPacketDownlink,
		Payload:               payload,
		Frequency:             packet.Frequency,
		Bandwidth:             bandwidth,
		SpreadingFactor:       spreadingFactorForDataRate(device.LocationConfig.Region, dataRate),
		Power:                 classBBeaconPower,
		DataRate:              fmt.Sprintf("SF%dBW%d", spreadingFactorForDataRate(device.LocationConfig.Region, dataRate), bandwidth/1000),
		TransmitAt:            transmitAt,
		ConcentratorTimestamp: timestamp,
	})
	return true
}

func sameEUI(value string, numeric uint64) bool {
	parsed, err := hex.DecodeString(value)
	if err != nil || len(parsed) != 8 {
		return false
	}
	return binary.BigEndian.Uint64(parsed) == numeric
}

func (e *Engine) processVirtualGatewayHeartbeatLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	gateway, ok := e.registry.Gateway(scheduled.GatewayID)
	if !ok || !gateway.Active || gateway.Type != contracts.Virtual || gateway.KeepAlive == nil {
		return nil
	}
	if runtime := e.gatewayRuntime[gateway.ID]; runtime != nil {
		runtime.State = contracts.Connected
		runtime.LastHeartbeat = scheduled.At
	}
	e.metrics.GatewayHeartbeats++
	event := e.newGatewayEventLocked(
		contracts.GatewayHeartbeat,
		"virtual gateway heartbeat received",
		gateway.ID,
		contracts.Connected,
		nil,
		"",
		"",
	)
	next := types.ScheduledEvent{
		ID: uuid.NewString(), At: scheduled.At + time.Duration(*gateway.KeepAlive)*time.Second,
		Type: contracts.GatewayHeartbeat, Message: "virtual gateway heartbeat scheduled",
		GatewayID: gateway.ID, Kind: types.ScheduledEventGatewayHeartbeat,
	}
	if err := e.scheduler.Schedule(next); err != nil {
		return []contracts.SimulationEvent{event, e.newEventLocked(contracts.PacketDropped, "virtual gateway heartbeat could not be scheduled", "", gateway.ID, next.ID)}
	}
	return []contracts.SimulationEvent{event}
}

func parseGatewayDataRate(value string) int {
	if !strings.HasPrefix(value, "SF") {
		return -1
	}
	parts := strings.SplitN(strings.TrimPrefix(value, "SF"), "BW", 2)
	if len(parts) != 2 {
		return -1
	}
	sf, err := strconv.Atoi(parts[0])
	if err != nil || sf < 7 || sf > 12 {
		return -1
	}
	return 12 - sf
}

func gatewayPacketDataRate(device contracts.Device, packet types.GatewayPacket) int {
	spreadingFactor := packet.SpreadingFactor
	if spreadingFactor == 0 {
		if parsed := parseGatewayDataRate(packet.DataRate); parsed >= 0 {
			spreadingFactor = 12 - parsed
		}
	}
	bandwidth := packet.Bandwidth
	if bandwidth == 0 {
		bandwidth = radioBandwidthHz
	}
	if dataRate, ok := regional.DataRateForModulation(device.LocationConfig.Region, spreadingFactor, bandwidth); ok {
		return dataRate
	}
	return -1
}

func gatewayPacketBandwidth(packet types.GatewayPacket) int64 {
	if packet.Bandwidth > 0 {
		return packet.Bandwidth
	}
	return radioBandwidthHz
}

func (e *Engine) publishGatewayError(gatewayID string, err error, timeout bool) {
	e.handleGatewayAdapterEvent(types.GatewayAdapterEvent{
		GatewayID: gatewayID,
		State:     contracts.Error,
		Error:     err.Error(),
		Timeout:   timeout,
	})
}

func (e *Engine) updateGatewayRuntimeLocked(
	gatewayID string,
	state contracts.GatewayConnectionState,
	errorMessage string,
	countAttempt bool,
) contracts.SimulationEvent {
	runtime := e.gatewayRuntime[gatewayID]
	if runtime == nil {
		runtime = &types.GatewayRuntime{}
		e.gatewayRuntime[gatewayID] = runtime
	}
	runtime.State = state
	runtime.LastNetworkError = errorMessage
	if countAttempt {
		runtime.ConnectionAttempts++
	}
	return e.newGatewayEventLocked(
		contracts.GatewayConnecting,
		"gateway connection starting",
		gatewayID,
		state,
		nil,
		"",
		errorMessage,
	)
}

func (e *Engine) newGatewayEventLocked(
	eventType contracts.SimulationEventType,
	message string,
	gatewayID string,
	state contracts.GatewayConnectionState,
	payloadSize *int64,
	packetID string,
	errorMessage string,
) contracts.SimulationEvent {
	event := e.newEventLocked(eventType, message, "", gatewayID, "")
	if packetID != "" {
		event.PacketID = &packetID
	}
	event.GatewayState = &state
	if payloadSize != nil {
		event.PayloadSize = payloadSize
	}
	if errorMessage != "" {
		event.Error = &errorMessage
	}
	e.eventLog[len(e.eventLog)-1] = event
	return event
}

func (e *Engine) stopGatewayAdapters() {
	e.mu.Lock()
	adapters := make([]types.GatewayAdapter, 0, len(e.gatewayAdapters))
	for gatewayID, adapter := range e.gatewayAdapters {
		adapters = append(adapters, adapter)
		delete(e.gatewayAdapters, gatewayID)
	}
	e.mu.Unlock()

	for _, adapter := range adapters {
		_ = adapter.Close()
	}
	if closer, ok := e.gatewayAdapterFactory.(types.GatewayAdapterFactoryCloser); ok {
		_ = closer.Close()
	}
}

func (e *Engine) SendGatewayPacket(ctx context.Context, gatewayID string, payload []byte) error {
	if ctx == nil {
		return fmt.Errorf("send gateway packet context cannot be nil")
	}
	return e.sendGatewayPacket(ctx, types.GatewayPacket{
		GatewayID: gatewayID,
		Payload:   append([]byte(nil), payload...),
	})
}

func (e *Engine) sendGatewayPacket(ctx context.Context, packet types.GatewayPacket) error {
	if ctx == nil {
		return fmt.Errorf("send gateway packet context cannot be nil")
	}

	e.mu.RLock()
	adapter, ok := e.gatewayAdapters[packet.GatewayID]
	gateway, gatewayExists := e.registry.Gateway(packet.GatewayID)
	e.mu.RUnlock()
	// Virtual gateways are externalized in phase 2 only for uplinks. Their
	// downlinks and Class-B beacons are still simulated in-process; the
	// Gateway Bridge receive path is introduced with the downlink phase.
	if gatewayExists && gateway.Type == contracts.Virtual && packet.Kind != types.GatewayPacketUplink {
		return nil
	}
	if !ok || adapter == nil {
		return fmt.Errorf("%w: %s", ErrGatewayTransportUnavailable, packet.GatewayID)
	}

	var err error
	if packet.Kind == types.GatewayPacketUplink {
		uplinkAdapter, ok := adapter.(types.GatewayUplinkAdapter)
		if !ok {
			return fmt.Errorf("gateway %s does not support uplink transport", packet.GatewayID)
		}
		err = uplinkAdapter.SendUplink(ctx, packet)
	} else {
		err = adapter.Send(ctx, packet)
	}
	if err != nil {
		e.mu.Lock()
		e.gatewayPackets = append(e.gatewayPackets, cloneGatewayPackets([]types.GatewayPacket{packet})...)
		e.mu.Unlock()
		e.publishGatewayError(packet.GatewayID, err, false)
		return err
	}

	e.mu.Lock()
	runtime := e.gatewayRuntime[packet.GatewayID]
	if runtime != nil {
		runtime.EgressPackets++
	}
	e.metrics.GatewayEgressPackets++
	payloadSize := int64(len(packet.Payload))
	event := e.newGatewayEventLocked(
		contracts.GatewayPacketEgress,
		"packet sent to gateway transport",
		packet.GatewayID,
		contracts.Connected,
		&payloadSize,
		"",
		"",
	)
	e.mu.Unlock()
	e.publish(event)
	return nil
}

func (e *Engine) drainGatewayPacketsLocked() []types.GatewayPacket {
	if len(e.gatewayPackets) == 0 {
		return nil
	}
	packets := append([]types.GatewayPacket(nil), e.gatewayPackets...)
	e.gatewayPackets = e.gatewayPackets[:0]
	return packets
}

func (e *Engine) takeGatewayPacketsLocked(gatewayID string) []types.GatewayPacket {
	if len(e.gatewayPackets) == 0 {
		return nil
	}
	packets := make([]types.GatewayPacket, 0)
	remaining := e.gatewayPackets[:0]
	for _, packet := range e.gatewayPackets {
		if packet.GatewayID == gatewayID {
			packets = append(packets, packet)
			continue
		}
		remaining = append(remaining, packet)
	}
	e.gatewayPackets = remaining
	return packets
}

func (e *Engine) dispatchGatewayPackets(packets []types.GatewayPacket) {
	for _, packet := range packets {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = e.sendGatewayPacket(ctx, packet)
		cancel()
	}
}

// Timed packet-forwarder responses can arrive before their radio transmission.
// Match the requested transmission time against the current uplink, not arrival.
func (e *Engine) externalACKMatchesUplink(device contracts.Device, session *types.DeviceSession, packet types.GatewayPacket) bool {
	pending := session.PendingUplink
	if pending == nil || packet.TransmitAt.IsZero() {
		return true
	}
	base := pending.SentAt
	if base.IsZero() {
		base = time.Now().Add(pending.TransmissionAt + session.LastAirtime - e.clock.Now())
	}
	if packet.TransmitAt.Before(base) {
		return false
	}
	if device.Class == contracts.ClassC {
		return true
	}
	rx1 := durationMilliseconds(device.RX1Config.Delay)
	rx2 := durationMilliseconds(device.RX2Config.Delay)
	if session.ReceiveDelay > 0 {
		if session.ReceiveDelay != rx1 {
			rx2 = session.ReceiveDelay + time.Second
		}
		rx1 = session.ReceiveDelay
	}
	delay := packet.TransmitAt.Sub(base)
	return (delay >= rx1 && delay <= rx1+windowDuration(device, contracts.RX1)) ||
		(delay >= rx2 && delay <= rx2+windowDuration(device, contracts.RX2))
}

// Replies use the originating RX counter so network and processing latency do
// not shift the device's receive window. uint32 addition handles counter wrap.
func gatewayReplyTiming(packet types.GatewayPacket, delay time.Duration) (time.Time, *uint32) {
	receivedAt := packet.ReceivedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now()
	}
	if packet.ConcentratorTimestamp == nil {
		return receivedAt.Add(delay), nil
	}
	timestamp := *packet.ConcentratorTimestamp + uint32(delay.Microseconds())
	return receivedAt.Add(delay), &timestamp
}

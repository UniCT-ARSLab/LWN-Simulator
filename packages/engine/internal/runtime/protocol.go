package runtime

import (
	"fmt"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/geometry"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

func (e *Engine) processScheduledEventLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	switch scheduled.Kind {
	case types.ScheduledEventJoinRequest:
		return e.processJoinRequestLocked(scheduled)
	case types.ScheduledEventDeviceUplink:
		return e.processUplinkLocked(scheduled)
	case types.ScheduledEventRX1Window, types.ScheduledEventRX2Window:
		return e.processReceiveWindowLocked(scheduled)
	case types.ScheduledEventRadioComplete:
		return e.processRadioTransmissionCompletedLocked(scheduled)
	case types.ScheduledEventGatewayHeartbeat:
		return e.processVirtualGatewayHeartbeatLocked(scheduled)
	case types.ScheduledEventGatewayBeacon:
		return e.processGatewayBeaconLocked(scheduled)
	case types.ScheduledEventClassBBeaconTimeout:
		return e.processClassBBeaconTimeoutLocked(scheduled)
	case types.ScheduledEventClassBPingSlot:
		return e.processClassBPingSlotLocked(scheduled)
	case types.ScheduledEventClassCDownlink:
		return e.processClassCDownlinkLocked(scheduled)
	default:
		return []contracts.SimulationEvent{
			e.newEventLocked(scheduled.Type, scheduled.Message, scheduled.DeviceID, scheduled.GatewayID, scheduled.ID),
		}
	}
}

func (e *Engine) processJoinRequestLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(scheduled.DeviceID)
	if !ok || !device.Active {
		return []contracts.SimulationEvent{e.newEventLocked(contracts.PacketDropped, "join request dropped because the device is not active", scheduled.DeviceID, "", scheduled.ID)}
	}
	if session.Joined {
		return e.scheduleUplinkLocked(device, scheduled.At)
	}

	packetID := uuid.NewString()
	session.DevNonce++
	e.metrics.JoinRequests++
	gateways := e.coveredGateways(device)
	session.PendingJoinRequest = &types.PendingJoinRequest{
		RequestedAt:    time.Now(),
		PacketID:       packetID,
		JoinEUI:        device.OOTAConfig.JoinEUI,
		GatewayIDs:     gatewayIDs(gateways),
		TransmissionAt: scheduled.At,
	}
	events := []contracts.SimulationEvent{e.newProtocolEventLocked(
		contracts.DeviceJoinRequestTransmitted,
		"OTAA join request transmitted",
		scheduled.DeviceID, "", scheduled.ID, packetID, -1, scheduled.Attempt, false, nil,
	)}
	return append(events, e.startRadioTransmissionLocked(device, scheduled, packetID, -1, false)...)
}

func (e *Engine) processUplinkLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(scheduled.DeviceID)
	if !ok || !device.Active {
		return []contracts.SimulationEvent{e.newEventLocked(contracts.PacketDropped, "uplink dropped because the device is not active", scheduled.DeviceID, "", scheduled.ID)}
	}
	if deviceRadioProtocol(device) != contracts.RawLoRa && !session.Joined {
		return []contracts.SimulationEvent{e.newEventLocked(contracts.PacketDropped, "uplink dropped because the device has not joined", scheduled.DeviceID, "", scheduled.ID)}
	}

	if pending := session.PendingUplink; pending != nil && pending.PacketID == scheduled.PacketID && pending.PayloadConfig != nil {
		device.PayloadConfig = *pending.PayloadConfig
	}
	attempt := scheduled.Attempt
	if attempt < 1 {
		attempt = 1
	}
	confirmed := deviceRadioProtocol(device) == contracts.LoRaWAN && device.PayloadConfig.MType == contracts.ConfirmedDataUp
	packetID := scheduled.PacketID
	frameCounter := scheduled.FrameCounter
	dataRate := session.CurrentDataRate
	fragmentCount := scheduled.FragmentCount
	if fragmentCount < 1 {
		fragmentCount = payloadFragmentCount(device, dataRate, session)
	}
	fragmentIndex := scheduled.FragmentIndex
	isFragmentContinuation := attempt == 1 && packetID != "" && session.PendingUplink != nil && session.PendingUplink.PacketID == packetID
	if attempt == 1 && !isFragmentContinuation && frameCounter > 0 {
		if !device.FrameConfig.DisableFrameCounterValidation && frameCounter <= session.FrameCounterUp {
			e.metrics.FrameCounterErrors++
			return []contracts.SimulationEvent{e.newProtocolEventLocked(
				contracts.FrameCounterRejected,
				"uplink rejected because its frame counter is not newer than the session counter",
				device.ID, "", scheduled.ID, scheduled.PacketID, frameCounter, attempt, confirmed, nil,
			)}
		}
		if frameCounter > session.FrameCounterUp {
			session.FrameCounterUp = frameCounter
		}
	}
	payloadSize := payloadSizeForFragment(device, dataRate, fragmentIndex, session)
	newUplink := attempt == 1 && !isFragmentContinuation
	if newUplink {
		if frameCounter == 0 {
			session.FrameCounterUp++
			frameCounter = session.FrameCounterUp
		}
		packetID = uuid.NewString()
		e.metrics.TotalUplinks++
		if confirmed || fragmentCount > 1 || session.UnconfirmedRepetitions > 1 {
			payloadConfig := device.PayloadConfig
			session.PendingUplink = &types.PendingUplink{
				PayloadConfig: &payloadConfig,
				ReservedFOpts: uplinkMACReserve(session),
				PacketID:      packetID, FrameCounter: frameCounter, Attempt: attempt,
				Confirmed: confirmed, TransmissionAt: scheduled.At, FPort: device.FrameConfig.FPort,
				PayloadSize: payloadSize, DataRate: dataRate,
				FragmentIndex: fragmentIndex, FragmentCount: fragmentCount,
			}
		}
	} else if attempt > 1 || isFragmentContinuation {
		pending := session.PendingUplink
		if pending == nil || pending.PacketID == "" || pending.FrameCounter != frameCounter {
			e.metrics.FrameCounterErrors++
			return []contracts.SimulationEvent{e.newProtocolEventLocked(
				contracts.FrameCounterRejected,
				"uplink retransmission rejected because its session is no longer pending",
				scheduled.DeviceID, "", scheduled.ID, packetID, frameCounter, attempt, confirmed, nil,
			)}
		}
		packetID = pending.PacketID
		frameCounter = pending.FrameCounter
		dataRate = pending.DataRate
		fragmentIndex = pending.FragmentIndex
		fragmentCount = pending.FragmentCount
		payloadSize = payloadSizeForFragment(device, dataRate, fragmentIndex, session)
		pending.Attempt = attempt
		if attempt > 1 {
			e.metrics.Retransmissions++
		}
	}

	events := []contracts.SimulationEvent{e.newProtocolEventLocked(
		contracts.DeviceUplinkTransmitted, uplinkMessage(attempt, confirmed),
		scheduled.DeviceID, "", scheduled.ID, packetID, frameCounter, attempt, confirmed, nil,
	)}
	e.decorateDeviceEventLocked(&events[0], session, device.LocationConfig.Region, device.FrameConfig.FPort, dataRate, payloadSize)
	if newUplink {
		e.setUplinkOutcomeLocked(&events[0], contracts.UplinkStarted)
	} else {
		e.eventLog[len(e.eventLog)-1] = events[0]
	}
	scheduled.FragmentIndex = fragmentIndex
	scheduled.FragmentCount = fragmentCount
	return append(events, e.startRadioTransmissionLocked(device, scheduled, packetID, frameCounter, confirmed)...)
}

func (e *Engine) processReceiveWindowLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(scheduled.DeviceID)
	if !ok {
		return nil
	}
	if scheduled.FrameCounter < 0 && session.PendingJoinRequest == nil {
		return nil
	}
	window := scheduled.Window
	if window == "" {
		if scheduled.Kind == types.ScheduledEventRX2Window {
			window = contracts.RX2
		} else {
			window = contracts.RX1
		}
	}
	events := []contracts.SimulationEvent{e.newProtocolEventLocked(
		windowEventType(window), fmt.Sprintf("%s receive window opened", window),
		scheduled.DeviceID, "", scheduled.ID, scheduled.PacketID,
		scheduled.FrameCounter, scheduled.Attempt, scheduled.Confirmed, &window,
	)}
	e.decorateWindowEventLocked(&events[0], scheduled, device, window)
	e.eventLog[len(e.eventLog)-1] = events[0]
	// A scheduled receive-window event represents the opening instant of the
	// window. The engine may be catching up after a large manual-clock jump, so
	// comparing against clock.Now() would incorrectly expire RX1/RX2 before the
	// event is processed. eventTimestampLocked() resolves to scheduled.At while
	// the event is being handled.
	windowExpired := scheduled.WindowDuration > 0 && e.eventTimestampLocked() > scheduled.At+scheduled.WindowDuration

	// Unconfirmed uplinks normally do not create receive-window events. When a
	// Class A downlink is queued, however, the next uplink opens the same RX1/
	// RX2 opportunity even if no ACK is required.
	if session.PendingUplink == nil && len(session.PendingClassADownlinks) > 0 {
		if !windowExpired {
			if downlinkEvents, delivered := e.transmitQueuedClassADownlinkLocked(device, session, scheduled); delivered {
				return append(events, downlinkEvents...)
			}
		}
		if window == contracts.RX1 {
			return append(events, e.scheduleWindowLocked(device, scheduled, contracts.RX2, scheduled.PacketID, scheduled.FrameCounter, false))
		}
		return events
	}

	if session.PendingJoinRequest != nil && session.PendingJoinRequest.PacketID == scheduled.PacketID {
		pending := session.PendingJoinRequest
		usesExternalTransport := e.gatewaysUseExternalTransportLocked(pending.GatewayIDs)
		if len(pending.GatewayIDs) > 0 && !usesExternalTransport && !windowExpired {
			session.Joined = true
			session.JoinEUI = pending.JoinEUI
			session.DeviceAddress = joinedDeviceAddress(session)
			deriveOTAAKeys(device, session)
			session.PendingJoinRequest = nil
			session.FrameCounterDown++
			e.metrics.JoinAccepts++
			events = append(events, e.newProtocolEventLocked(
				contracts.JoinAcceptReceived, fmt.Sprintf("OTAA join accept received for DevAddr %s", session.DeviceAddress), device.ID, "", scheduled.ID,
				pending.PacketID, -1, 1, false, &window,
			))
			return append(events, e.scheduleUplinkLocked(device, scheduled.At)...)
		}
		if window == contracts.RX1 {
			return append(events, e.scheduleWindowLocked(device, scheduled, contracts.RX2, pending.PacketID, -1, false))
		}
		return append(events, e.rescheduleJoinRequestLocked(device, scheduled.At)...)
	}

	pending := session.PendingUplink
	if pending == nil {
		return events
	}
	if pending.PacketID != scheduled.PacketID || pending.FrameCounter != scheduled.FrameCounter {
		e.metrics.FrameCounterErrors++
		return append(events, e.newProtocolEventLocked(
			contracts.FrameCounterRejected, "receive window does not match the pending frame counter",
			device.ID, "", scheduled.ID, scheduled.PacketID, scheduled.FrameCounter,
			scheduled.Attempt, scheduled.Confirmed, &window,
		))
	}
	continuousReceive := device.Class == contracts.ClassC
	usesExternalTransport := e.gatewaysUseExternalTransportLocked(pending.GatewayIDs)
	if len(pending.GatewayIDs) > 0 && !usesExternalTransport && (!windowExpired || continuousReceive) {
		session.FrameCounterDown++
		e.applyADRLocked(device, session)
		e.metrics.AcknowledgedUplinks++
		events = append(events, e.newProtocolEventLocked(
			contracts.UplinkACKReceived, "uplink acknowledgement received", device.ID,
			pending.GatewayIDs[0], scheduled.ID, pending.PacketID, pending.FrameCounter,
			pending.Attempt, true, &window,
		))
		e.decorateWindowEventLocked(&events[len(events)-1], scheduled, device, window)
		e.eventLog[len(e.eventLog)-1] = events[len(events)-1]
		if pending.FragmentIndex+1 < pending.FragmentCount {
			pending.FragmentIndex++
			pending.WirePayload = nil
			pending.Attempt = 1
			pending.GatewayIDs = nil
			next := types.ScheduledEvent{
				ID: uuid.NewString(), At: scheduled.At + scheduled.WindowDuration, Type: contracts.DeviceUplinkTransmitted,
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
		if downlinkEvents, delivered := e.transmitQueuedClassADownlinkLocked(device, session, scheduled); delivered {
			events = append(events, downlinkEvents...)
		}
		session.PendingUplink = nil
		e.metrics.SuccessfulUplinks++
		e.metrics.PacketSuccessRate = e.packetSuccessRate()
		events = append(events, e.newUplinkOutcomeEventLocked(contracts.MetricsUpdated, "simulation metrics updated", device.ID, "", scheduled.ID, pending.PacketID, contracts.UplinkSucceeded))
		return append(events, e.scheduleUplinkLocked(device, pending.TransmissionAt)...)
	}

	if window == contracts.RX1 {
		return append(events, e.scheduleWindowLocked(device, scheduled, contracts.RX2, pending.PacketID, pending.FrameCounter, true))
	}

	maxRetransmissions := intValue(device.FrameConfig.Retransmission)
	if pending.Attempt <= maxRetransmissions {
		nextAttempt := pending.Attempt + 1
		pending.Attempt = nextAttempt
		retryAt := scheduled.At + time.Duration(device.RX2Config.ACKTimeout)*time.Millisecond
		if retryAt <= scheduled.At {
			retryAt = scheduled.At + time.Second
		}
		next := types.ScheduledEvent{
			ID: uuid.NewString(), At: retryAt, Type: contracts.DeviceUplinkTransmitted,
			Message: "confirmed uplink retransmission scheduled", DeviceID: device.ID,
			Kind: types.ScheduledEventDeviceUplink, Attempt: nextAttempt,
			PacketID: pending.PacketID, FrameCounter: pending.FrameCounter, Confirmed: true,
		}
		if err := e.scheduler.Schedule(next); err != nil {
			session.PendingUplink = nil
			return append(events,
				e.finalizeDroppedUplinkLocked(device.ID, scheduled.ID, pending),
				e.newPacketEventLocked(contracts.MetricsUpdated, "simulation metrics updated", device.ID, "", scheduled.ID, pending.PacketID),
			)
		}
		return append(events, e.newProtocolEventLocked(
			contracts.UplinkRetryScheduled, "confirmed uplink retransmission scheduled", device.ID,
			"", next.ID, pending.PacketID, pending.FrameCounter, nextAttempt, true, nil,
		))
	}

	session.PendingUplink = nil
	events = append(events,
		e.finalizeDroppedUplinkLocked(device.ID, scheduled.ID, pending),
		e.newPacketEventLocked(contracts.MetricsUpdated, "simulation metrics updated", device.ID, "", scheduled.ID, pending.PacketID),
	)
	return append(events, e.scheduleUplinkLocked(device, scheduled.At)...)
}

func (e *Engine) gatewaysUseExternalTransportLocked(gatewayIDs []string) bool {
	for _, gatewayID := range gatewayIDs {
		if _, ok := e.gatewayAdapters[gatewayID]; ok {
			return true
		}
	}
	return false
}

func (e *Engine) applyADRLocked(device contracts.Device, session *types.DeviceSession) {
	if !device.AdvancedConfig.ADREnabled || session == nil || session.ExternalADR {
		return
	}
	for _, gateway := range e.coveredGateways(device) {
		if _, external := e.gatewayAdapters[gateway.ID]; external {
			return
		}
	}
	plan := regional.MustPlan(device.LocationConfig.Region)
	if session.LastSNR > 10 && session.CurrentDataRate < plan.MaximumDataRate {
		candidate := session.CurrentDataRate + 1
		if regional.SupportsUplinkDataRate(device.LocationConfig.Region, candidate) {
			session.CurrentDataRate = candidate
		}
	} else if session.LastSNR < -5 && session.CurrentDataRate > 0 {
		session.CurrentDataRate--
	}
	session.CurrentSpreadingFactor = spreadingFactorForDataRate(device.LocationConfig.Region, session.CurrentDataRate)
}

func (e *Engine) deviceSessionLocked(id string) (contracts.Device, *types.DeviceSession, bool) {
	device, ok := e.registry.Device(id)
	session, sessionOK := e.sessions[id]
	return device, session, ok && sessionOK
}

func (e *Engine) coveredGateways(device contracts.Device) []contracts.Gateway {
	deviceLatitude, deviceLongitude := locationCoordinates(device)
	gateways := make([]contracts.Gateway, 0)
	for _, gateway := range e.registry.ActiveVirtualGateways() {
		gatewayLatitude, gatewayLongitude := gatewayCoordinates(gateway)
		if geometry.DistanceMeters(deviceLatitude, deviceLongitude, gatewayLatitude, gatewayLongitude) <= float64(device.AdvancedConfig.AntennaRange) {
			gateways = append(gateways, gateway)
		}
	}
	return gateways
}

func gatewayIDs(gateways []contracts.Gateway) []string {
	ids := make([]string, 0, len(gateways))
	for _, gateway := range gateways {
		ids = append(ids, gateway.ID)
	}
	return ids
}

func (e *Engine) scheduleWindowLocked(device contracts.Device, source types.ScheduledEvent, window contracts.SimulationRxWindow, packetID string, frameCounter int64, confirmed bool) contracts.SimulationEvent {
	delay := durationMilliseconds(device.RX1Config.Delay)
	session := e.sessions[device.ID]
	if session != nil && session.ReceiveDelay > 0 {
		delay = session.ReceiveDelay
	}
	kind := types.ScheduledEventRX1Window
	dataRate := currentDeviceDataRate(e.sessions[device.ID], device)
	frequency := source.ChannelFrequency
	if window == contracts.RX1 && session != nil {
		for _, ch := range session.NetworkChannels {
			if ch.Frequency == frequency && ch.DownlinkFrequency > 0 {
				frequency = ch.DownlinkFrequency
				break
			}
		}
	}
	if window == contracts.RX2 {
		delay = durationMilliseconds(device.RX2Config.Delay)
		configuredRX1Delay := durationMilliseconds(device.RX1Config.Delay)
		if session != nil && session.ReceiveDelay > 0 && session.ReceiveDelay != configuredRX1Delay {
			delay = session.ReceiveDelay + time.Second
		}
		kind = types.ScheduledEventRX2Window
		dataRate = intValue(device.RX2Config.DataRate)
		frequency = int64(device.RX2Config.ChannelFrequency)
		if session != nil {
			dataRate = session.RX2DataRate
			frequency = session.RX2Frequency
		}
	} else {
		offset := intValue(device.RX1Config.DataRateOffset)
		if session != nil {
			offset = session.RX1DataRateOffset
		}
		dataRate = regional.ClampDataRate(device.LocationConfig.Region, dataRate-offset)
	}
	baseAt := source.At
	if source.WindowBaseAt > 0 {
		baseAt = source.WindowBaseAt
	}
	next := types.ScheduledEvent{
		ID: uuid.NewString(), At: baseAt + delay, Type: windowEventType(window),
		Message: fmt.Sprintf("%s receive window scheduled", window), DeviceID: device.ID,
		Kind: kind, Attempt: source.Attempt, PacketID: packetID, FrameCounter: frameCounter,
		Confirmed: confirmed, Window: window, WindowDuration: windowDuration(device, window),
		DataRate: dataRate, ChannelFrequency: frequency, WindowBaseAt: baseAt,
	}
	if err := e.scheduler.Schedule(next); err != nil {
		return e.newPacketEventLocked(contracts.PacketDropped, fmt.Sprintf("receive window scheduling failed: %v", err), device.ID, "", source.ID, packetID)
	}
	return e.newEventLocked(contracts.SchedulerEventScheduled, fmt.Sprintf("%s receive window scheduled", window), device.ID, "", next.ID)
}

func (e *Engine) scheduleUplinkLocked(device contracts.Device, base time.Duration) []contracts.SimulationEvent {
	return e.scheduleUplinkAtLocked(device, base+durationSecondsFloat(device.PayloadConfig.UplinkInterval))
}

func (e *Engine) scheduleUplinkAtLocked(device contracts.Device, at time.Duration) []contracts.SimulationEvent {
	next := types.ScheduledEvent{
		ID: uuid.NewString(), At: at,
		Type: contracts.DeviceUplinkTransmitted, Message: "device uplink scheduled", DeviceID: device.ID,
		Kind: types.ScheduledEventDeviceUplink, Attempt: 1,
	}
	if err := e.scheduler.Schedule(next); err != nil {
		return []contracts.SimulationEvent{e.newEventLocked(contracts.PacketDropped, fmt.Sprintf("uplink reschedule failed: %v", err), device.ID, "", next.ID)}
	}
	return []contracts.SimulationEvent{e.newEventLocked(contracts.DeviceUplinkScheduled, "device uplink scheduled", device.ID, "", next.ID)}
}

func (e *Engine) rescheduleJoinRequestLocked(device contracts.Device, base time.Duration) []contracts.SimulationEvent {
	next := types.ScheduledEvent{
		ID: uuid.NewString(), At: base + durationSecondsFloat(device.PayloadConfig.UplinkInterval),
		Type: contracts.DeviceJoinRequestTransmitted, Message: "device join request scheduled", DeviceID: device.ID,
		Kind: types.ScheduledEventJoinRequest, Attempt: 1,
	}
	if err := e.scheduler.Schedule(next); err != nil {
		return []contracts.SimulationEvent{e.newEventLocked(contracts.PacketDropped, fmt.Sprintf("join request reschedule failed: %v", err), device.ID, "", next.ID)}
	}
	return []contracts.SimulationEvent{e.newEventLocked(contracts.DeviceJoinRequestScheduled, "device join request scheduled", device.ID, "", next.ID)}
}

func (e *Engine) finalizeDroppedUplinkLocked(deviceID, scheduledID string, pending *types.PendingUplink) contracts.SimulationEvent {
	e.metrics.DroppedUplinks++
	e.metrics.PacketSuccessRate = e.packetSuccessRate()
	return e.newUplinkOutcomeEventLocked(contracts.PacketDropped, "confirmed uplink dropped after retry limit", deviceID, "", scheduledID, pending.PacketID, contracts.UplinkDropped)
}

func (e *Engine) newProtocolEventLocked(eventType contracts.SimulationEventType, message, deviceID, gatewayID, scheduledID, packetID string, frameCounter int64, attempt int, confirmed bool, window *contracts.SimulationRxWindow) contracts.SimulationEvent {
	event := e.newPacketEventLocked(eventType, message, deviceID, gatewayID, scheduledID, packetID)
	if frameCounter >= 0 {
		value := frameCounter
		event.FrameCounter = &value
	}
	if attempt > 0 {
		value := int32(attempt)
		event.RetryAttempt = &value
	}
	confirmedValue := confirmed
	event.Confirmed = &confirmedValue
	if window != nil {
		value := *window
		event.RxWindow = &value
	}
	e.eventLog[len(e.eventLog)-1] = event
	return event
}

func (e *Engine) decorateDeviceEventLocked(event *contracts.SimulationEvent, session *types.DeviceSession, region contracts.DeviceRegion, fPort, dataRate, payloadSize int) {
	port := fPort
	rate := dataRate
	size := int64(payloadSize)
	event.FPort = &port
	event.DataRate = &rate
	event.PayloadSize = &size
	if session != nil {
		event.SpreadingFactor = int32Pointer(spreadingFactorForDataRate(region, session.CurrentDataRate))
	}
}

func (e *Engine) decorateWindowEventLocked(event *contracts.SimulationEvent, scheduled types.ScheduledEvent, device contracts.Device, window contracts.SimulationRxWindow) {
	rate := scheduled.DataRate
	frequency := scheduled.ChannelFrequency
	if scheduled.Window != "" {
		event.DataRate = &rate
	}
	if frequency > 0 {
		event.ChannelFrequency = &frequency
	}
	_ = device
	_ = window
}

func windowDuration(device contracts.Device, window contracts.SimulationRxWindow) time.Duration {
	if window == contracts.RX2 {
		return durationMilliseconds(device.RX2Config.Duration)
	}
	return durationMilliseconds(device.RX1Config.Duration)
}

func (e *Engine) packetSuccessRate() float64 {
	if e.metrics.TotalUplinks == 0 {
		return 0
	}
	return float64(e.metrics.SuccessfulUplinks) / float64(e.metrics.TotalUplinks)
}

func windowEventType(window contracts.SimulationRxWindow) contracts.SimulationEventType {
	if window == contracts.RX2 {
		return contracts.RX2WindowOpened
	}
	return contracts.RX1WindowOpened
}

func uplinkMessage(attempt int, confirmed bool) string {
	if confirmed && attempt > 1 {
		return "confirmed device uplink retransmitted"
	}
	if !confirmed && attempt > 1 {
		return "unconfirmed device uplink repeated"
	}
	return "device uplink transmitted"
}

func joinAcceptReceiveDelay(value int) time.Duration {
	if value == 0 {
		value = 1
	}
	return time.Duration(value) * time.Second
}

func durationSecondsFloat(value float32) time.Duration {
	return time.Duration(float64(value) * float64(time.Second))
}

func durationMilliseconds(value *int) time.Duration {
	if value == nil || *value <= 0 {
		return 0
	}
	return time.Duration(*value) * time.Millisecond
}

func millisecondsAsWholeSeconds(value *int) int {
	if value == nil || *value <= 0 {
		return 0
	}
	return (*value + 999) / 1000
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

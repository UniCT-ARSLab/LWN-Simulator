package runtime

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/protocol"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func (e *Engine) newDownlinkEventLocked(
	eventType contracts.SimulationEventType,
	message string,
	device contracts.Device,
	session *types.DeviceSession,
	downlink types.Downlink,
	scheduledEventID string,
) contracts.SimulationEvent {
	event := e.newPacketEventLocked(eventType, message, device.ID, session.ClassBGatewayID, scheduledEventID, downlink.ID)
	payloadSize := int64(len(downlink.Payload))
	fPort := downlink.FPort
	dataRate := downlink.DataRate
	if dataRate < 0 {
		dataRate = session.CurrentDataRate
	}
	frequency := session.RX2Frequency
	event.PayloadSize = &payloadSize
	event.DataRate = &dataRate
	event.ChannelFrequency = &frequency
	if deviceRadioProtocol(device) == contracts.LoRaWAN {
		event.FPort = &fPort
		event.Confirmed = pointerTo(downlink.Confirmed)
		frameCounter := session.FrameCounterDown
		event.FrameCounter = &frameCounter
	}
	e.eventLog[len(e.eventLog)-1] = event
	return event
}

func (e *Engine) transmitQueuedClassADownlinkLocked(
	device contracts.Device,
	session *types.DeviceSession,
	scheduled types.ScheduledEvent,
) ([]contracts.SimulationEvent, bool) {
	if len(session.PendingClassADownlinks) == 0 {
		return nil, false
	}
	gateways := e.classBCoveredGateways(device)
	if len(gateways) == 0 {
		return nil, false
	}
	downlink := session.PendingClassADownlinks[0]
	session.PendingClassADownlinks = session.PendingClassADownlinks[1:]
	downlink.ACK = scheduled.Confirmed
	if len(session.PendingClassADownlinks) > 0 {
		downlink.FPending = true
	}
	session.FrameCounterDown++
	gateway := gateways[0]
	if gateway.Type == contracts.Real {
		e.queueDataDownlinkLocked(gateway, device, session, downlink, scheduled.ChannelFrequency, scheduled.DataRate, scheduled.At)
	}
	event := e.newDownlinkEventLocked(
		contracts.DeviceDownlinkTransmitted,
		"Class A downlink transmitted in receive window",
		device,
		session,
		downlink,
		scheduled.ID,
	)
	if scheduled.ChannelFrequency > 0 {
		frequency := scheduled.ChannelFrequency
		event.ChannelFrequency = &frequency
	}
	if scheduled.DataRate >= 0 {
		dataRate := scheduled.DataRate
		event.DataRate = &dataRate
	}
	if event.GatewayID == nil {
		event.GatewayID = &gateway.ID
		e.eventLog[len(e.eventLog)-1] = event
	}
	events := []contracts.SimulationEvent{event}
	events = append(events, e.applyDownlinkEffectsLocked(device, session, downlink, scheduled.At+scheduled.WindowDuration)...)
	return events, true
}

func (e *Engine) queueDataDownlinkLocked(
	gateway contracts.Gateway,
	device contracts.Device,
	session *types.DeviceSession,
	downlink types.Downlink,
	frequency int64,
	dataRate int,
	at time.Duration,
) {
	if frequency <= 0 {
		frequency = session.RX2Frequency
	}
	if dataRate < 0 {
		dataRate = currentDeviceDataRate(session, device)
	}
	if deviceRadioProtocol(device) == contracts.RawLoRa {
		if len(downlink.MACCommands) > 0 || len(downlink.Payload) == 0 {
			return
		}
		target := time.Now().Add(classBRealGatewayTimingLead)
		if at > 0 {
			target = time.Now().Add(time.Duration(float64(at-e.clock.Now()) / e.state.Speed))
			if target.Before(time.Now()) {
				target = time.Now().Add(classBRealGatewayTimingLead)
			}
		}
		e.gatewayPackets = append(e.gatewayPackets, types.GatewayPacket{
			GatewayID:       gateway.ID,
			DeviceID:        device.ID,
			Kind:            types.GatewayPacketDownlink,
			Protocol:        contracts.RawLoRa,
			Payload:         append([]byte(nil), downlink.Payload...),
			Frequency:       frequency,
			Bandwidth:       radioBandwidthHz,
			SpreadingFactor: spreadingFactorForDataRate(device.LocationConfig.Region, dataRate),
			Power:           int(radioTransmitPowerDBm),
			DataRate:        fmt.Sprintf("DR%d", dataRate),
			TransmitAt:      target,
		})
		return
	}
	devAddr, err := hex.DecodeString(session.DeviceAddress)
	if err != nil || len(devAddr) != 4 || len(session.NwkSKey) != 16 || len(session.AppSKey) != 16 {
		return
	}
	address := binary.BigEndian.Uint32(devAddr)
	var fPort *byte
	if len(downlink.Payload) > 0 {
		value := byte(downlink.FPort)
		fPort = &value
	}
	macCommands, err := lorawanHandler.EncodeMACCommands(downlink.MACCommands)
	if err != nil {
		return
	}
	payload, err := lorawanHandler.EncodeDataFrame(protocol.DataFrame{
		DevAddr:   address,
		FCnt:      uint32(session.FrameCounterDown),
		FPort:     fPort,
		Payload:   downlink.Payload,
		Confirmed: downlink.Confirmed,
		ACK:       downlink.ACK,
		FPending:  downlink.FPending,
		Direction: 1,
		NwkSKey:   session.NwkSKey,
		AppSKey:   session.AppSKey,
		FOpts:     macCommands,
	})
	if err != nil {
		return
	}
	target := time.Now().Add(classBRealGatewayTimingLead)
	if at > 0 {
		target = time.Now().Add(time.Duration(float64(at-e.clock.Now()) / e.state.Speed))
		if target.Before(time.Now()) {
			target = time.Now().Add(classBRealGatewayTimingLead)
		}
	}
	profile := regional.Profile(device.LocationConfig.Region, dataRate)
	spreadingFactor := profile.SpreadingFactor
	e.gatewayPackets = append(e.gatewayPackets, types.GatewayPacket{
		GatewayID:       gateway.ID,
		Kind:            GatewayPacketKindForDeviceClass(device.Class),
		Payload:         payload,
		Frequency:       frequency,
		Bandwidth:       profile.Bandwidth,
		SpreadingFactor: spreadingFactor,
		Power:           classBBeaconPower,
		DataRate:        fmt.Sprintf("SF%dBW%d", spreadingFactor, profile.Bandwidth/1000),
		TransmitAt:      target,
	})
}

func GatewayPacketKindForDeviceClass(class contracts.DeviceClass) types.GatewayPacketKind {
	if class == contracts.ClassB {
		return types.GatewayPacketClassBDownlink
	}
	return types.GatewayPacketDownlink
}

package runtime

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/geometry"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

const (
	classBRealGatewayInitialDelay = time.Second
	classBRealGatewayTimingLead   = 100 * time.Millisecond
	classBBeaconFrequency         = int64(869525000)
	classBBeaconPower             = 14
)

func (e *Engine) processGatewayBeaconLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	gateway, ok := e.registry.Gateway(scheduled.GatewayID)
	if !ok || !gateway.Active || (gateway.Type != contracts.Virtual && gateway.Type != contracts.Real) {
		return nil
	}
	if gateway.Type == contracts.Real && !e.realGatewayReadyLocked(gateway.ID) {
		return []contracts.SimulationEvent{
			e.scheduleClassBBeaconLocked(gateway.ID, scheduled.At+types.ClassBBeaconPeriod),
		}
	}

	e.metrics.ClassBBeacons++
	beaconMessage := "virtual gateway beacon transmitted"
	if gateway.Type == contracts.Real {
		beaconMessage = "real gateway beacon scheduled for transmission"
	}
	events := []contracts.SimulationEvent{e.newEventLocked(
		contracts.GatewayBeaconTransmitted,
		beaconMessage,
		"",
		gateway.ID,
		scheduled.ID,
	)}

	for _, device := range e.registry.ActiveDevices() {
		if device.Class != contracts.ClassB {
			continue
		}
		session := e.sessions[device.ID]
		if session == nil || !session.Joined || !deviceIsCoveredByGateway(device, gateway) {
			continue
		}

		selectedGateway := e.classBGatewayForDevice(device)
		if selectedGateway == nil || selectedGateway.ID != gateway.ID {
			continue
		}

		if session.ClassBNextPingEventID != "" {
			e.scheduler.Cancel(session.ClassBNextPingEventID)
		}
		if session.ClassBBeaconTimeoutID != "" {
			e.scheduler.Cancel(session.ClassBBeaconTimeoutID)
		}

		session.ClassBSynchronized = true
		session.ClassBGatewayID = gateway.ID
		session.LastBeaconAt = scheduled.At
		session.NextPingSlotAt = 0
		e.metrics.ClassBSynchronizations++
		events = append(events, e.newEventLocked(
			contracts.DeviceBeaconSynchronized,
			"Class B device synchronized with gateway beacon",
			device.ID,
			gateway.ID,
			scheduled.ID,
		))

		if pingEvent := e.scheduleClassBPingSlotLocked(device, session, scheduled.At+classBPingSlotPeriod(session)); pingEvent != nil {
			events = append(events, *pingEvent)
		}
		timeoutID := uuid.NewString()
		session.ClassBBeaconTimeoutID = timeoutID
		if err := e.scheduler.Schedule(types.ScheduledEvent{
			ID: timeoutID, At: scheduled.At + 2*types.ClassBBeaconPeriod,
			Type: contracts.DeviceBeaconMissed, Message: "Class B beacon synchronization timeout",
			DeviceID: device.ID, GatewayID: gateway.ID,
			Kind: types.ScheduledEventClassBBeaconTimeout, WindowBaseAt: scheduled.At,
		}); err != nil {
			session.ClassBBeaconTimeoutID = ""
		}
	}
	if gateway.Type == contracts.Real {
		e.queueClassBBeaconLocked(gateway)
	}

	events = append(events, e.scheduleClassBBeaconLocked(gateway.ID, scheduled.At+types.ClassBBeaconPeriod))
	return events
}

func (e *Engine) processClassBBeaconTimeoutLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(scheduled.DeviceID)
	if !ok || device.Class != contracts.ClassB || session == nil || !session.ClassBSynchronized {
		return nil
	}
	if session.LastBeaconAt != scheduled.WindowBaseAt {
		return nil
	}

	if session.ClassBNextPingEventID != "" {
		e.scheduler.Cancel(session.ClassBNextPingEventID)
	}
	session.ClassBNextPingEventID = ""
	session.NextPingSlotAt = 0
	session.ClassBSynchronized = false
	session.ClassBGatewayID = ""
	session.ClassBMissedBeacons++
	e.metrics.ClassBMissedBeacons++
	return []contracts.SimulationEvent{e.newEventLocked(
		contracts.DeviceBeaconMissed,
		"Class B device lost beacon synchronization",
		device.ID,
		scheduled.GatewayID,
		scheduled.ID,
	)}
}

func (e *Engine) processClassBPingSlotLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(scheduled.DeviceID)
	if !ok || !device.Active || device.Class != contracts.ClassB || session == nil ||
		!session.ClassBSynchronized || session.ClassBGatewayID != scheduled.GatewayID {
		return nil
	}
	gateway, gatewayExists := e.registry.Gateway(scheduled.GatewayID)
	if !gatewayExists || (gateway.Type == contracts.Real && !e.realGatewayReadyLocked(gateway.ID)) {
		if next := e.scheduleClassBPingSlotLocked(device, session, scheduled.At+classBPingSlotPeriod(session)); next != nil {
			return []contracts.SimulationEvent{*next}
		}
		return nil
	}

	session.ClassBNextPingEventID = ""
	session.NextPingSlotAt = scheduled.At
	e.metrics.ClassBPingSlots++
	events := []contracts.SimulationEvent{e.newEventLocked(
		contracts.ClassBPingSlotOpened,
		"Class B ping slot opened",
		device.ID,
		scheduled.GatewayID,
		scheduled.ID,
	)}

	if len(session.PendingClassBDownlinks) > 0 {
		downlink := session.PendingClassBDownlinks[0]
		session.PendingClassBDownlinks = session.PendingClassBDownlinks[1:]
		session.FrameCounterDown++
		e.metrics.ClassBDownlinks++
		if gateway.Type == contracts.Real {
			e.queueClassBDownlinkLocked(gateway, device, session, downlink)
		}
		events = append(events, e.newClassBDownlinkEventLocked(
			contracts.ClassBDownlinkTransmitted,
			"Class B downlink transmitted in ping slot",
			device,
			session,
			downlink,
			scheduled.ID,
		))
		events = append(events, e.applyDownlinkEffectsLocked(device, session, downlink, scheduled.At)...)
	}

	if next := e.scheduleClassBPingSlotLocked(device, session, scheduled.At+classBPingSlotPeriod(session)); next != nil {
		events = append(events, *next)
	}
	return events
}

func (e *Engine) scheduleClassBBeaconLocked(gatewayID string, at time.Duration) contracts.SimulationEvent {
	eventID := uuid.NewString()
	if err := e.scheduler.Schedule(types.ScheduledEvent{
		ID: eventID, At: at, Type: contracts.GatewayBeaconTransmitted,
		Message: "virtual gateway beacon scheduled", GatewayID: gatewayID,
		Kind: types.ScheduledEventGatewayBeacon,
	}); err != nil {
		return e.newEventLocked(contracts.SimulationFailed, fmt.Sprintf("schedule Class B beacon: %v", err), "", gatewayID, eventID)
	}
	return e.newEventLocked(contracts.GatewayBeaconScheduled, "virtual gateway beacon scheduled", "", gatewayID, eventID)
}

func (e *Engine) scheduleClassBPingSlotLocked(device contracts.Device, session *types.DeviceSession, at time.Duration) *contracts.SimulationEvent {
	eventID := uuid.NewString()
	if err := e.scheduler.Schedule(types.ScheduledEvent{
		ID: eventID, At: at, Type: contracts.ClassBPingSlotOpened,
		Message: "Class B ping slot scheduled", DeviceID: device.ID,
		GatewayID: session.ClassBGatewayID, Kind: types.ScheduledEventClassBPingSlot,
	}); err != nil {
		return nil
	}
	session.ClassBNextPingEventID = eventID
	session.NextPingSlotAt = at
	event := e.newEventLocked(contracts.ClassBPingSlotScheduled, "Class B ping slot scheduled", device.ID, session.ClassBGatewayID, eventID)
	return &event
}

func (e *Engine) classBGatewayForDevice(device contracts.Device) *contracts.Gateway {
	gateways := e.classBCoveredGateways(device)
	if len(gateways) == 0 {
		return nil
	}

	deviceLatitude, deviceLongitude := locationCoordinates(device)
	selected := gateways[0]
	selectedDistance := math.Inf(1)
	for _, gateway := range gateways {
		gatewayLatitude, gatewayLongitude := gatewayCoordinates(gateway)
		distance := geometry.DistanceMeters(deviceLatitude, deviceLongitude, gatewayLatitude, gatewayLongitude)
		// Registry.Gateways is sorted by ID, so a strict comparison keeps the
		// result deterministic when two gateways are equally distant.
		if distance < selectedDistance {
			selected = gateway
			selectedDistance = distance
		}
	}
	return &selected
}

func (e *Engine) classBCoveredGateways(device contracts.Device) []contracts.Gateway {
	deviceLatitude, deviceLongitude := locationCoordinates(device)
	gateways := make([]contracts.Gateway, 0)
	for _, gateway := range e.registry.ActiveGateways() {
		if gateway.Type == contracts.Real && !e.realGatewayReadyLocked(gateway.ID) {
			continue
		}
		gatewayLatitude, gatewayLongitude := gatewayCoordinates(gateway)
		if geometry.DistanceMeters(deviceLatitude, deviceLongitude, gatewayLatitude, gatewayLongitude) <= float64(device.AdvancedConfig.AntennaRange) {
			gateways = append(gateways, gateway)
		}
	}
	return gateways
}

func (e *Engine) realGatewayReadyLocked(gatewayID string) bool {
	runtime, ok := e.gatewayRuntime[gatewayID]
	if !ok || runtime.State != contracts.Connected {
		return false
	}
	_, ok = e.gatewayAdapters[gatewayID]
	return ok
}

func (e *Engine) rescheduleRealGatewayBeaconLocked(gatewayID string) *contracts.SimulationEvent {
	gateway, ok := e.registry.Gateway(gatewayID)
	if !ok || gateway.Type != contracts.Real {
		return nil
	}
	for _, scheduled := range e.scheduler.Events() {
		if scheduled.Kind == types.ScheduledEventGatewayBeacon && scheduled.GatewayID == gatewayID {
			e.scheduler.Cancel(scheduled.ID)
		}
	}
	event := e.scheduleClassBBeaconLocked(gatewayID, e.clock.Now()+classBRealGatewayInitialDelay)
	return &event
}

func deviceIsCoveredByGateway(device contracts.Device, gateway contracts.Gateway) bool {
	deviceLatitude, deviceLongitude := locationCoordinates(device)
	gatewayLatitude, gatewayLongitude := gatewayCoordinates(gateway)
	return geometry.DistanceMeters(deviceLatitude, deviceLongitude, gatewayLatitude, gatewayLongitude) <= float64(device.AdvancedConfig.AntennaRange)
}

func (e *Engine) newClassBDownlinkEventLocked(
	eventType contracts.SimulationEventType,
	message string,
	device contracts.Device,
	session *types.DeviceSession,
	downlink types.ClassBDownlink,
	scheduledEventID string,
) contracts.SimulationEvent {
	event := e.newPacketEventLocked(eventType, message, device.ID, session.ClassBGatewayID, scheduledEventID, downlink.ID)
	payloadSize := int64(len(downlink.Payload))
	fPort := downlink.FPort
	dataRate := downlink.DataRate
	if dataRate < 0 {
		dataRate = session.PingSlotDataRate
	}
	frequency := session.PingSlotFrequency
	event.PayloadSize = &payloadSize
	event.FPort = &fPort
	event.DataRate = &dataRate
	event.ChannelFrequency = &frequency
	event.Confirmed = pointerTo(downlink.Confirmed)
	frameCounter := session.FrameCounterDown
	event.FrameCounter = &frameCounter
	e.eventLog[len(e.eventLog)-1] = event
	return event
}

func (e *Engine) queueClassBBeaconLocked(gateway contracts.Gateway) {
	target := time.Now().Add(classBRealGatewayTimingLead)
	frequency := classBBeaconFrequency
	dataRate := 3
	for _, device := range e.registry.ActiveDevices() {
		if device.Class == contracts.ClassB && deviceIsCoveredByGateway(device, gateway) {
			plan := regional.MustPlan(device.LocationConfig.Region)
			frequency = plan.BeaconFrequency
			dataRate = plan.BeaconDataRate
			break
		}
	}
	profile := regional.Profile(contracts.EU868, dataRate)
	// Beacon DR8 is a 500 kHz profile in US/AU. Resolve it against the first
	// covered device's region when one is available.
	for _, device := range e.registry.ActiveDevices() {
		if device.Class == contracts.ClassB && deviceIsCoveredByGateway(device, gateway) {
			profile = regional.Profile(device.LocationConfig.Region, dataRate)
			break
		}
	}
	e.gatewayPackets = append(e.gatewayPackets, types.GatewayPacket{
		GatewayID:       gateway.ID,
		Kind:            types.GatewayPacketClassBBeacon,
		Payload:         classBBeaconPayload(target, gateway),
		Frequency:       frequency,
		Bandwidth:       profile.Bandwidth,
		SpreadingFactor: profile.SpreadingFactor,
		Power:           classBBeaconPower,
		DataRate:        fmt.Sprintf("SF%dBW%d", profile.SpreadingFactor, profile.Bandwidth/1000),
		TransmitAt:      target,
	})
}

func (e *Engine) queueClassBDownlinkLocked(
	gateway contracts.Gateway,
	device contracts.Device,
	session *types.DeviceSession,
	downlink types.ClassBDownlink,
) {
	dataRate := downlink.DataRate
	if dataRate < 0 {
		dataRate = session.PingSlotDataRate
	}
	e.queueDataDownlinkLocked(
		gateway,
		device,
		session,
		downlink,
		session.PingSlotFrequency,
		dataRate,
		0,
	)
}

func classBBeaconPayload(transmitAt time.Time, gateway contracts.Gateway) []byte {
	payload := make([]byte, 17)
	gpsEpoch := time.Date(1980, time.January, 6, 0, 0, 0, 0, time.UTC)
	seconds := uint32(transmitAt.UTC().Sub(gpsEpoch).Seconds())
	binary.LittleEndian.PutUint32(payload[2:6], seconds)
	binary.LittleEndian.PutUint16(payload[6:8], classBBeaconCRC(payload[:6]))
	payload[8] = 0
	putBeaconCoordinate(payload[9:12], float64(pointerValue(gateway.Latitude)), -90, 90)
	putBeaconCoordinate(payload[12:15], float64(pointerValue(gateway.Longitude)), -180, 180)
	binary.LittleEndian.PutUint16(payload[15:17], classBBeaconCRC(payload[8:15]))
	return payload
}

func putBeaconCoordinate(destination []byte, value, minimum, maximum float64) {
	normalized := (value - minimum) / (maximum - minimum)
	if normalized < 0 {
		normalized = 0
	}
	if normalized > 1 {
		normalized = 1
	}
	encoded := uint32(normalized * float64(1<<24-1))
	destination[0] = byte(encoded)
	destination[1] = byte(encoded >> 8)
	destination[2] = byte(encoded >> 16)
}

func classBBeaconCRC(value []byte) uint16 {
	var crc uint16
	for _, current := range value {
		crc ^= uint16(current) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func pointerTo[T any](value T) *T {
	return &value
}

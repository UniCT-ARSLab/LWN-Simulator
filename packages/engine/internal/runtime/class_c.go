package runtime

import (
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

func (e *Engine) scheduleClassCDownlinkLocked(
	device contracts.Device,
	session *types.DeviceSession,
	at time.Duration,
) *contracts.SimulationEvent {
	if len(session.PendingClassCDownlinks) == 0 {
		return nil
	}
	eventID := uuid.NewString()
	if err := e.scheduler.Schedule(types.ScheduledEvent{
		ID:       eventID,
		At:       at,
		Type:     contracts.DeviceDownlinkTransmitted,
		Message:  "Class C downlink scheduled",
		DeviceID: device.ID,
		Kind:     types.ScheduledEventClassCDownlink,
	}); err != nil {
		return nil
	}
	session.ClassCNextDownlinkEventID = eventID
	event := e.newEventLocked(contracts.DeviceDownlinkScheduled, "Class C downlink scheduled", device.ID, "", eventID)
	return &event
}

func (e *Engine) processClassCDownlinkLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(scheduled.DeviceID)
	if !ok || !device.Active || device.Class != contracts.ClassC || session == nil {
		return nil
	}
	session.ClassCNextDownlinkEventID = ""
	if len(session.PendingClassCDownlinks) == 0 {
		return nil
	}
	gateways := e.classBCoveredGateways(device)
	if len(gateways) == 0 {
		next := e.scheduleClassCDownlinkLocked(device, session, scheduled.At+time.Second)
		if next == nil {
			return []contracts.SimulationEvent{e.newEventLocked(contracts.DeviceDownlinkDropped, "Class C downlink has no available gateway", device.ID, "", scheduled.ID)}
		}
		return []contracts.SimulationEvent{*next}
	}
	downlink := session.PendingClassCDownlinks[0]
	session.PendingClassCDownlinks = session.PendingClassCDownlinks[1:]
	session.FrameCounterDown++
	gateway := gateways[0]
	if gateway.Type == contracts.Real {
		e.queueDataDownlinkLocked(gateway, device, session, downlink, session.RX2Frequency, downlink.DataRate, scheduled.At)
	}
	event := e.newDownlinkEventLocked(contracts.DeviceDownlinkTransmitted, "Class C downlink transmitted", device, session, downlink, scheduled.ID)
	event.GatewayID = &gateway.ID
	e.eventLog[len(e.eventLog)-1] = event
	events := []contracts.SimulationEvent{event}
	events = append(events, e.applyDownlinkEffectsLocked(device, session, downlink, scheduled.At)...)
	if len(session.PendingClassCDownlinks) > 0 {
		if next := e.scheduleClassCDownlinkLocked(device, session, scheduled.At); next != nil {
			events = append(events, *next)
		}
	}
	return events
}

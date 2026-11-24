package runtime

import (
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestClassADownlinkUsesNextReceiveWindow(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000031")
	configureABP(&device)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c31")
	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	if _, err := engine.QueueDownlink(types.Downlink{DeviceID: device.ID, Payload: []byte("class-a")}); err != nil {
		t.Fatalf("queue Class A downlink: %v", err)
	}
	if err := clock.Advance(2 * time.Second); err != nil {
		t.Fatalf("advance to uplink: %v", err)
	}
	transmitted := waitForEventTypeWithLog(t, engine, contracts.DeviceDownlinkTransmitted)
	if transmitted.DeviceID == nil || *transmitted.DeviceID != device.ID {
		t.Fatalf("downlink references the wrong device: %+v", transmitted.DeviceID)
	}
	if engine.Snapshot().Devices[0].FrameCounterDown != 1 {
		t.Fatalf("Class A downlink did not advance the downlink counter: %+v", engine.Snapshot().Devices[0])
	}
}

func TestClassCDownlinkIsDeliveredWithoutAnUplinkWindow(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000032")
	configureABP(&device)
	device.Class = contracts.ClassC
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c32")
	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	if _, err := engine.QueueDownlink(types.Downlink{DeviceID: device.ID, Payload: []byte("class-c")}); err != nil {
		t.Fatalf("queue Class C downlink: %v", err)
	}
	transmitted := waitForEventType(t, engine.Events(), contracts.DeviceDownlinkTransmitted)
	if transmitted.DeviceID == nil || *transmitted.DeviceID != device.ID {
		t.Fatalf("downlink references the wrong device: %+v", transmitted.DeviceID)
	}
	if engine.Snapshot().Devices[0].FrameCounterDown != 1 {
		t.Fatalf("Class C downlink did not advance the downlink counter: %+v", engine.Snapshot().Devices[0])
	}
}

func waitForEventTypeWithLog(t *testing.T, engine *Engine, expected contracts.SimulationEventType) contracts.SimulationEvent {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case event := <-engine.Events():
			if event.Type == expected {
				return event
			}
		case <-deadline:
			t.Logf("event log: %+v", engine.EventLog())
			t.Fatalf("timed out waiting for event %q", expected)
		}
	}
}

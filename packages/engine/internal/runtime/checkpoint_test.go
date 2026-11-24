package runtime

import (
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestCheckpointRestoresSessionAndPendingSchedule(t *testing.T) {
	firstClock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.PayloadConfig.UplinkInterval = 10
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")

	first, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{Clock: firstClock})
	if err != nil {
		t.Fatalf("create first engine: %v", err)
	}
	startEngine(t, first)
	waitForEventType(t, first.Events(), contracts.DeviceUplinkTransmitted)
	if err := firstClock.Advance(time.Second); err != nil {
		t.Fatalf("advance first engine: %v", err)
	}
	waitForEventType(t, first.Events(), contracts.GatewayPacketReceived)
	checkpoint := first.Checkpoint()
	stopEngine(t, first)

	secondClock := types.NewManualClock(time.Second)
	second, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{Clock: secondClock, Checkpoint: &checkpoint})
	if err != nil {
		t.Fatalf("restore engine: %v", err)
	}
	startEngine(t, second)
	defer stopEngine(t, second)

	snapshot := second.Snapshot()
	if snapshot.Devices[0].FrameCounterUp != 1 || snapshot.Metrics.SuccessfulUplinks != 1 {
		t.Fatalf("checkpoint did not restore runtime state: %+v", snapshot)
	}
	if err := secondClock.Advance(9 * time.Second); err != nil {
		t.Fatalf("advance restored engine: %v", err)
	}
	waitForEventType(t, second.Events(), contracts.DeviceUplinkTransmitted)
}

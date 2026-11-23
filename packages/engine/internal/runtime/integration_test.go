package runtime

import (
	"context"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

// TestIntegrationMixedFleetLongRun exercises the interactions between the
// scheduler, protocol state machines, radio model, coverage registry and
// metrics over a longer deterministic run than the focused unit tests.
func TestIntegrationMixedFleetLongRun(t *testing.T) {
	clock := types.NewManualClock(0)

	abp := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&abp)
	abp.Name = "Catania Temperature"
	abp.DevEUI = "70B3D57ED0000001"
	abp.PayloadConfig.UplinkInterval = 5
	abp.AdvancedConfig.AntennaRange = 1_500

	otaa := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000002")
	otaa.Name = "Catania Air Quality"
	otaa.DevEUI = "70B3D57ED0000002"
	otaa.OOTAConfig.JoinEUI = "70B3D57ED0000002"
	otaa.PayloadConfig.UplinkInterval = 7
	otaa.AdvancedConfig.AntennaRange = 1_500

	outside := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000003")
	configureABP(&outside)
	outside.Name = "Outside Coverage"
	outside.DevEUI = "70B3D57ED0000003"
	outside.LocationConfig.Latitude = float32Ptr(37.52)
	outside.AdvancedConfig.AntennaRange = 500
	outside.PayloadConfig.UplinkInterval = 5

	firstGateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")
	secondGateway := validGateway("7f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c02")
	secondGateway.Name = "Catania Lab Gateway 2"
	secondGateway.GatewayEUI = "A840410001000102"

	engine, err := New(
		contracts.SimulationConfig{Speed: 1, Seed: int64Ptr(42)},
		[]contracts.Device{abp, otaa, outside},
		[]contracts.Gateway{firstGateway, secondGateway},
		types.Options{Clock: clock, EventBuffer: 4096},
	)
	if err != nil {
		t.Fatalf("create mixed-fleet engine: %v", err)
	}
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("start mixed-fleet engine: %v", err)
	}
	defer stopEngine(t, engine)

	// Advance in one-second steps so the test exercises periodic scheduling,
	// receive windows and radio completions rather than only the initial tick.
	for second := int64(1); second <= 40; second++ {
		if err := clock.Advance(time.Second); err != nil {
			t.Fatalf("advance clock to %ds: %v", second, err)
		}
		waitForSimulationTime(t, engine, second*1000)
	}
	waitForDueEventsToDrain(t, engine, clock)
	if err := engine.Pause(); err != nil {
		t.Fatalf("pause mixed-fleet engine before assertions: %v", err)
	}

	checkpoint := engine.Checkpoint()
	snapshot := engine.Snapshot()
	if snapshot.State.Status != contracts.SimulationStatusPaused {
		t.Fatalf("expected paused simulation, got %q", snapshot.State.Status)
	}
	if !snapshot.Devices[0].Joined || !snapshot.Devices[1].Joined {
		t.Fatalf("ABP and OTAA devices should be joined: %+v", snapshot.Devices)
	}
	if snapshot.Metrics.JoinRequests != 1 || snapshot.Metrics.JoinAccepts != 1 {
		t.Fatalf("unexpected OTAA metrics: %+v", snapshot.Metrics)
	}
	if snapshot.Metrics.TotalUplinks < 8 || snapshot.Metrics.SuccessfulUplinks < 8 {
		t.Fatalf("expected sustained successful traffic: %+v", snapshot.Metrics)
	}
	if snapshot.Metrics.DroppedUplinks == 0 || snapshot.Metrics.RadioPacketLosses == 0 {
		t.Fatalf("outside-coverage traffic should be dropped: %+v", snapshot.Metrics)
	}
	if snapshot.Metrics.TotalPacketsReceived <= snapshot.Metrics.SuccessfulUplinks {
		t.Fatalf("both gateways should receive covered packets: %+v", snapshot.Metrics)
	}
	if snapshot.Metrics.PacketSuccessRate <= 0 || snapshot.Metrics.PacketSuccessRate > 1 {
		t.Fatalf("invalid packet success rate: %f", snapshot.Metrics.PacketSuccessRate)
	}

	log := checkpoint.EventLog
	if len(log) == 0 || int64(len(log)) != checkpoint.State.EventSequence {
		t.Fatalf("event log and sequence diverged: events=%d sequence=%d", len(log), checkpoint.State.EventSequence)
	}
	previousTimestamp := int64(0)
	for index, event := range log {
		expectedSequence := int64(index + 1)
		if event.Sequence != expectedSequence {
			t.Fatalf("event sequence is not contiguous at index %d: got %d want %d", index, event.Sequence, expectedSequence)
		}
		if event.TimestampMilliseconds < previousTimestamp {
			previous := log[index-1]
			t.Logf("timestamp ordering failure: previous sequence=%d type=%s timestamp=%d, current sequence=%d type=%s timestamp=%d", previous.Sequence, previous.Type, previous.TimestampMilliseconds, event.Sequence, event.Type, event.TimestampMilliseconds)
			t.Fatalf("event timestamps moved backwards at index %d: previous=%d current=%d", index, previousTimestamp, event.TimestampMilliseconds)
		}
		previousTimestamp = event.TimestampMilliseconds
	}
}

func waitForSimulationTime(t *testing.T, engine *Engine, milliseconds int64) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if engine.Snapshot().State.ElapsedMilliseconds >= milliseconds {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("simulation did not reach %dms", milliseconds)
}

func waitForDueEventsToDrain(t *testing.T, engine *Engine, clock *types.ManualClock) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		now := clock.Now()
		engine.mu.RLock()
		next, hasNext := engine.scheduler.Peek()
		engine.mu.RUnlock()
		if !hasNext || next.At > now {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("scheduler did not drain events due by %s", clock.Now())
}

package runtime

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestInitialTrafficIsDeterministicallyDistributed(t *testing.T) {
	devices := make([]contracts.Device, 200)
	for index := range devices {
		device := validDevice(fmt.Sprintf("00000000-0000-4000-8000-%012x", index+1))
		device.Name = fmt.Sprintf("Dense device %d", index+1)
		device.DevEUI = fmt.Sprintf("70B3D57E%08X", index+1)
		device.OOTAConfig.JoinEUI = fmt.Sprintf("70B3D57F%08X", index+1)
		device.PayloadConfig.UplinkInterval = 60
		devices[index] = device
	}

	collect := func() map[string]time.Duration {
		clock := types.NewManualClock(0)
		runtime, err := New(
			contracts.SimulationConfig{Speed: 1, Seed: int64Ptr(42)},
			devices,
			nil,
			types.Options{Clock: clock, EventBuffer: 1024},
		)
		if err != nil {
			t.Fatalf("create dense engine: %v", err)
		}
		startEngine(t, runtime)
		t.Cleanup(func() { stopEngine(t, runtime) })
		delays := make(map[string]time.Duration, len(devices))
		for _, event := range runtime.Checkpoint().Scheduled {
			if event.Kind == types.ScheduledEventDeviceUplink || event.Kind == types.ScheduledEventJoinRequest {
				delays[event.DeviceID] = event.At
			}
		}
		return delays
	}

	first := collect()
	second := collect()
	if !reflect.DeepEqual(first, second) {
		t.Fatal("the same seed did not reproduce initial device phases")
	}
	if len(first) != len(devices) {
		t.Fatalf("expected %d initial transmissions, got %d", len(devices), len(first))
	}
	distinct := make(map[time.Duration]struct{}, len(first))
	zeroDelays := 0
	for _, delay := range first {
		if delay < 0 || delay >= 60*time.Second {
			t.Fatalf("initial delay outside reporting interval: %s", delay)
		}
		if delay == 0 {
			zeroDelays++
		}
		distinct[delay] = struct{}{}
	}
	if zeroDelays > 1 || len(distinct) < len(devices)*9/10 {
		t.Fatalf("initial traffic was not sufficiently distributed: zero=%d distinct=%d", zeroDelays, len(distinct))
	}
}

func TestEngineLifecycleAndScheduledEvents(t *testing.T) {
	clock := types.NewManualClock(0)
	events := make(chan contracts.SimulationEvent, 32)
	engine, err := New(
		contracts.SimulationConfig{Speed: 1, Seed: int64Ptr(7)},
		[]contracts.Device{validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")},
		[]contracts.Gateway{validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")},
		types.Options{
			Clock:       clock,
			EventSink:   func(event contracts.SimulationEvent) { events <- event },
			EventBuffer: 64,
		},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("start engine: %v", err)
	}

	if got := engine.Snapshot().State.Status; got != contracts.SimulationStatusRunning {
		t.Fatalf("expected running state, got %q", got)
	}
	for range 3 {
		readEvent(t, events)
	}

	if err := engine.Pause(); err != nil {
		t.Fatalf("pause engine: %v", err)
	}
	if got := engine.Snapshot().State.Status; got != contracts.SimulationStatusPaused {
		t.Fatalf("expected paused state, got %q", got)
	}
	readEvent(t, events)

	if err := engine.Resume(); err != nil {
		t.Fatalf("resume engine: %v", err)
	}
	readEvent(t, events)

	if err := engine.Schedule(types.ScheduledEvent{
		ID:      "event-1",
		At:      100 * time.Millisecond,
		Type:    contracts.DeviceRegistered,
		Message: "scheduled device event",
	}); err != nil {
		t.Fatalf("schedule event: %v", err)
	}
	readEvent(t, events)

	if err := clock.Advance(100 * time.Millisecond); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	for {
		event := readEvent(t, events)
		if event.Message == "scheduled device event" {
			if event.ScheduledEventID == nil || *event.ScheduledEventID != "event-1" {
				t.Fatalf("expected scheduled event reference, got %+v", event.ScheduledEventID)
			}
			break
		}
	}

	stopContext, stopCancel := context.WithTimeout(context.Background(), time.Second)
	defer stopCancel()
	if err := engine.Stop(stopContext); err != nil {
		t.Fatalf("stop engine: %v", err)
	}
	readEvent(t, events)

	if got := engine.Snapshot().State.Status; got != contracts.SimulationStatusStopped {
		t.Fatalf("expected stopped state, got %q", got)
	}
	if len(engine.EventLog()) != int(engine.Snapshot().State.EventSequence) {
		t.Fatal("event log and state sequence are inconsistent")
	}
}

func TestEngineRejectsInvalidLifecycleTransitions(t *testing.T) {
	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		nil,
		nil,
		types.Options{Clock: types.NewManualClock(0)},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}

	if err := engine.Pause(); err == nil {
		t.Fatal("expected pause before start to fail")
	}
	if err := engine.Resume(); err == nil {
		t.Fatal("expected resume before start to fail")
	}
}

func readEvent(t *testing.T, events <-chan contracts.SimulationEvent) contracts.SimulationEvent {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for engine event")
		return contracts.SimulationEvent{}
	}
}

package runtime

import (
	"context"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestPhaseTwoTransmitsPeriodicUplinksToCoveredVirtualGateways(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.PayloadConfig.UplinkInterval = 10
	device.AdvancedConfig.AntennaRange = 1_000
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device},
		[]contracts.Gateway{gateway},
		types.Options{Clock: clock, EventBuffer: 128},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	first := advanceAndWait(t, clock, time.Second, engine.Events(), contracts.GatewayPacketReceived)
	if first.DeviceID == nil || *first.DeviceID != device.ID {
		t.Fatalf("expected packet to reference device %s, got %+v", device.ID, first.DeviceID)
	}
	if first.GatewayID == nil || *first.GatewayID != gateway.ID {
		t.Fatalf("expected packet to reference gateway %s, got %+v", gateway.ID, first.GatewayID)
	}
	if first.PacketID == nil || *first.PacketID == "" {
		t.Fatal("expected received packet to have an id")
	}

	if err := clock.Advance(9 * time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	if err := clock.Advance(time.Second); err != nil {
		t.Fatalf("advance to radio completion: %v", err)
	}
	second := waitForEventType(t, engine.Events(), contracts.GatewayPacketReceived)
	if second.TimestampMilliseconds < 10_000 {
		t.Fatalf("expected second uplink at or after 10 seconds, got %dms", second.TimestampMilliseconds)
	}

	metrics := engine.Snapshot().Metrics
	if metrics.TotalUplinks != 2 || metrics.SuccessfulUplinks != 2 || metrics.TotalPacketsReceived != 2 {
		t.Fatalf("unexpected traffic metrics: %+v", metrics)
	}
	if metrics.PacketSuccessRate != 1 {
		t.Fatalf("expected 100%% success rate, got %f", metrics.PacketSuccessRate)
	}
}

func TestPhaseTwoDropsUplinkOutsideCoverageAndIgnoresRealGateway(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.AdvancedConfig.AntennaRange = 100

	outsideGateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")
	outsideGateway.Latitude = float32Ptr(37.51)
	insideRealGateway := validGateway("7f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c02")
	insideRealGateway.Type = contracts.Real
	insideRealGateway.KeepAlive = nil
	insideRealGateway.GatewayEUI = "A840410001000102"
	insideRealGateway.GatewayIPv4 = stringPtr("127.0.0.1")
	insideRealGateway.GatewayPort = int32Ptr(1700)

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device},
		[]contracts.Gateway{outsideGateway, insideRealGateway},
		types.Options{Clock: clock, EventBuffer: 128},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	dropped := advanceAndWait(t, clock, time.Second, engine.Events(), contracts.PacketDropped)
	if dropped.PacketID == nil {
		t.Fatal("expected dropped packet to have an id")
	}

	metrics := engine.Snapshot().Metrics
	if metrics.TotalUplinks != 1 || metrics.SuccessfulUplinks != 0 || metrics.DroppedUplinks != 1 {
		t.Fatalf("unexpected drop metrics: %+v", metrics)
	}
	if metrics.TotalPacketsReceived != 0 || metrics.PacketSuccessRate != 0 {
		t.Fatalf("expected no received packets: %+v", metrics)
	}
}

func TestPhaseTwoSchedulesOnlyActiveDevices(t *testing.T) {
	clock := types.NewManualClock(0)
	active := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&active)
	inactive := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000002")
	configureABP(&inactive)
	inactive.Active = false
	inactive.DevEUI = "70B3D57ED0000002"

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{active, inactive},
		nil,
		types.Options{Clock: clock, EventBuffer: 128},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	event := advanceAndWait(t, clock, time.Second, engine.Events(), contracts.PacketDropped)
	if event.DeviceID == nil || *event.DeviceID != active.ID {
		t.Fatalf("expected only active device to transmit, got %+v", event.DeviceID)
	}

	for _, logged := range engine.EventLog() {
		if logged.Type == contracts.DeviceUplinkScheduled && logged.DeviceID != nil && *logged.DeviceID == inactive.ID {
			t.Fatalf("inactive device was scheduled: %+v", logged)
		}
	}
}

func startEngine(t *testing.T, engine *Engine) {
	t.Helper()
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("start engine: %v", err)
	}
}

func stopEngine(t *testing.T, engine *Engine) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := engine.Stop(ctx); err != nil {
		t.Fatalf("stop engine: %v", err)
	}
}

func waitForEventType(t *testing.T, events <-chan contracts.SimulationEvent, expected contracts.SimulationEventType) contracts.SimulationEvent {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if event.Type == expected {
				return event
			}
		case <-deadline:
			t.Fatalf("timed out waiting for event %q", expected)
			return contracts.SimulationEvent{}
		}
	}
}

func stringPtr(value string) *string { return &value }

func configureABP(device *contracts.Device) {
	device.Activation = contracts.ABP
	device.OOTAConfig = nil
	device.ABPConfig = &contracts.ABPConfig{
		DevAddr: "26011BDA",
		NwkSKey: "00112233445566778899AABBCCDDEEFF",
		AppSKey: "FFEEDDCCBBAA99887766554433221100",
	}
}

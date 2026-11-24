package runtime

import (
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestReceiveWindowConfigurationUsesMilliseconds(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000099")
	device.RX1Config.Delay = intPtr(1000)
	device.RX1Config.Duration = intPtr(3000)
	device.RX2Config.Delay = intPtr(2000)
	device.RX2Config.Duration = intPtr(3000)
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, nil, types.Options{})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	if got := engine.sessions[device.ID].ReceiveDelay; got != time.Second {
		t.Fatalf("RX1 delay was not converted from milliseconds: got %s want %s", got, time.Second)
	}

	source := types.ScheduledEvent{At: 500 * time.Millisecond, WindowBaseAt: 500 * time.Millisecond, DeviceID: device.ID}
	engine.mu.Lock()
	engine.scheduleWindowLocked(device, source, contracts.RX1, "packet", -1, false)
	engine.scheduleWindowLocked(device, source, contracts.RX2, "packet", -1, false)
	engine.mu.Unlock()

	foundRX1, foundRX2 := false, false
	for _, scheduled := range engine.scheduler.Events() {
		switch scheduled.Kind {
		case types.ScheduledEventRX1Window:
			foundRX1 = true
			if scheduled.At != 1500*time.Millisecond || scheduled.WindowDuration != 3*time.Second {
				t.Fatalf("unexpected RX1 timing: at=%s duration=%s", scheduled.At, scheduled.WindowDuration)
			}
		case types.ScheduledEventRX2Window:
			foundRX2 = true
			if scheduled.At != 2500*time.Millisecond || scheduled.WindowDuration != 3*time.Second {
				t.Fatalf("unexpected RX2 timing: at=%s duration=%s", scheduled.At, scheduled.WindowDuration)
			}
		}
	}
	if !foundRX1 || !foundRX2 {
		t.Fatalf("receive windows were not both scheduled: rx1=%t rx2=%t", foundRX1, foundRX2)
	}
}

func TestJoinAcceptReceiveDelayUsesLoRaWANSecondsEncoding(t *testing.T) {
	tests := []struct {
		name  string
		value int
		want  time.Duration
	}{
		{name: "zero encodes one second", value: 0, want: time.Second},
		{name: "five encodes five seconds", value: 5, want: 5 * time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := joinAcceptReceiveDelay(test.value); got != test.want {
				t.Fatalf("unexpected Join-Accept RXDelay conversion: got %s want %s", got, test.want)
			}
		})
	}
}

func TestPhaseThreeOTAAJoinOpensRX1AndStartsUplinks(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	device.PayloadConfig.UplinkInterval = 10
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")

	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	waitForEventType(t, engine.Events(), contracts.DeviceJoinRequestTransmitted)
	advanceAndWait(t, clock, 2*time.Second, engine.Events(), contracts.JoinRequestReceived)
	joinAccept := waitForEventType(t, engine.Events(), contracts.JoinAcceptReceived)
	if joinAccept.RxWindow == nil || *joinAccept.RxWindow != contracts.RX1 {
		t.Fatalf("expected join accept in RX1, got %+v", joinAccept.RxWindow)
	}
	if !engine.Snapshot().Devices[0].Joined {
		t.Fatal("expected OTAA device to be joined")
	}

	if err := clock.Advance(10 * time.Second); err != nil {
		t.Fatalf("advance to first post-join uplink: %v", err)
	}
	uplink := waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	if uplink.FrameCounter == nil || *uplink.FrameCounter != 1 {
		t.Fatalf("expected first uplink frame counter to be 1, got %+v", uplink.FrameCounter)
	}

	metrics := engine.Snapshot().Metrics
	if metrics.JoinRequests != 1 || metrics.JoinAccepts != 1 || metrics.TotalUplinks != 1 {
		t.Fatalf("unexpected OTAA metrics: %+v", metrics)
	}
}

func TestPhaseThreeConfirmedUplinkReceivesACKInRX1(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	device.FrameConfig.Retransmission = intPtr(2)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")

	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	uplink := waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	if uplink.Confirmed == nil || !*uplink.Confirmed {
		t.Fatal("expected confirmed uplink metadata")
	}
	if !engine.Snapshot().Devices[0].PendingConfirmedUplink {
		t.Fatal("expected confirmed uplink to remain pending before ACK")
	}

	if err := clock.Advance(2 * time.Second); err != nil {
		t.Fatalf("advance to RX1: %v", err)
	}
	ack := waitForEventType(t, engine.Events(), contracts.UplinkACKReceived)
	if ack.RxWindow == nil || *ack.RxWindow != contracts.RX1 {
		t.Fatalf("expected ACK in RX1, got %+v", ack.RxWindow)
	}

	snapshot := engine.Snapshot()
	if snapshot.Devices[0].PendingConfirmedUplink {
		t.Fatal("expected confirmed uplink to be cleared after ACK")
	}
	if snapshot.Devices[0].FrameCounterUp != 1 || snapshot.Devices[0].FrameCounterDown != 1 {
		t.Fatalf("unexpected frame counters: %+v", snapshot.Devices[0])
	}
	if snapshot.Metrics.AcknowledgedUplinks != 1 || snapshot.Metrics.Retransmissions != 0 {
		t.Fatalf("unexpected ACK metrics: %+v", snapshot.Metrics)
	}
}

func TestPhaseThreeConfirmedUplinkRetriesAfterRX2Timeout(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	device.FrameConfig.Retransmission = intPtr(2)
	device.RX2Config.Duration = intPtr(1000)

	engine := newTestEngine(t, clock, device)
	defer stopEngine(t, engine)

	waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	for attempt := 1; attempt <= 2; attempt++ {
		advanceAndWait(t, clock, 2*time.Second, engine.Events(), contracts.RX1WindowOpened)
		advanceAndWait(t, clock, 2*time.Second, engine.Events(), contracts.RX2WindowOpened)
		advanceAndWait(t, clock, 2*time.Second, engine.Events(), contracts.UplinkRetryScheduled)
		transmitted := waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
		if transmitted.RetryAttempt == nil || int(*transmitted.RetryAttempt) != attempt+1 {
			t.Fatalf("expected retry attempt %d, got %+v", attempt+1, transmitted.RetryAttempt)
		}
	}

	advanceAndWait(t, clock, 2*time.Second, engine.Events(), contracts.RX1WindowOpened)
	advanceAndWait(t, clock, 2*time.Second, engine.Events(), contracts.RX2WindowOpened)
	waitForEventType(t, engine.Events(), contracts.PacketDropped)

	metrics := engine.Snapshot().Metrics
	if metrics.TotalUplinks != 1 || metrics.Retransmissions != 2 || metrics.DroppedUplinks != 1 {
		t.Fatalf("unexpected retransmission metrics: %+v", metrics)
	}
	if metrics.FrameCounterErrors != 0 || metrics.SuccessfulUplinks != 0 {
		t.Fatalf("unexpected frame counter/success metrics: %+v", metrics)
	}
}

func TestDeviceRadioAndReceiveWindowConfigurationIsApplied(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	device.PayloadConfig.Payload = "this payload is intentionally longer than the DR0 regional limit"
	device.PayloadConfig.OversizedPayloadBehavior = contracts.Truncate
	device.RX1Config.DataRateOffset = intPtr(2)
	configuredRate := 0
	device.AdvancedConfig.UplinkDataRate = &configuredRate
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")

	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	uplink := waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	if uplink.DataRate == nil || *uplink.DataRate != 0 || uplink.SpreadingFactor == nil || *uplink.SpreadingFactor != 12 {
		t.Fatalf("expected configured DR0/SF12 uplink, got %+v", uplink)
	}
	if uplink.PayloadSize == nil || *uplink.PayloadSize != 51 {
		t.Fatalf("expected DR0 payload truncation to 51 bytes, got %+v", uplink.PayloadSize)
	}

	advanceAndWait(t, clock, 4*time.Second, engine.Events(), contracts.RX1WindowOpened)
	rx1 := waitForEventType(t, engine.Events(), contracts.UplinkACKReceived)
	if rx1.RxWindow == nil || *rx1.RxWindow != contracts.RX1 {
		t.Fatalf("expected RX1 ACK, got %+v", rx1.RxWindow)
	}
	if rx1.DataRate == nil || *rx1.DataRate != 0 {
		t.Fatalf("expected RX1 DR0 after offset clamp, got %+v", rx1.DataRate)
	}
}

func TestOversizedPayloadFragmentationUsesMultiplePHYFrames(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.PayloadConfig.Payload = "this payload is intentionally longer than the regional DR0 maximum payload size"
	device.PayloadConfig.OversizedPayloadBehavior = contracts.Fragment
	fragmentRate := 0
	device.AdvancedConfig.UplinkDataRate = &fragmentRate
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")

	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)
	waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)

	if err := clock.Advance(5 * time.Second); err != nil {
		t.Fatalf("advance to complete fragmented payload: %v", err)
	}
	waitForEventType(t, engine.Events(), contracts.GatewayPacketReceived)
	waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	waitForEventType(t, engine.Events(), contracts.GatewayPacketReceived)

	transmissions := make([]contracts.SimulationEvent, 0, 2)
	for _, event := range engine.EventLog() {
		if event.Type == contracts.RadioTransmissionStarted && event.PayloadSize != nil {
			transmissions = append(transmissions, event)
		}
	}
	if len(transmissions) < 2 {
		t.Fatalf("expected at least two radio frames for fragmented payload, got %d", len(transmissions))
	}
	if *transmissions[0].PayloadSize != 51 || *transmissions[1].PayloadSize <= 0 || *transmissions[1].PayloadSize >= 51 {
		t.Fatalf("unexpected fragment sizes: %+v, %+v", transmissions[0].PayloadSize, transmissions[1].PayloadSize)
	}
}

func newTestEngine(t *testing.T, clock *types.ManualClock, device contracts.Device, gateways ...contracts.Gateway) *Engine {
	t.Helper()
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, gateways, types.Options{Clock: clock, EventBuffer: 256})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	return engine
}

func advanceAndWait(t *testing.T, clock *types.ManualClock, delta time.Duration, events <-chan contracts.SimulationEvent, eventType contracts.SimulationEventType) contracts.SimulationEvent {
	t.Helper()
	if err := clock.Advance(delta); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	return waitForEventType(t, events, eventType)
}

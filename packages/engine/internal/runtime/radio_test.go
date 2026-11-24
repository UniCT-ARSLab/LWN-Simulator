package runtime

import (
	"encoding/base64"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestPayloadAndRadioParameterHelpers(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000207")
	device.PayloadConfig.Payload = base64.StdEncoding.EncodeToString([]byte("binary payload"))
	device.PayloadConfig.Base64Encoded = true
	if got := string(payloadBytes(device)); got != "binary payload" {
		t.Fatalf("base64 payload was not decoded: %q", got)
	}
	if size := effectivePayloadSize(device, 5); size != len("binary payload") {
		t.Fatalf("effective payload size ignored encoding: %d", size)
	}
	device.PayloadConfig.Payload = "invalid%%"
	if payload := payloadBytes(device); payload != nil {
		t.Fatalf("invalid base64 should not produce bytes: %x", payload)
	}

	configured := 2
	device.AdvancedConfig.UplinkDataRate = &configured
	if got := currentDeviceDataRate(nil, device); got != 2 {
		t.Fatalf("configured data rate was ignored: %d", got)
	}
	session := newDeviceSession(device)
	session.CurrentDataRate = 4
	if got := currentDeviceDataRate(session, device); got != 4 {
		t.Fatalf("session data rate was ignored: %d", got)
	}
	session.CurrentDataRate = 99
	if got := currentDeviceDataRate(session, device); got != 2 {
		t.Fatalf("invalid session rate should fall back to configured rate: %d", got)
	}

	if minimumRequiredSNR(7) != -7.5 || minimumRequiredSNR(99) != -20 {
		t.Fatal("spreading-factor SNR thresholds are incorrect")
	}
	if dataRateString(6, 125_000) != "" || dataRateString(7, 0) != "" || dataRateString(12, 125_000) != "SF12BW125" {
		t.Fatal("Semtech data-rate formatting accepted invalid modulation parameters")
	}
}

func TestPhaseFourRadioEventsExposeLinkMetrics(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.PayloadConfig.Payload = "temperature=21.8"
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")

	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	if err := clock.Advance(time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	started := waitForEventType(t, engine.Events(), contracts.RadioTransmissionStarted)
	if started.ChannelFrequency == nil || *started.ChannelFrequency != 868300000 {
		t.Fatalf("expected EU868 channel frequency, got %+v", started.ChannelFrequency)
	}
	if started.Bandwidth == nil || *started.Bandwidth != 125000 {
		t.Fatalf("expected 125 kHz bandwidth, got %+v", started.Bandwidth)
	}
	if started.SpreadingFactor == nil || *started.SpreadingFactor != 7 {
		t.Fatalf("expected SF7 for data rate 5, got %+v", started.SpreadingFactor)
	}
	if started.AirtimeMilliseconds == nil || *started.AirtimeMilliseconds < 1 {
		t.Fatalf("expected positive airtime, got %+v", started.AirtimeMilliseconds)
	}
	if started.RSSI == nil || started.SNR == nil {
		t.Fatal("expected RSSI and SNR metadata on radio event")
	}
	if started.RadioFrame == nil || started.RadioFrame.Protocol != contracts.LoRaWAN {
		t.Fatalf("expected an explicit LoRaWAN radio frame, got %+v", started.RadioFrame)
	}

	received := waitForEventType(t, engine.Events(), contracts.GatewayPacketReceived)
	if received.RSSI == nil || received.SNR == nil || received.RadioLossReason != nil {
		t.Fatalf("unexpected received radio metadata: %+v", received)
	}

	snapshot := engine.Snapshot()
	if snapshot.Metrics.TotalTransmissions != 1 || snapshot.Metrics.TotalPacketsReceived != 1 {
		t.Fatalf("unexpected radio metrics: %+v", snapshot.Metrics)
	}
	if snapshot.Metrics.AverageAirtimeMilliseconds <= 0 {
		t.Fatalf("expected positive average airtime: %+v", snapshot.Metrics)
	}
	if snapshot.Devices[0].LastChannelFrequency != 868300000 || snapshot.Devices[0].LastAirtimeMilliseconds <= 0 {
		t.Fatalf("expected runtime radio metrics: %+v", snapshot.Devices[0])
	}
}

func TestRadioEventsExposeDecodedPayloadWhenCodecIsEnabled(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000209")
	configureABP(&device)
	device.PayloadConfig.Payload = "21"
	device.PayloadConfig.PayloadCodec = &contracts.PayloadCodec{
		Enabled: true,
		Script:  "function decodeUplink(input) { return { data: { value: input.bytes[0], port: input.fPort } }; }",
	}
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c09")
	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	if err := clock.Advance(time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	event := waitForEventType(t, engine.Events(), contracts.RadioTransmissionStarted)
	if event.DecodedPayload == nil {
		t.Fatalf("expected decoded payload on radio event: %+v", event)
	}
	decoded := *event.DecodedPayload
	if decoded["value"] != float64('2') || decoded["port"] != float64(device.FrameConfig.FPort) {
		t.Fatalf("unexpected decoded payload: %#v", decoded)
	}
}

func TestPhaseFourRadioDetectsOverlappingSameChannelTransmissions(t *testing.T) {
	clock := types.NewManualClock(0)
	first := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&first)
	second := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000002")
	configureABP(&second)
	second.Name = "Test Device 2"
	second.DevEUI = "70B3D57ED0000002"
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{first, second},
		[]contracts.Gateway{gateway},
		types.Options{Clock: clock, EventBuffer: 256},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	if _, err := engine.QueueUplink(first.ID); err != nil {
		t.Fatalf("queue first simultaneous uplink: %v", err)
	}
	if _, err := engine.QueueUplink(second.ID); err != nil {
		t.Fatalf("queue second simultaneous uplink: %v", err)
	}

	if err := clock.Advance(time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	waitForEventCounts(t, engine.Events(), map[contracts.SimulationEventType]int{
		contracts.RadioCollisionDetected: 2,
		contracts.RadioPacketLost:        2,
	})

	metrics := engine.Snapshot().Metrics
	if metrics.TotalTransmissions != 2 || metrics.Collisions != 2 || metrics.RadioPacketLosses != 2 {
		t.Fatalf("unexpected collision metrics: %+v", metrics)
	}
	if metrics.TotalPacketsReceived != 0 || metrics.SuccessfulUplinks != 0 {
		t.Fatalf("collided packets must not be received: %+v", metrics)
	}
}

func waitForEventCounts(t *testing.T, events <-chan contracts.SimulationEvent, expected map[contracts.SimulationEventType]int) {
	t.Helper()
	remaining := make(map[contracts.SimulationEventType]int, len(expected))
	for eventType, count := range expected {
		remaining[eventType] = count
	}
	deadline := time.After(time.Second)
	for len(remaining) > 0 {
		select {
		case event := <-events:
			if remaining[event.Type] > 0 {
				remaining[event.Type]--
				if remaining[event.Type] == 0 {
					delete(remaining, event.Type)
				}
			}
		case <-deadline:
			t.Fatalf("timed out waiting for event counts: %+v", remaining)
		}
	}
}

func TestPhaseFourRadioReportsNoGatewayLoss(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)

	engine := newTestEngine(t, clock, device)
	defer stopEngine(t, engine)

	if err := clock.Advance(time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	lost := waitForEventType(t, engine.Events(), contracts.RadioPacketLost)
	if lost.RadioLossReason == nil || *lost.RadioLossReason != contracts.NoGateway {
		t.Fatalf("expected no-gateway loss, got %+v", lost.RadioLossReason)
	}

	metrics := engine.Snapshot().Metrics
	if metrics.RadioPacketLosses != 1 || metrics.DroppedUplinks != 1 || metrics.TotalPacketsReceived != 0 {
		t.Fatalf("unexpected no-gateway metrics: %+v", metrics)
	}
}

func TestPhaseFourRegionalChannelsAndAirtime(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")

	device.LocationConfig.Region = contracts.EU868
	eu868 := selectRadioChannel(device, 1)
	if eu868.Frequency != 868300000 {
		t.Fatalf("unexpected EU868 channel: %+v", eu868)
	}

	device.LocationConfig.Region = contracts.US915
	us915 := selectRadioChannel(device, 1)
	if us915.Frequency != 902500000 {
		t.Fatalf("unexpected US915 channel: %+v", us915)
	}

	sf7 := calculateAirtime(device, 7, radioBandwidthHz)
	sf12 := calculateAirtime(device, 12, radioBandwidthHz)
	if sf7 <= 0 || sf12 <= sf7 {
		t.Fatalf("expected SF12 airtime to exceed SF7: sf7=%s sf12=%s", sf7, sf12)
	}
}

func TestBridgeUplinkPreservesCalculatedSignal(t *testing.T) {
	for _, protocol := range []contracts.RadioProtocol{contracts.LoRaWAN, contracts.RawLoRa} {
		t.Run(string(protocol), func(t *testing.T) {
			device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000207")
			configureABP(&device)
			device.RadioProtocol = &protocol
			gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c09")
			engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{Clock: types.NewManualClock(0)})
			if err != nil {
				t.Fatal(err)
			}
			engine.gatewayAdapters[gateway.ID] = newFakeGatewayAdapter()
			engine.startRadioTransmissionLocked(device, types.ScheduledEvent{DeviceID: device.ID, Attempt: 1}, "signal-test", 1, false)
			transmission := engine.radio["signal-test"]
			if transmission == nil {
				t.Fatal("missing radio transmission")
			}
			engine.processRadioTransmissionCompletedLocked(types.ScheduledEvent{PacketID: transmission.PacketID, DeviceID: device.ID, At: transmission.EndAt})
			if len(engine.gatewayPackets) != 1 {
				t.Fatalf("expected a bridge uplink, got %d", len(engine.gatewayPackets))
			}
			packet := engine.gatewayPackets[0]
			if packet.RSSI != transmission.RSSI || packet.SNR != transmission.SNR {
				t.Fatalf("calculated signal lost on bridge uplink: packet=%+v RSSI=%v SNR=%v", packet, transmission.RSSI, transmission.SNR)
			}
		})
	}
}

func TestLivePositionChangesAffectNextTransmission(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{Clock: types.NewManualClock(0)})
	if err != nil {
		t.Fatal(err)
	}
	engine.state.Status = contracts.SimulationStatusRunning
	session := engine.sessions[device.ID]
	transmit := func(id string) *types.RadioTransmission {
		current, _ := engine.registry.Device(device.ID)
		engine.startRadioTransmissionLocked(current, types.ScheduledEvent{DeviceID: device.ID, Attempt: 1}, id, 1, false)
		packet := engine.radio[id]
		if packet == nil {
			t.Fatal("expected a radio transmission")
		}
		delete(engine.radio, id)
		return packet
	}

	initial := transmit("initial")
	if len(initial.GatewayIDs) != 1 {
		t.Fatal("expected initial coverage")
	}
	device.LocationConfig.Latitude = float32Ptr(40)
	if err := engine.UpdateDevice(device); err != nil {
		t.Fatal(err)
	}
	if packet := transmit("out-of-range"); len(packet.GatewayIDs) != 0 {
		t.Fatal("device movement did not remove coverage")
	}
	gateway.Latitude = float32Ptr(40)
	gateway.Altitude = float32Ptr(150)
	if err := engine.UpdateGateway(gateway); err != nil {
		t.Fatal(err)
	}
	if packet := transmit("back-in-range"); len(packet.GatewayIDs) != 1 || packet.GatewayIDs[0] != gateway.ID || packet.RSSI >= initial.RSSI {
		t.Fatalf("expected coverage restored with weaker signal after moving the gateway: %+v", packet)
	}
	if engine.sessions[device.ID] != session || !session.Joined {
		t.Fatal("movement reset the device session")
	}
}

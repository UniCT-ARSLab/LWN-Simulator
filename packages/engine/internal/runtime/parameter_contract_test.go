package runtime

import (
	"encoding/base64"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestDeviceParametersDriveProtocolSessionAndUplinkFrame(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000201")
	configureABP(&device)
	device.Name = "Parameter contract device"
	device.Class = contracts.ClassC
	device.FrameConfig.FPort = 42
	device.FrameConfig.FCntUp = optionalCounter(17)
	device.FrameConfig.FCntDown = optionalCounter(23)
	device.FrameConfig.Retransmission = intPtr(4)
	device.FrameConfig.DisableFrameCounterValidation = true
	device.PayloadConfig.Payload = base64.StdEncoding.EncodeToString([]byte("decoded payload"))
	device.PayloadConfig.Base64Encoded = true
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	device.PayloadConfig.UplinkInterval = 37.5
	device.RX1Config.Delay = intPtr(1250)
	device.RX1Config.Duration = intPtr(750)
	device.RX1Config.DataRateOffset = intPtr(2)
	device.RX2Config.Delay = intPtr(2750)
	device.RX2Config.Duration = intPtr(900)
	device.RX2Config.ChannelFrequency = 869_525_000
	device.RX2Config.DataRate = intPtr(3)
	device.RX2Config.ACKTimeout = 2300
	device.AdvancedConfig.ADREnabled = true
	dataRate := 3
	device.AdvancedConfig.UplinkDataRate = &dataRate

	session := newDeviceSession(device)
	if !session.Joined || session.DeviceAddress != device.ABPConfig.DevAddr {
		t.Fatalf("ABP activation parameters were not applied: %+v", session)
	}
	if session.FrameCounterUp != 17 || session.FrameCounterDown != 23 {
		t.Fatalf("initial frame counters were not applied: %+v", session)
	}
	if session.CurrentDataRate != 3 || session.CurrentSpreadingFactor != 9 {
		t.Fatalf("configured uplink data rate was not applied: %+v", session)
	}
	if session.RX1DataRateOffset != 2 || session.RX2DataRate != 3 || session.RX2Frequency != 869_525_000 {
		t.Fatalf("receive-window radio parameters were not applied: %+v", session)
	}
	if session.ReceiveDelay != 1250*time.Millisecond {
		t.Fatalf("RX1 delay has the wrong unit: got %s", session.ReceiveDelay)
	}

	frame := (&Engine{}).buildUplinkPayloadLocked(device, session, session.FrameCounterUp, true, 0, dataRate)
	packet, err := lorawan.Parse(frame)
	if err != nil {
		t.Fatalf("parse generated uplink: %v", err)
	}
	if !packet.Confirmed || !packet.ADR || packet.FCnt != 17 || packet.FPort == nil || *packet.FPort != 42 {
		t.Fatalf("frame, MType, ADR, counter or FPort parameter was lost: %+v", packet)
	}
	payload, err := lorawan.DecryptPayload(packet, session.AppSKey, 0)
	if err != nil || string(payload) != "decoded payload" {
		t.Fatalf("base64 payload was not decoded before transmission: payload=%q err=%v", payload, err)
	}
}

func TestConfiguredReceiveWindowParametersAreUsedExactly(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000202")
	configureABP(&device)
	device.RX1Config.Delay = intPtr(1250)
	device.RX1Config.Duration = intPtr(750)
	device.RX1Config.DataRateOffset = intPtr(2)
	device.RX2Config.Delay = intPtr(2750)
	device.RX2Config.Duration = intPtr(900)
	device.RX2Config.ChannelFrequency = 869_525_000
	device.RX2Config.DataRate = intPtr(3)
	dataRate := 5
	device.AdvancedConfig.UplinkDataRate = &dataRate

	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, nil, types.Options{})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	source := types.ScheduledEvent{At: 400 * time.Millisecond, WindowBaseAt: 400 * time.Millisecond, DeviceID: device.ID, ChannelFrequency: 868_300_000}
	engine.mu.Lock()
	engine.scheduleWindowLocked(device, source, contracts.RX1, "rx1-packet", 1, true)
	engine.scheduleWindowLocked(device, source, contracts.RX2, "rx2-packet", 1, true)
	engine.mu.Unlock()

	for _, scheduled := range engine.scheduler.Events() {
		switch scheduled.Window {
		case contracts.RX1:
			if scheduled.At != 1650*time.Millisecond || scheduled.WindowDuration != 750*time.Millisecond || scheduled.DataRate != 3 || scheduled.ChannelFrequency != 868_300_000 {
				t.Fatalf("RX1 parameters were not used exactly: %+v", scheduled)
			}
		case contracts.RX2:
			if scheduled.At != 3150*time.Millisecond || scheduled.WindowDuration != 900*time.Millisecond || scheduled.DataRate != 3 || scheduled.ChannelFrequency != 869_525_000 {
				t.Fatalf("RX2 parameters were not used exactly: %+v", scheduled)
			}
		}
	}
}

func TestGatewayParametersDriveCoverageSignalHeartbeatAndSnapshot(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000203")
	configureABP(&device)
	device.AdvancedConfig.AntennaRange = 1500
	device.LocationConfig.Altitude = float32Ptr(10)

	inside := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7203")
	inside.Name = "Inside gateway"
	inside.GatewayEUI = "A840410001000203"
	inside.KeepAlive = int32Ptr(17)
	inside.Altitude = float32Ptr(310)
	out := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7204")
	out.Name = "Outside gateway"
	out.GatewayEUI = "A840410001000204"
	out.Latitude = float32Ptr(38.5)
	out.Longitude = float32Ptr(16.1)

	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{inside, out}, types.Options{})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	covered := engine.coveredGateways(device)
	if len(covered) != 1 || covered[0].ID != inside.ID {
		t.Fatalf("location and antenna range selected the wrong gateways: %+v", covered)
	}
	flat := inside
	flat.Altitude = device.LocationConfig.Altitude
	rssiWithAltitude, _ := calculateSignal(device, []contracts.Gateway{inside}, 868_300_000, 14)
	rssiFlat, _ := calculateSignal(device, []contracts.Gateway{flat}, 868_300_000, 14)
	if rssiWithAltitude >= rssiFlat {
		t.Fatalf("gateway/device altitude did not affect signal strength: elevated=%f flat=%f", rssiWithAltitude, rssiFlat)
	}

	startEngine(t, engine)
	defer stopEngine(t, engine)
	foundHeartbeat := false
	for _, scheduled := range engine.scheduler.Events() {
		if scheduled.GatewayID == inside.ID && scheduled.Kind == types.ScheduledEventGatewayHeartbeat {
			foundHeartbeat = scheduled.At == 17*time.Second
		}
	}
	if !foundHeartbeat {
		t.Fatal("virtual gateway keepAlive did not schedule the exact heartbeat interval")
	}
	checkpoint := engine.Checkpoint()
	if len(checkpoint.Gateways) != 2 || checkpoint.Gateways[0].Name != inside.Name || checkpoint.Gateways[0].GatewayEUI != inside.GatewayEUI {
		t.Fatalf("gateway identity parameters were not preserved: %+v", checkpoint.Gateways)
	}
}

func TestFrameCounterValidationFlagAndACKTimeoutAreApplied(t *testing.T) {
	makeEngine := func(disableValidation bool) (*Engine, contracts.Device) {
		device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000208")
		configureABP(&device)
		device.FrameConfig.DisableFrameCounterValidation = disableValidation
		device.FrameConfig.Retransmission = intPtr(1)
		device.RX2Config.ACKTimeout = 2300
		engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, nil, types.Options{})
		if err != nil {
			t.Fatalf("create engine: %v", err)
		}
		engine.sessions[device.ID].FrameCounterUp = 5
		return engine, device
	}

	strict, strictDevice := makeEngine(false)
	strict.mu.Lock()
	strictEvents := strict.processUplinkLocked(types.ScheduledEvent{ID: "strict", At: time.Second, DeviceID: strictDevice.ID, Attempt: 1, FrameCounter: 5})
	strict.mu.Unlock()
	if len(strictEvents) != 1 || strictEvents[0].Type != contracts.FrameCounterRejected {
		t.Fatalf("strict frame-counter validation accepted a replay: %+v", strictEvents)
	}

	permissive, permissiveDevice := makeEngine(true)
	permissive.mu.Lock()
	permissiveEvents := permissive.processUplinkLocked(types.ScheduledEvent{ID: "permissive", At: time.Second, DeviceID: permissiveDevice.ID, Attempt: 1, FrameCounter: 5})
	permissive.mu.Unlock()
	if len(permissiveEvents) == 0 || permissiveEvents[0].Type == contracts.FrameCounterRejected {
		t.Fatalf("disabled frame-counter validation still rejected a replay: %+v", permissiveEvents)
	}

	session := permissive.sessions[permissiveDevice.ID]
	session.PendingUplink = &types.PendingUplink{PacketID: "pending", FrameCounter: 6, Attempt: 1, Confirmed: true}
	permissive.mu.Lock()
	permissive.processReceiveWindowLocked(types.ScheduledEvent{
		ID: "rx2", At: 10 * time.Second, DeviceID: permissiveDevice.ID, Kind: types.ScheduledEventRX2Window,
		Window: contracts.RX2, PacketID: "pending", FrameCounter: 6, Attempt: 1, Confirmed: true,
	})
	permissive.mu.Unlock()
	foundRetry := false
	for _, scheduled := range permissive.scheduler.Events() {
		if scheduled.Kind == types.ScheduledEventDeviceUplink && scheduled.PacketID == "pending" && scheduled.Attempt == 2 {
			foundRetry = scheduled.At == 12_300*time.Millisecond
		}
	}
	if !foundRetry {
		t.Fatal("ACK timeout was not applied in milliseconds to the retransmission schedule")
	}
}

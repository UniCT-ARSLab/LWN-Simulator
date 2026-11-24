package runtime

import (
	"fmt"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestRawDeviceTransmitsWithoutLoRaWANJoin(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000701")
	raw := contracts.RawLoRa
	device.RadioProtocol = &raw
	device.RadioReceiveMode = receiveModePointer(contracts.ReceiveRawLoRa)
	device.PayloadConfig.Payload = "raw payload"
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c71")

	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	started := waitForEventType(t, engine.Events(), contracts.RadioTransmissionStarted)
	if started.RadioFrame == nil || started.RadioFrame.Protocol != contracts.RawLoRa {
		t.Fatalf("expected raw radio frame, got %+v", started.RadioFrame)
	}
	if string(started.RadioFrame.Payload) != "raw payload" {
		t.Fatalf("unexpected raw payload: %q", started.RadioFrame.Payload)
	}

	if err := clock.Advance(time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	received := waitForEventType(t, engine.Events(), contracts.GatewayPacketReceived)
	if received.RadioFrame == nil || received.RadioFrame.Protocol != contracts.RawLoRa {
		t.Fatalf("expected raw protocol on gateway event, got %+v", received.RadioFrame)
	}
	if engine.Snapshot().Devices[0].Joined {
		t.Fatal("raw device must not require a LoRaWAN join")
	}
}

func TestRawUplinkDoesNotUseLoRaWANConfirmationSemantics(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000704")
	raw := contracts.RawLoRa
	device.RadioProtocol = &raw
	device.RadioReceiveMode = receiveModePointer(contracts.ReceiveRawLoRa)
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c74")

	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	uplink := waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	if uplink.Confirmed == nil || *uplink.Confirmed {
		t.Fatalf("raw uplink inherited LoRaWAN confirmation semantics: %+v", uplink.Confirmed)
	}
	engine.mu.RLock()
	pending := engine.sessions[device.ID].PendingUplink
	engine.mu.RUnlock()
	if pending != nil {
		t.Fatalf("raw uplink created a LoRaWAN retry state: %+v", pending)
	}
}

func TestRawDownlinkOmitsLoRaWANOnlyMetadata(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000705")
	raw := contracts.RawLoRa
	device.RadioProtocol = &raw
	device.RadioReceiveMode = receiveModePointer(contracts.ReceiveRawLoRa)
	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device},
		nil,
		types.Options{EventBuffer: 256},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	if _, err := engine.QueueDownlink(types.Downlink{
		DeviceID:  device.ID,
		Payload:   []byte("raw downlink"),
		FPort:     223,
		Confirmed: true,
		ACK:       true,
		FPending:  true,
		DataRate:  -1,
	}); err != nil {
		t.Fatalf("queue raw downlink: %v", err)
	}
	event := waitForEventType(t, engine.Events(), contracts.DeviceDownlinkScheduled)
	if event.FPort != nil || event.Confirmed != nil || event.FrameCounter != nil {
		t.Fatalf("raw downlink exposed LoRaWAN-only metadata: %+v", event)
	}
}

func TestRawDownlinkIsAcceptedOnlyByRawReceivingDevice(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000702")
	raw := contracts.RawLoRa
	device.RadioProtocol = &raw
	device.RadioReceiveMode = receiveModePointer(contracts.ReceiveRawLoRa)
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, nil, types.Options{})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}

	engine.mu.Lock()
	events := engine.processRawDownlinkLocked(types.GatewayPacket{
		GatewayID: "gateway",
		DeviceID:  device.ID,
		Protocol:  contracts.RawLoRa,
		Kind:      types.GatewayPacketDownlink,
		Payload:   []byte{0x01, 0x02},
	})
	engine.mu.Unlock()
	if len(events) != 1 || events[0].Type != contracts.DeviceDownlinkTransmitted {
		t.Fatalf("raw downlink was not delivered: %+v", events)
	}
	if events[0].RadioFrame == nil || events[0].RadioFrame.Protocol != contracts.RawLoRa {
		t.Fatalf("raw downlink protocol was not exposed: %+v", events[0].RadioFrame)
	}

	lorawanMode := contracts.ReceiveLoRaWAN
	device.RadioReceiveMode = &lorawanMode
	registry, err := NewRegistry([]contracts.Device{device}, nil)
	if err != nil {
		t.Fatalf("rebuild registry: %v", err)
	}
	engine.registry = registry
	engine.mu.Lock()
	rejected := engine.processRawDownlinkLocked(types.GatewayPacket{
		GatewayID: "gateway",
		DeviceID:  device.ID,
		Protocol:  contracts.RawLoRa,
		Kind:      types.GatewayPacketDownlink,
		Payload:   []byte{0x01},
	})
	engine.mu.Unlock()
	if len(rejected) != 1 || rejected[0].Type != contracts.DeviceDownlinkDropped {
		t.Fatalf("raw downlink should be rejected by LoRaWAN-only device: %+v", rejected)
	}
}

func TestRawDownlinkTravelsThroughConnectedGateway(t *testing.T) {
	adapter := newFakeGatewayAdapter()
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000703")
	raw := contracts.RawLoRa
	device.RadioProtocol = &raw
	device.RadioReceiveMode = receiveModePointer(contracts.ReceiveRawLoRa)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c73")
	gateway.Type = contracts.Real
	gateway.KeepAlive = nil
	gateway.GatewayEUI = "A840410001000173"
	gateway.GatewayIPv4 = engineStringPtr("127.0.0.1")
	gateway.GatewayPort = engineInt32Ptr(1773)

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device},
		[]contracts.Gateway{gateway},
		types.Options{EventBuffer: 256, GatewayAdapterFactory: &fakeGatewayFactory{adapter: adapter}},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	adapter.packets <- types.GatewayPacket{
		GatewayID: gateway.ID,
		DeviceID:  device.ID,
		Protocol:  contracts.RawLoRa,
		Kind:      types.GatewayPacketDownlink,
		Payload:   []byte{0xde, 0xad, 0xbe, 0xef},
	}

	event := waitForEventType(t, engine.Events(), contracts.DeviceDownlinkTransmitted)
	if event.DeviceID == nil || *event.DeviceID != device.ID {
		t.Fatalf("downlink was assigned to device %v, want %q", event.DeviceID, device.ID)
	}
	if event.RadioFrame == nil || event.RadioFrame.Protocol != contracts.RawLoRa {
		t.Fatalf("expected raw radio frame metadata, got %+v", event.RadioFrame)
	}
	if string(event.RadioFrame.Payload) != string([]byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Fatalf("unexpected raw downlink payload: %x", event.RadioFrame.Payload)
	}
}

func receiveModePointer(mode contracts.RadioReceiveMode) *contracts.RadioReceiveMode {
	return &mode
}

func TestLoRaWANUplinkRespectsDeviceReceiveMode(t *testing.T) {
	for _, mode := range []contracts.RadioReceiveMode{contracts.ReceiveLoRaWAN, contracts.ReceiveRawLoRa, contracts.ReceiveAuto} {
		t.Run(string(mode), func(t *testing.T) {
			device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000214")
			configureABP(&device)
			device.RadioReceiveMode = &mode
			engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, nil, types.Options{})
			if err != nil {
				t.Fatal(err)
			}
			session := newDeviceSession(device)
			port := byte(1)
			frame, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{DevAddr: 0x26011bda, FCnt: 1, FPort: &port, Payload: []byte{21, 50}, NwkSKey: session.NwkSKey, AppSKey: session.AppSKey})
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := lorawan.Parse(frame)
			if err != nil {
				t.Fatal(err)
			}
			events := engine.processRealUplinkLocked(types.GatewayPacket{GatewayID: "gateway", Payload: frame}, decoded)
			if len(events) == 0 {
				t.Fatal("missing reception outcome")
			}
			if mode == contracts.ReceiveRawLoRa {
				if events[0].Type != contracts.PacketDropped || engine.Snapshot().Metrics.SuccessfulUplinks != 0 {
					t.Fatal("LoRaWAN traffic was accepted by a raw-only receiver")
				}
			} else if events[0].Type != contracts.DeviceUplinkTransmitted || engine.Snapshot().Metrics.SuccessfulUplinks != 1 {
				t.Fatalf("LoRaWAN traffic was not accepted in mode %s: %+v", mode, events)
			}
		})
	}
}

func TestRawGatewayReceptionModesAndDeviceFailures(t *testing.T) {
	for _, mode := range []contracts.RadioReceiveMode{contracts.ReceiveAuto, contracts.ReceiveRawLoRa, contracts.ReceiveLoRaWAN} {
		for _, active := range []bool{true, false} {
			t.Run(string(mode)+"/active="+fmt.Sprint(active), func(t *testing.T) {
				device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000706")
				device.Active = active
				device.RadioReceiveMode = &mode
				engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, nil, types.Options{})
				if err != nil {
					t.Fatal(err)
				}
				packet := types.GatewayPacket{DeviceID: device.ID, GatewayID: "gateway", Protocol: contracts.RawLoRa, Payload: []byte{0xde, 0xad}}
				for _, downlink := range []bool{false, true} {
					var events []contracts.SimulationEvent
					expected := contracts.PacketDropped
					if downlink {
						events = engine.processRawDownlinkLocked(packet)
						expected = contracts.DeviceDownlinkDropped
					} else {
						events = engine.processRawUplinkLocked(packet)
					}
					if active && mode != contracts.ReceiveLoRaWAN {
						expected = contracts.DeviceUplinkTransmitted
						if downlink {
							expected = contracts.DeviceDownlinkTransmitted
						}
					}
					if len(events) != 1 || events[0].Type != expected {
						t.Fatalf("downlink=%v: expected %s, got %+v", downlink, expected, events)
					}
					if events[0].RadioFrame != nil && events[0].RadioFrame.TimestampMilliseconds != events[0].TimestampMilliseconds {
						t.Fatal("raw frame timestamp must use simulation time")
					}
				}
			})
		}
	}
}

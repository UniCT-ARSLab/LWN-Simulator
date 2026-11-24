package runtime

import (
	"context"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestBuildUplinkPayloadCreatesABPDataFrame(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000081")
	configureABP(&device)
	session := newDeviceSession(device)
	session.FrameCounterUp = 12

	payload := (&Engine{}).buildUplinkPayloadLocked(device, session, 13, true, 0, 5)
	packet, err := lorawan.Parse(payload)
	if err != nil {
		t.Fatalf("parse generated ABP uplink: %v", err)
	}
	if packet.MType != lorawan.MTypeConfirmedDataUp || packet.DevAddr != 0x26011BDA || packet.FCnt != 13 {
		t.Fatalf("unexpected generated ABP frame: %+v", packet)
	}
	if packet.FPort == nil || *packet.FPort != byte(device.FrameConfig.FPort) {
		t.Fatalf("unexpected generated ABP FPort: %+v", packet.FPort)
	}
}

func TestBuildUplinkPayloadCreatesOTAAJoinRequest(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000082")
	session := newDeviceSession(device)
	session.PendingJoinRequest = &types.PendingJoinRequest{JoinEUI: device.OOTAConfig.JoinEUI}

	payload := (&Engine{}).buildUplinkPayloadLocked(device, session, -1, false, 0, 5)
	packet, err := lorawan.Parse(payload)
	if err != nil {
		t.Fatalf("parse generated OTAA join request: %v", err)
	}
	if packet.MType != lorawan.MTypeJoinRequest || packet.JoinEUI != 0x70B3D57ED0000001 || packet.DevEUI != 0x70B3D57ED0000001 {
		t.Fatalf("unexpected generated OTAA frame: %+v", packet)
	}
	if !lorawan.VerifyJoinRequestMIC(packet, mustAppKey(device)) {
		t.Fatal("generated OTAA frame has invalid MIC")
	}
}

func mustAppKey(device contracts.Device) []byte {
	key, err := lorawan.DecodeHex(device.OOTAConfig.AppKey)
	if err != nil {
		panic(err)
	}
	return key
}

func TestEngineDispatchesSuccessfulVirtualUplinkToExternalAdapter(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000083")
	configureABP(&device)
	device.PayloadConfig.UplinkInterval = 1
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c83")
	adapter := newVirtualUplinkTestAdapter()
	clock := types.NewManualClock(0)
	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device}, []contracts.Gateway{gateway},
		types.Options{Clock: clock, GatewayAdapterFactory: &virtualUplinkTestFactory{adapter: adapter}},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	if err := engine.Start(context.Background()); err != nil {
		t.Fatalf("start engine: %v", err)
	}
	defer stopEngine(t, engine)
	if err := clock.Advance(2 * time.Second); err != nil {
		t.Fatalf("advance simulation clock: %v", err)
	}

	select {
	case packet := <-adapter.uplinks:
		decoded, parseErr := lorawan.Parse(packet.Payload)
		if parseErr != nil || decoded.MType != lorawan.MTypeUnconfirmedDataUp {
			t.Fatalf("expected a valid ABP uplink, got packet=%+v err=%v", packet, parseErr)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for virtual gateway uplink")
	}
}

type virtualUplinkTestFactory struct {
	adapter *virtualUplinkTestAdapter
}

func (factory *virtualUplinkTestFactory) SupportsVirtualGateways() bool { return true }

func (factory *virtualUplinkTestFactory) NewGatewayAdapter(contracts.Gateway) (types.GatewayAdapter, error) {
	return factory.adapter, nil
}

type virtualUplinkTestAdapter struct {
	uplinks chan types.GatewayPacket
	events  chan types.GatewayAdapterEvent
	packets chan types.GatewayPacket
}

func newVirtualUplinkTestAdapter() *virtualUplinkTestAdapter {
	return &virtualUplinkTestAdapter{
		uplinks: make(chan types.GatewayPacket, 8),
		events:  make(chan types.GatewayAdapterEvent),
		packets: make(chan types.GatewayPacket),
	}
}

func (adapter *virtualUplinkTestAdapter) Start(context.Context) error { return nil }

func (adapter *virtualUplinkTestAdapter) Send(context.Context, types.GatewayPacket) error { return nil }

func (adapter *virtualUplinkTestAdapter) SendUplink(_ context.Context, packet types.GatewayPacket) error {
	adapter.uplinks <- packet
	return nil
}

func (adapter *virtualUplinkTestAdapter) Packets() <-chan types.GatewayPacket { return adapter.packets }

func (adapter *virtualUplinkTestAdapter) Events() <-chan types.GatewayAdapterEvent {
	return adapter.events
}

func (adapter *virtualUplinkTestAdapter) Close() error {
	close(adapter.events)
	close(adapter.packets)
	return nil
}

package runtime

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"testing"
	"time"
)

func TestExternalJoinAcceptFindsDeviceWithMatchingKey(t *testing.T) {
	a := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	b := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000002")
	b.DevEUI = "70B3D57ED0000002"
	cfg := *b.OOTAConfig
	cfg.AppKey = "11223344556677889900AABBCCDDEEFF"
	b.OOTAConfig = &cfg
	gw := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7205")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{a, b}, []contracts.Gateway{gw}, types.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, device := range []contracts.Device{a, b} {
		engine.sessions[device.ID].PendingJoinRequest = &types.PendingJoinRequest{PacketID: device.ID, GatewayIDs: []string{gw.ID}}
	}
	key, _ := lorawan.DecodeHex(cfg.AppKey)
	raw, err := lorawan.BuildJoinAccept(lorawan.JoinAcceptOptions{AppKey: key, AppNonce: 123, DevAddr: 0x26011bda})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := lorawan.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	events := engine.processExternalJoinAcceptLocked(types.GatewayPacket{GatewayID: gw.ID, Payload: raw}, parsed)
	if !engine.sessions[b.ID].Joined || engine.sessions[a.ID].Joined || !containsEventType(events, contracts.JoinAcceptReceived) {
		t.Fatalf("wrong join recipient: %+v", events)
	}
	expected, _, err := lorawan.DeriveSessionKeys(key, 123, 0, engine.sessions[b.ID].DevNonce)
	if err != nil || string(expected) != string(engine.sessions[b.ID].NwkSKey) {
		t.Fatal("wrong session key")
	}
}

func TestExternalACKTimingPreservesPendingUplinkAndCounters(t *testing.T) {
	for _, test := range []struct {
		name     string
		delay    time.Duration
		accepted bool
	}{
		{"expired", -time.Hour, false}, {"before_rx1", 500 * time.Millisecond, false}, {"rx1", time.Second, true}, {"rx2", 2 * time.Second, true}, {"after_windows", time.Hour, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
			configureABP(&device)
			device.RX1Config.Delay = intPtr(1000)
			device.RX1Config.Duration = intPtr(200)
			device.RX2Config.Delay = intPtr(2000)
			device.RX2Config.Duration = intPtr(200)
			gw := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7205")
			engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gw}, types.Options{})
			if err != nil {
				t.Fatal(err)
			}
			session := engine.sessions[device.ID]
			sentAt := time.Now()
			session.PendingUplink = &types.PendingUplink{PacketID: "current", Confirmed: true, FrameCounter: 2, FragmentCount: 1, SentAt: sentAt}
			raw, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{DevAddr: 0x26011bda, FCnt: 1, ACK: true, Direction: 1, NwkSKey: session.NwkSKey, AppSKey: session.AppSKey})
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := lorawan.Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			events := engine.processExternalDownlinkLocked(types.GatewayPacket{GatewayID: gw.ID, Payload: raw, TransmitAt: sentAt.Add(test.delay)}, parsed)
			if containsEventType(events, contracts.UplinkACKReceived) != test.accepted {
				t.Fatalf("unexpected ACK result: %+v", events)
			}
			if !test.accepted && (session.PendingUplink == nil || engine.metrics.AcknowledgedUplinks != 0 || session.ExternalDownlinkFrameCounterSet) {
				t.Fatal("rejected ACK changed transaction or replay counters")
			}
		})
	}
}

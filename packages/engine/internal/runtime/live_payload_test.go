package runtime

import (
	"bytes"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestLivePayloadUpdatePreservesRetriesFragmentsAndSession(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000981")
	configureABP(&device)
	device.PayloadConfig.Payload = strings.Repeat("A", 80)
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	device.PayloadConfig.OversizedPayloadBehavior = contracts.Fragment
	device.RX2Config.DataRate = intPtr(0)
	device.AdvancedConfig.UplinkDataRate = intPtr(0)
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, nil, types.Options{Clock: types.NewManualClock(0), EventBuffer: 256})
	if err != nil {
		t.Fatal(err)
	}
	engine.state.Status = contracts.SimulationStatusRunning
	engine.processUplinkLocked(types.ScheduledEvent{DeviceID: device.ID, Attempt: 1})
	session := engine.sessions[device.ID]
	pending := session.PendingUplink
	if pending == nil || pending.FragmentCount != 2 {
		t.Fatalf("missing fragmented transaction: %+v", pending)
	}
	first := append([]byte(nil), engine.radio[pending.PacketID].Payload...)
	updated := device
	updated.PayloadConfig.Payload = "bmV3"
	updated.PayloadConfig.Base64Encoded = true
	if err := engine.UpdateDevice(updated); err != nil {
		t.Fatal(err)
	}
	engine.processUplinkLocked(types.ScheduledEvent{DeviceID: device.ID, Attempt: 2, PacketID: pending.PacketID, FrameCounter: pending.FrameCounter})
	if !bytes.Equal(first, engine.radio[pending.PacketID].Payload) {
		t.Fatal("retry changed its original bytes")
	}
	pending.FragmentIndex = 1
	engine.processUplinkLocked(types.ScheduledEvent{DeviceID: device.ID, Attempt: 1, PacketID: pending.PacketID, FrameCounter: pending.FrameCounter, FragmentIndex: 1, FragmentCount: 2})
	packet, err := lorawan.Parse(engine.radio[pending.PacketID].Payload)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := lorawan.DecryptPayload(packet, session.AppSKey, 0)
	if err != nil || string(decoded) != strings.Repeat("A", 29) {
		t.Fatalf("remaining fragment changed: %q %v", decoded, err)
	}
	checkpoint := engine.Checkpoint()
	checkpoint.Sessions[device.ID].PendingUplink.PayloadConfig.Payload = "corrupted copy"
	if session.PendingUplink.PayloadConfig.Payload != device.PayloadConfig.Payload {
		t.Fatal("checkpoint aliases the frozen payload")
	}
	session.PendingUplink = nil
	engine.processUplinkLocked(types.ScheduledEvent{DeviceID: device.ID, Attempt: 1})
	packet, err = lorawan.Parse(engine.radio[session.PendingUplink.PacketID].Payload)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err = lorawan.DecryptPayload(packet, session.AppSKey, 0)
	if err != nil || string(decoded) != "new" || packet.FCnt != 2 {
		t.Fatalf("new message did not use updated bytes with existing session: %q FCnt=%d %v", decoded, packet.FCnt, err)
	}
}

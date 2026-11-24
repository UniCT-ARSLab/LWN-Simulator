package runtime

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func newExternalMACTestEngine(t *testing.T) (*Engine, contracts.Device, *types.DeviceSession) {
	t.Helper()
	d := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000997")
	configureABP(&d)
	e, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{d}, nil, types.Options{Clock: types.NewManualClock(0)})
	if err != nil {
		t.Fatal(err)
	}
	return e, d, e.sessions[d.ID]
}
func externalTestFrame(t *testing.T, s *types.DeviceSession, portZero bool, commands []byte, counter uint32, confirmed ...bool) lorawan.Packet {
	t.Helper()
	addr, _ := lorawan.DecodeHex(s.DeviceAddress)
	options := lorawan.DataFrameOptions{DevAddr: binary.BigEndian.Uint32(addr), FCnt: counter, Direction: 1, NwkSKey: s.NwkSKey, AppSKey: s.AppSKey}
	if len(confirmed) > 0 {
		options.Confirmed = confirmed[0]
	}
	if portZero {
		p := byte(0)
		options.FPort = &p
		options.Payload = commands
	} else {
		options.FOpts = commands
	}
	wire, err := lorawan.BuildDataFrame(options)
	if err != nil {
		t.Fatal(err)
	}
	packet, err := lorawan.Parse(wire)
	if err != nil {
		t.Fatal(err)
	}
	return packet
}
func newChannelCommand(index byte, frequency int64) []byte {
	f := frequency / 100
	return []byte{7, index, byte(f), byte(f >> 8), byte(f >> 16), 0x50}
}

func TestExternalMACNegotiationInFOptsAndEncryptedPortZero(t *testing.T) {
	for _, portZero := range []bool{false, true} {
		t.Run(map[bool]string{false: "FOpts", true: "FPort0"}[portZero], func(t *testing.T) {
			e, d, s := newExternalMACTestEngine(t)
			commands := append(newChannelCommand(3, 867100000), 3, 0x51, 15, 0, 1)
			packet := externalTestFrame(t, s, portZero, commands, 1, true)
			events := e.processExternalDownlinkLocked(types.GatewayPacket{GatewayID: "bridge"}, packet)
			if len(events) == 0 || events[0].Type != contracts.DeviceDownlinkTransmitted {
				t.Fatalf("downlink rejected: %+v", events)
			}
			if s.NetworkChannels[3].Frequency != 867100000 || s.CurrentDataRate != 5 || s.CurrentTxPower != 14 || !s.ExternalADR {
				t.Fatalf("MAC configuration not applied: %+v", s)
			}
			if ch := selectRadioChannelAtDataRate(d, s, 3, 5); ch.Frequency != 867100000 {
				t.Fatalf("configured channel not used: %+v", ch)
			}
			s.PendingUplink = &types.PendingUplink{FrameCounter: 2, ReservedFOpts: uplinkMACReserve(s)}
			first := e.buildUplinkPayloadLocked(d, s, 2, true, 0, 5)
			up, err := lorawan.Parse(first)
			if err != nil || !bytes.Equal(up.FOpts, []byte{7, 3, 3, 7}) || !lorawan.VerifyDataMIC(up, s.NwkSKey, 0) || !up.ACK {
				t.Fatalf("invalid MAC response: %+v %v", up, err)
			}
			s.PendingMACResponses = append(s.PendingMACResponses, []byte{6, 255, 1})
			if retry := e.buildUplinkPayloadLocked(d, s, 2, true, 0, 5); !bytes.Equal(first, retry) || len(s.PendingMACResponses) != 1 {
				t.Fatal("retry changed or consumed new responses")
			}
			s.PendingUplink = nil
			next, _ := lorawan.Parse(e.buildUplinkPayloadLocked(d, s, 3, false, 0, 5))
			if !bytes.Equal(next.FOpts, []byte{6, 255, 1}) {
				t.Fatalf("next response lost: %x", next.FOpts)
			}
			// Replayed downlinks must not apply or enqueue responses again.
			before := len(s.PendingMACResponses)
			e.processExternalDownlinkLocked(types.GatewayPacket{GatewayID: "bridge"}, packet)
			if len(s.PendingMACResponses) != before {
				t.Fatal("replay applied twice")
			}
		})
	}
}

func TestMalformedExternalMACDoesNotMutateSession(t *testing.T) {
	for _, wire := range [][]byte{{7, 3}, {6, 0xff}, {3, 0x51, 7}} {
		e, _, s := newExternalMACTestEngine(t)
		packet := externalTestFrame(t, s, false, wire, 1)
		events := e.processExternalDownlinkLocked(types.GatewayPacket{GatewayID: "bridge"}, packet)
		if len(events) != 1 || events[0].Type != contracts.DeviceDownlinkDropped || len(s.PendingMACResponses) != 0 || len(s.NetworkChannels) != 0 || s.ExternalDownlinkFrameCounterSet {
			t.Fatalf("invalid MAC changed session: %+v", events)
		}
	}
	e, _, s := newExternalMACTestEngine(t)
	packet := externalTestFrame(t, s, false, []byte{6}, 1)
	packet.Raw[len(packet.Raw)-1] ^= 1
	packet, _ = lorawan.Parse(packet.Raw)
	e.processExternalDownlinkLocked(types.GatewayPacket{GatewayID: "bridge"}, packet)
	if len(s.PendingMACResponses) != 0 {
		t.Fatal("invalid MIC applied MAC commands")
	}
}

func TestExternalMACRejectsInvalidParametersAndPreservesCheckpoint(t *testing.T) {
	e, d, s := newExternalMACTestEngine(t)
	wire := append(newChannelCommand(3, 867100000), newChannelCommand(3, 867300000)...)
	commands, _ := lorawan.DecodeMACCommands(wire)
	e.applyExternalMACLocked(d, s, commands)
	if len(s.NetworkChannels) != 1 || s.NetworkChannels[3].Frequency != 867300000 {
		t.Fatal("channel update duplicated its entry")
	}
	commands, _ = lorawan.DecodeMACCommands(newChannelCommand(4, 900000000))
	e.applyExternalMACLocked(d, s, commands)
	if s.NetworkChannels[4].Frequency != 0 || !bytes.Equal(s.PendingMACResponses[2], []byte{7, 2}) {
		t.Fatal("invalid frequency acknowledged")
	}
	checkpoint := e.Checkpoint()
	cp := checkpoint.Sessions[d.ID]
	cp.PendingMACResponses[0][1] = 0
	cp.NetworkChannels[3] = types.NetworkChannel{}
	if s.PendingMACResponses[0][1] != 3 || s.NetworkChannels[3].Frequency == 0 {
		t.Fatal("checkpoint aliases MAC state")
	}
}

func TestMACResponsesRespectFrameLimitsAndFragmentBoundaries(t *testing.T) {
	e, d, s := newExternalMACTestEngine(t)
	d.PayloadConfig.Payload = string(bytes.Repeat([]byte{'A'}, 120))
	d.PayloadConfig.OversizedPayloadBehavior = contracts.Fragment
	d.AdvancedConfig.UplinkDataRate = intPtr(0)
	devices := []contracts.Device{d}
	registry, err := NewRegistry(devices, nil)
	if err != nil {
		t.Fatal(err)
	}
	e.registry = registry
	s.CurrentDataRate = 0
	s.PendingMACResponses = [][]byte{{7, 3}, {3, 7}}
	e.processUplinkLocked(types.ScheduledEvent{DeviceID: d.ID, Attempt: 1})
	pending := s.PendingUplink
	if pending == nil || pending.FragmentCount != 3 || pending.ReservedFOpts != 4 {
		t.Fatalf("missing capacity reservation: %+v", pending)
	}
	var joined []byte
	for i := 0; i < 3; i++ {
		if i > 0 {
			pending.FragmentIndex = i
			e.processUplinkLocked(types.ScheduledEvent{DeviceID: d.ID, Attempt: 1, PacketID: pending.PacketID, FrameCounter: pending.FrameCounter, FragmentIndex: i, FragmentCount: 3})
		}
		wire := e.radio[pending.PacketID].Payload
		p, err := lorawan.Parse(wire)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.FRMPayload)+len(p.FOpts) > maximumPayloadSize(d.LocationConfig.Region, 0) {
			t.Fatal("MAC responses exceeded regional frame capacity")
		}
		payload, err := lorawan.DecryptPayload(p, s.AppSKey, 0)
		if err != nil {
			t.Fatal(err)
		}
		joined = append(joined, payload...)
	}
	if string(joined) != d.PayloadConfig.Payload {
		t.Fatalf("fragmented payload lost bytes: %d", len(joined))
	}
	checkpoint := e.Checkpoint()
	checkpoint.Sessions[d.ID].PendingUplink.WirePayload[0] ^= 1
	if bytes.Equal(checkpoint.Sessions[d.ID].PendingUplink.WirePayload, s.PendingUplink.WirePayload) {
		t.Fatal("checkpoint aliases cached wire frame")
	}
}

func TestMACResponsesSplitAtCommandBoundariesAndADRBlockIsAtomic(t *testing.T) {
	e, d, s := newExternalMACTestEngine(t)
	for i := 0; i < 9; i++ {
		s.PendingMACResponses = append(s.PendingMACResponses, []byte{7, 3})
	}
	p, err := lorawan.Parse(e.buildUplinkPayloadLocked(d, s, 1, false, 0, 5))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.FOpts) != 14 || len(s.PendingMACResponses) != 2 {
		t.Fatalf("responses split incorrectly: %x %+v", p.FOpts, s.PendingMACResponses)
	}
	s.PendingMACResponses = nil
	beforeRate, beforePower := s.CurrentDataRate, s.CurrentTxPower
	commands, _ := lorawan.DecodeMACCommands([]byte{3, 0x21, 7, 0, 1, 3, 0x01, 0, 0, 1})
	e.applyExternalMACLocked(d, s, commands)
	if s.CurrentDataRate != beforeRate || s.CurrentTxPower != beforePower || len(s.PendingMACResponses) != 1 || s.PendingMACResponses[0][1]&1 != 0 {
		t.Fatal("partially applied invalid ADR block")
	}
}

func TestExternalChannelMasksCoverDynamicAndFixedPlans(t *testing.T) {
	_, d, s := newExternalMACTestEngine(t)
	if _, ok := externalChannelMask(d, s, 0, 0); ok {
		t.Fatal("empty channel mask accepted")
	}
	if _, ok := externalChannelMask(d, s, 8, 0); ok {
		t.Fatal("undefined dynamic channel accepted")
	}
	for _, region := range []contracts.DeviceRegion{contracts.US915, contracts.AU915} {
		d.LocationConfig.Region = region
		mask, ok := externalChannelMask(d, s, 1, 7)
		if !ok || mask[64] || !mask[0] || !mask[65] {
			t.Fatalf("wideband-only mask incorrect for %s: %+v %v", region, mask, ok)
		}
		if _, ok := externalChannelMask(d, s, 0xffff, 4); ok {
			t.Fatal("reserved mask bits accepted")
		}
	}
}

func TestExternalADRDoesNotDisableUnrelatedInternalNetwork(t *testing.T) {
	e, d, s := newExternalMACTestEngine(t)
	d.AdvancedConfig.ADREnabled = true
	s.CurrentDataRate = 0
	s.LastSNR = 20
	e.gatewayAdapters["unrelated-real-gateway"] = newFakeGatewayAdapter()
	e.applyADRLocked(d, s)
	if s.CurrentDataRate != 1 {
		t.Fatal("unrelated external gateway disabled internal ADR")
	}
	s.CurrentDataRate = 0
	s.ExternalADR = true
	e.applyADRLocked(d, s)
	if s.CurrentDataRate != 0 {
		t.Fatal("internal ADR overrode network-server control")
	}
}

func TestExternalMACParametersAffectRuntimeAndReply(t *testing.T) {
	frequency := int64(869525000) / 100
	freq := []byte{byte(frequency), byte(frequency >> 8), byte(frequency >> 16)}
	cases := []struct {
		name         string
		wire, answer []byte
		check        func(*types.DeviceSession) bool
	}{
		{"duty cycle", []byte{4, 3}, []byte{4}, func(s *types.DeviceSession) bool { return s.MaximumDutyCycle == 0.125 }},
		{"RX parameters", append([]byte{5, 0x10}, freq...), []byte{5, 7}, func(s *types.DeviceSession) bool {
			return s.RX1DataRateOffset == 1 && s.RX2Frequency == 869525000 && s.RX2DataRate == 0
		}},
		{"status", []byte{6}, []byte{6, 255, 0}, func(s *types.DeviceSession) bool { return true }},
		{"RX timing", []byte{8, 0}, []byte{8}, func(s *types.DeviceSession) bool { return s.ReceiveDelay == time.Second }},
		{"TX parameters", []byte{9, 0x35}, []byte{9}, func(s *types.DeviceSession) bool {
			return s.UplinkDwellTime && s.DownlinkDwellTime && s.MaximumEIRP == 5
		}},
		{"RX1 channel", append([]byte{10, 0}, freq...), []byte{10, 3}, func(s *types.DeviceSession) bool { return s.NetworkChannels[0].DownlinkFrequency == 869525000 }},
		{"ping slot", append(append([]byte{17}, freq...), 3), []byte{17, 3}, func(s *types.DeviceSession) bool { return s.PingSlotFrequency == 869525000 && s.PingSlotDataRate == 3 }},
		{"beacon", append([]byte{19}, freq...), []byte{19, 1}, func(s *types.DeviceSession) bool { return s.BeaconFrequency == 869525000 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, d, s := newExternalMACTestEngine(t)
			commands, err := lorawan.DecodeMACCommands(c.wire)
			if err != nil {
				t.Fatal(err)
			}
			e.applyExternalMACLocked(d, s, commands)
			if !c.check(s) || len(s.PendingMACResponses) != 1 || !bytes.Equal(s.PendingMACResponses[0], c.answer) {
				t.Fatalf("MAC effect or response missing: %+v", s)
			}
		})
	}
}

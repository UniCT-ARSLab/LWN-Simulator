package lorawan

import (
	"errors"
	"testing"

	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestDecodeDownlinkMACCommands(t *testing.T) {
	cases := []struct {
		wire []byte
		kind types.MACCommandType
	}{
		{[]byte{2, 10, 2}, types.MACLinkCheckAns}, {[]byte{3, 0x51, 7, 0, 0}, types.MACLinkADRReq},
		{[]byte{4, 3}, types.MACDutyCycleReq}, {[]byte{5, 0x10, 0xd2, 0xad, 0x84}, types.MACRXParamSetupReq},
		{[]byte{6}, types.MACDevStatusReq}, {[]byte{7, 3, 0x18, 0x4f, 0x84, 0x50}, types.MACNewChannelReq},
		{[]byte{8, 0}, types.MACRXTimingSetupReq}, {[]byte{9, 0x35}, types.MACTXParamSetupReq},
		{[]byte{10, 3, 0x18, 0x4f, 0x84}, types.MACDLChannelReq}, {[]byte{13, 0, 0, 0, 1, 128}, types.MACDeviceTimeAns},
		{[]byte{16}, types.MACPingSlotInfoAns}, {[]byte{17, 0xd2, 0xad, 0x84, 3}, types.MACPingSlotChannelReq},
		{[]byte{19, 0xd2, 0xad, 0x84}, types.MACBeaconFreqReq},
	}
	var combined []byte
	for _, c := range cases {
		t.Run(string(c.kind), func(t *testing.T) {
			commands, err := DecodeMACCommands(c.wire)
			if err != nil || len(commands) != 1 || commands[0].Type != c.kind {
				t.Fatalf("decode: %+v %v", commands, err)
			}
			for n := 1; n < len(c.wire); n++ {
				if partial, err := DecodeMACCommands(c.wire[:n]); !errors.Is(err, ErrInvalidFrame) || partial != nil {
					t.Fatalf("truncated command accepted at %d: %+v %v", n, partial, err)
				}
			}
		})
		combined = append(combined, c.wire...)
	}
	commands, err := DecodeMACCommands(combined)
	if err != nil || len(commands) != len(cases) {
		t.Fatalf("combined stream: %+v %v", commands, err)
	}
	if partial, err := DecodeMACCommands(append(combined, 0xff)); !errors.Is(err, ErrInvalidFrame) || partial != nil {
		t.Fatal("unknown CID returned partial commands")
	}
}

func FuzzDecodeMACCommands(f *testing.F) {
	for _, data := range [][]byte{{}, {6}, {7, 3, 0x18, 0x4f, 0x84, 0x50}, {3, 0x51, 7, 0, 1}, {0xff}} {
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		commands, err := DecodeMACCommands(data)
		if err != nil && commands != nil {
			t.Fatal("invalid stream returned partial commands")
		}
	})
}

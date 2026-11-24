package lorawan

import (
	"testing"
	"time"

	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestEncodeMACCommandsProducesLoRaWANFOpts(t *testing.T) {
	dataRate, power, repetitions := 3, 1, 2
	frequency := int64(869_525_000)
	commands, err := EncodeMACCommands([]types.MACCommand{
		{Type: types.MACLinkADRReq, DataRate: &dataRate, TxPower: &power, NbTrans: &repetitions},
		{Type: types.MACBeaconFreqReq, Frequency: &frequency},
	})
	if err != nil {
		t.Fatalf("encode MAC commands: %v", err)
	}
	if len(commands) != 9 || commands[0] != 0x03 || commands[5] != 0x13 {
		t.Fatalf("unexpected encoded commands: %x", commands)
	}
}

func TestEverySupportedMACCommandEncodes(t *testing.T) {
	zero, one, two, three, four, five := 0, 1, 2, 3, 4, 5
	mask := uint16(0x00ff)
	frequency := int64(869_525_000)
	delay := 2 * time.Second
	enabled := true
	now := time.Date(2026, time.October, 4, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		command types.MACCommand
		cid     byte
	}{
		{types.MACCommand{Type: types.MACLinkCheckAns, Margin: &five, GatewayCount: &two}, 0x02},
		{types.MACCommand{Type: types.MACLinkADRReq, DataRate: &three, TxPower: &one, NbTrans: &two, ChannelMask: &mask, ChannelMaskControl: &zero}, 0x03},
		{types.MACCommand{Type: types.MACDutyCycleReq, MaxDutyCycleExponent: &four}, 0x04},
		{types.MACCommand{Type: types.MACRXParamSetupReq, DataRate: &three, RX1DataRateOffset: &one, Frequency: &frequency}, 0x05},
		{types.MACCommand{Type: types.MACDevStatusReq}, 0x06},
		{types.MACCommand{Type: types.MACNewChannelReq, ChannelIndex: &five, Frequency: &frequency, MinimumDataRate: &zero, MaximumDataRate: &five}, 0x07},
		{types.MACCommand{Type: types.MACRXTimingSetupReq, Delay: &delay}, 0x08},
		{types.MACCommand{Type: types.MACTXParamSetupReq, UplinkDwellTime: &enabled, DownlinkDwellTime: &enabled, MaximumEIRP: &five}, 0x09},
		{types.MACCommand{Type: types.MACDLChannelReq, ChannelIndex: &five, Frequency: &frequency}, 0x0a},
		{types.MACCommand{Type: types.MACDeviceTimeAns, DeviceTime: &now}, 0x0d},
		{types.MACCommand{Type: types.MACPingSlotInfoAns}, 0x10},
		{types.MACCommand{Type: types.MACPingSlotChannelReq, Frequency: &frequency, DataRate: &three}, 0x11},
		{types.MACCommand{Type: types.MACBeaconFreqReq, Frequency: &frequency}, 0x13},
	}
	for _, test := range tests {
		encoded, err := EncodeMACCommands([]types.MACCommand{test.command})
		if err != nil {
			t.Fatalf("encode %s: %v", test.command.Type, err)
		}
		if len(encoded) == 0 || encoded[0] != test.cid {
			t.Fatalf("unexpected %s encoding: %x", test.command.Type, encoded)
		}
	}
}

func TestMACCommandEncodingRejectsInvalidValues(t *testing.T) {
	beforeGPSEpoch := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	if _, err := EncodeMACCommands([]types.MACCommand{{Type: types.MACDeviceTimeAns, DeviceTime: &beforeGPSEpoch}}); err == nil {
		t.Fatal("expected pre-GPS device time to fail")
	}
	invalidFrequency := int64(1)
	if _, err := EncodeMACCommands([]types.MACCommand{{Type: types.MACBeaconFreqReq, Frequency: &invalidFrequency}}); err == nil {
		t.Fatal("expected non-representable frequency to fail")
	}
	if _, err := EncodeMACCommands([]types.MACCommand{{Type: types.MACCommandType("unknown")}}); err == nil {
		t.Fatal("expected unsupported command to fail")
	}
}

func TestEncodeMACCommandsEnforcesFOptsLimit(t *testing.T) {
	frequency := int64(869_525_000)
	_, err := EncodeMACCommands([]types.MACCommand{
		{Type: types.MACBeaconFreqReq, Frequency: &frequency},
		{Type: types.MACBeaconFreqReq, Frequency: &frequency},
		{Type: types.MACBeaconFreqReq, Frequency: &frequency},
		{Type: types.MACBeaconFreqReq, Frequency: &frequency},
	})
	if err == nil {
		t.Fatal("expected FOpts overflow to fail")
	}
}

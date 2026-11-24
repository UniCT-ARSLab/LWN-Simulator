package lorawan

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

var downlinkMACSizes = map[byte]int{2: 2, 3: 4, 4: 1, 5: 4, 6: 0, 7: 5, 8: 1, 9: 1, 10: 4, 13: 5, 16: 0, 17: 4, 19: 3}

// DecodeMACCommands decodes a complete LoRaWAN 1.0.x downlink MAC-command stream.
// Unknown CIDs and truncated commands fail without returning partially decoded commands.
func DecodeMACCommands(data []byte) ([]types.MACCommand, error) {
	var commands []types.MACCommand
	for len(data) > 0 {
		cid := data[0]
		data = data[1:]
		size, ok := downlinkMACSizes[cid]
		if !ok || len(data) < size {
			return nil, fmt.Errorf("%w: unknown or truncated downlink MAC command 0x%02x", ErrInvalidFrame, cid)
		}
		b := data[:size]
		data = data[size:]
		var c types.MACCommand
		integer := func(v int) *int { return &v }
		freq := func(b []byte) *int64 { v := int64(uint32(b[0])|uint32(b[1])<<8|uint32(b[2])<<16) * 100; return &v }
		switch cid {
		case 2:
			c.Type = types.MACLinkCheckAns
			c.Margin = integer(int(b[0]))
			c.GatewayCount = integer(int(b[1]))
		case 3:
			c.Type = types.MACLinkADRReq
			c.DataRate = integer(int(b[0] >> 4))
			c.TxPower = integer(int(b[0] & 15))
			mask := binary.LittleEndian.Uint16(b[1:3])
			c.ChannelMask = &mask
			c.ChannelMaskControl = integer(int(b[3] >> 4 & 7))
			c.NbTrans = integer(int(b[3] & 15))
		case 4:
			c.Type = types.MACDutyCycleReq
			c.MaxDutyCycleExponent = integer(int(b[0] & 15))
		case 5:
			c.Type = types.MACRXParamSetupReq
			c.RX1DataRateOffset = integer(int(b[0] >> 4 & 7))
			c.DataRate = integer(int(b[0] & 15))
			c.Frequency = freq(b[1:])
		case 6:
			c.Type = types.MACDevStatusReq
		case 7:
			c.Type = types.MACNewChannelReq
			c.ChannelIndex = integer(int(b[0]))
			c.Frequency = freq(b[1:4])
			c.MinimumDataRate = integer(int(b[4] & 15))
			c.MaximumDataRate = integer(int(b[4] >> 4))
		case 8:
			c.Type = types.MACRXTimingSetupReq
			delay := time.Duration(b[0]&15) * time.Second
			if delay == 0 {
				delay = time.Second
			}
			c.Delay = &delay
		case 9:
			c.Type = types.MACTXParamSetupReq
			up, down := b[0]&16 != 0, b[0]&32 != 0
			c.UplinkDwellTime = &up
			c.DownlinkDwellTime = &down
			c.MaximumEIRP = integer(int(b[0] & 15))
		case 10:
			c.Type = types.MACDLChannelReq
			c.ChannelIndex = integer(int(b[0]))
			c.Frequency = freq(b[1:])
		case 13:
			c.Type = types.MACDeviceTimeAns
			t := time.Date(1980, 1, 6, 0, 0, 0, 0, time.UTC).Add(time.Duration(binary.LittleEndian.Uint32(b[:4]))*time.Second + time.Duration(b[4])*time.Second/256)
			c.DeviceTime = &t
		case 16:
			c.Type = types.MACPingSlotInfoAns
		case 17:
			c.Type = types.MACPingSlotChannelReq
			c.Frequency = freq(b[:3])
			c.DataRate = integer(int(b[3] & 15))
		case 19:
			c.Type = types.MACBeaconFreqReq
			c.Frequency = freq(b)
		}
		commands = append(commands, c)
	}
	return commands, nil
}

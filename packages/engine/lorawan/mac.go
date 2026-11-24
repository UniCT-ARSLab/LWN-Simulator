package lorawan

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

const (
	macCIDLinkCheck       byte = 0x02
	macCIDLinkADR         byte = 0x03
	macCIDDutyCycle       byte = 0x04
	macCIDRXParamSetup    byte = 0x05
	macCIDDevStatus       byte = 0x06
	macCIDNewChannel      byte = 0x07
	macCIDRXTimingSetup   byte = 0x08
	macCIDTXParamSetup    byte = 0x09
	macCIDDLChannel       byte = 0x0a
	macCIDDeviceTime      byte = 0x0d
	macCIDPingSlotInfo    byte = 0x10
	macCIDPingSlotChannel byte = 0x11
	macCIDBeaconFreq      byte = 0x13
)

// EncodeMACCommands encodes LoRaWAN 1.0.x downlink MAC commands for FOpts.
// The engine validates semantic ranges before calling this function; this
// boundary still rejects values that cannot be represented on the wire.
func EncodeMACCommands(commands []types.MACCommand) ([]byte, error) {
	encoded := make([]byte, 0, 15)
	for _, command := range commands {
		value, err := encodeMACCommand(command)
		if err != nil {
			return nil, err
		}
		if len(encoded)+len(value) > 15 {
			return nil, fmt.Errorf("%w: MAC commands exceed the 15-byte FOpts limit", ErrInvalidFrame)
		}
		encoded = append(encoded, value...)
	}
	return encoded, nil
}

func encodeMACCommand(command types.MACCommand) ([]byte, error) {
	switch command.Type {
	case types.MACLinkCheckAns:
		return []byte{macCIDLinkCheck, byte(intPointer(command.Margin)), byte(intPointer(command.GatewayCount))}, nil
	case types.MACLinkADRReq:
		mask := uint16(0xffff)
		if command.ChannelMask != nil {
			mask = *command.ChannelMask
		}
		result := []byte{macCIDLinkADR, byte(nibble(command.DataRate)<<4 | nibble(command.TxPower))}
		maskBytes := make([]byte, 2)
		binary.LittleEndian.PutUint16(maskBytes, mask)
		result = append(result, maskBytes...)
		return append(result, byte(nibble(command.ChannelMaskControl)<<4|nibble(command.NbTrans))), nil
	case types.MACDutyCycleReq:
		return []byte{macCIDDutyCycle, byte(nibble(command.MaxDutyCycleExponent))}, nil
	case types.MACRXParamSetupReq:
		frequency, err := encodeFrequency(command.Frequency)
		if err != nil {
			return nil, err
		}
		return append([]byte{macCIDRXParamSetup, byte(threeBits(command.RX1DataRateOffset)<<4 | nibble(command.DataRate))}, frequency...), nil
	case types.MACDevStatusReq:
		return []byte{macCIDDevStatus}, nil
	case types.MACNewChannelReq:
		frequency, err := encodeFrequency(command.Frequency)
		if err != nil {
			return nil, err
		}
		return append(append([]byte{macCIDNewChannel, byte(intPointer(command.ChannelIndex))}, frequency...), byte(nibble(command.MaximumDataRate)<<4|nibble(command.MinimumDataRate))), nil
	case types.MACRXTimingSetupReq:
		delay := 0
		if command.Delay != nil {
			delay = int(*command.Delay / time.Second)
		}
		return []byte{macCIDRXTimingSetup, byte(delay & 0x0f)}, nil
	case types.MACTXParamSetupReq:
		value := byte(nibble(command.MaximumEIRP))
		if boolPointer(command.DownlinkDwellTime) {
			value |= 0x20
		}
		if boolPointer(command.UplinkDwellTime) {
			value |= 0x10
		}
		return []byte{macCIDTXParamSetup, value}, nil
	case types.MACDLChannelReq:
		frequency, err := encodeFrequency(command.Frequency)
		if err != nil {
			return nil, err
		}
		return append([]byte{macCIDDLChannel, byte(intPointer(command.ChannelIndex))}, frequency...), nil
	case types.MACDeviceTimeAns:
		if command.DeviceTime == nil {
			return nil, fmt.Errorf("%w: device-time-ans requires device time", ErrInvalidFrame)
		}
		gpsEpoch := time.Date(1980, time.January, 6, 0, 0, 0, 0, time.UTC)
		delta := command.DeviceTime.UTC().Sub(gpsEpoch)
		if delta < 0 {
			return nil, fmt.Errorf("%w: device time precedes GPS epoch", ErrInvalidFrame)
		}
		result := make([]byte, 6)
		result[0] = macCIDDeviceTime
		binary.LittleEndian.PutUint32(result[1:5], uint32(delta/time.Second))
		result[5] = byte((delta % time.Second) * 256 / time.Second)
		return result, nil
	case types.MACPingSlotInfoAns:
		return []byte{macCIDPingSlotInfo}, nil
	case types.MACPingSlotChannelReq:
		frequency, err := encodeFrequency(command.Frequency)
		if err != nil {
			return nil, err
		}
		return append(append([]byte{macCIDPingSlotChannel}, frequency...), byte(nibble(command.DataRate))), nil
	case types.MACBeaconFreqReq:
		frequency, err := encodeFrequency(command.Frequency)
		if err != nil {
			return nil, err
		}
		return append([]byte{macCIDBeaconFreq}, frequency...), nil
	default:
		return nil, fmt.Errorf("%w: unsupported MAC command %q", ErrInvalidFrame, command.Type)
	}
}

func encodeFrequency(value *int64) ([]byte, error) {
	if value == nil || *value <= 0 || *value%100 != 0 || *value/100 > 0xffffff {
		return nil, fmt.Errorf("%w: MAC command frequency must be a positive 100 Hz multiple representable in 24 bits", ErrInvalidFrame)
	}
	frequency := uint32(*value / 100)
	return []byte{byte(frequency), byte(frequency >> 8), byte(frequency >> 16)}, nil
}

func nibble(value *int) int {
	return intPointer(value) & 0x0f
}

func threeBits(value *int) int {
	return intPointer(value) & 0x07
}

func intPointer(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func boolPointer(value *bool) bool {
	return value != nil && *value
}

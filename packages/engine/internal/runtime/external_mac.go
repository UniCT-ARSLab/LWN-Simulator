package runtime

import (
	"fmt"
	"math"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func externalMACCommands(packet lorawan.Packet, payload []byte) ([]types.MACCommand, error) {
	data := packet.FOpts
	if packet.FPort != nil && *packet.FPort == 0 {
		if len(data) > 0 {
			return nil, fmt.Errorf("MAC commands cannot occupy both FOpts and FPort 0")
		}
		data = payload
	}
	return lorawan.DecodeMACCommands(data)
}

func (e *Engine) applyExternalMACLocked(device contracts.Device, session *types.DeviceSession, commands []types.MACCommand) {
	for i := 0; i < len(commands); i++ {
		c := commands[i]
		if c.Type == types.MACLinkADRReq {
			staged := *session
			staged.DisabledChannels = cloneSessionMap(session.DisabledChannels)
			status := byte(7)
			for {
				status &= applyExternalLinkADR(device, &staged, commands[i])
				if i+1 >= len(commands) || commands[i+1].Type != types.MACLinkADRReq {
					break
				}
				i++
			}
			if status == 7 {
				session.CurrentDataRate = staged.CurrentDataRate
				session.CurrentSpreadingFactor = staged.CurrentSpreadingFactor
				session.CurrentTxPower = staged.CurrentTxPower
				session.UnconfirmedRepetitions = staged.UnconfirmedRepetitions
				session.DisabledChannels = staged.DisabledChannels
				session.ExternalADR = true
			}
			session.PendingMACResponses = append(session.PendingMACResponses, []byte{3, status})
			continue
		}
		var response []byte
		apply := false
		switch c.Type {
		case types.MACNewChannelReq:
			status := byte(3)
			index := *c.ChannelIndex
			if !externalFrequencyValid(device.LocationConfig.Region, *c.Frequency) && *c.Frequency != 0 {
				status &^= 1
			}
			if index < len(regionalChannels(device.LocationConfig.Region, 0)) || index > 15 || device.LocationConfig.Region == contracts.US915 || device.LocationConfig.Region == contracts.AU915 {
				status &^= 1
			}
			if *c.MinimumDataRate > *c.MaximumDataRate || !regional.SupportsUplinkDataRate(device.LocationConfig.Region, *c.MinimumDataRate) || !regional.SupportsUplinkDataRate(device.LocationConfig.Region, *c.MaximumDataRate) {
				status &^= 2
			}
			if status == 3 {
				if session.NetworkChannels == nil {
					session.NetworkChannels = make(map[int]types.NetworkChannel)
				}
				session.NetworkChannels[index] = types.NetworkChannel{Frequency: *c.Frequency, MinimumDataRate: *c.MinimumDataRate, MaximumDataRate: *c.MaximumDataRate}
			}
			response = []byte{7, status}
		case types.MACRXParamSetupReq:
			status := byte(7)
			if !externalFrequencyValid(device.LocationConfig.Region, *c.Frequency) {
				status &^= 1
			}
			if _, ok := regional.MustPlan(device.LocationConfig.Region).DataRates[*c.DataRate]; !ok {
				status &^= 2
			}
			maxOffset := 5
			if device.LocationConfig.Region == contracts.US915 {
				maxOffset = 3
			}
			if *c.RX1DataRateOffset > maxOffset {
				status &^= 4
			}
			apply = status == 7
			response = []byte{5, status}
		case types.MACDevStatusReq:
			margin := int(math.Round(session.LastSNR))
			margin = max(-32, min(31, margin))
			response = []byte{6, 255, byte(margin) & 63}
		case types.MACDutyCycleReq:
			apply = true
			response = []byte{4}
		case types.MACRXTimingSetupReq:
			apply = true
			response = []byte{8}
		case types.MACTXParamSetupReq:
			apply = true
			response = []byte{9}
		case types.MACDLChannelReq:
			ch, ok := externalChannel(device, session, *c.ChannelIndex)
			status := byte(3)
			if !ok {
				status &^= 2
			}
			if !externalFrequencyValid(device.LocationConfig.Region, *c.Frequency) {
				status &^= 1
			}
			if status == 3 {
				ch.DownlinkFrequency = *c.Frequency
				if session.NetworkChannels == nil {
					session.NetworkChannels = make(map[int]types.NetworkChannel)
				}
				session.NetworkChannels[*c.ChannelIndex] = ch
			}
			response = []byte{10, status}
		case types.MACPingSlotChannelReq:
			status := byte(3)
			if !externalFrequencyValid(device.LocationConfig.Region, *c.Frequency) {
				status &^= 1
			}
			if _, ok := regional.MustPlan(device.LocationConfig.Region).DataRates[*c.DataRate]; !ok {
				status &^= 2
			}
			apply = status == 3
			response = []byte{17, status}
		case types.MACBeaconFreqReq:
			status := byte(1)
			if !externalFrequencyValid(device.LocationConfig.Region, *c.Frequency) {
				status = 0
			}
			apply = status == 1
			response = []byte{19, status}
		default:
			apply = true
		}
		if apply {
			applyMACCommands(device, session, []types.MACCommand{c})
		}
		if len(response) > 0 {
			session.PendingMACResponses = append(session.PendingMACResponses, response)
		}
	}
}

func externalFrequencyValid(region contracts.DeviceRegion, frequency int64) bool {
	bounds := map[contracts.DeviceRegion][2]int64{contracts.EU868: {863000000, 870000000}, contracts.EU433: {433050000, 434790000}, contracts.CN779: {779000000, 787000000}, contracts.CN470: {470000000, 510000000}, contracts.AS923: {915000000, 928000000}, contracts.IN865: {865000000, 867000000}, contracts.KR920: {920900000, 923300000}, contracts.RU864: {864000000, 870000000}, contracts.US915: {902000000, 928000000}, contracts.AU915: {915000000, 928000000}}
	b, ok := bounds[region]
	return ok && frequency >= b[0] && frequency <= b[1]
}

func externalChannelTable(device contracts.Device, session *types.DeviceSession) map[int]types.NetworkChannel {
	table := make(map[int]types.NetworkChannel)
	index := 0
	for _, group := range regional.MustPlan(device.LocationConfig.Region).UplinkChannels {
		for j := 0; j < group.ChannelCount; j++ {
			table[index] = types.NetworkChannel{Frequency: group.InitialFrequency + int64(j)*group.FrequencyStep, MinimumDataRate: group.MinimumDataRate, MaximumDataRate: group.MaximumDataRate}
			index++
		}
	}
	for i, ch := range session.NetworkChannels {
		if ch.Frequency > 0 {
			table[i] = ch
		} else {
			delete(table, i)
		}
	}
	return table
}
func externalChannel(device contracts.Device, session *types.DeviceSession, index int) (types.NetworkChannel, bool) {
	ch, ok := externalChannelTable(device, session)[index]
	return ch, ok
}

func externalChannelMask(device contracts.Device, session *types.DeviceSession, mask uint16, control int) (map[int]bool, bool) {
	table := externalChannelTable(device, session)
	result := cloneSessionMap(session.DisabledChannels)
	if result == nil {
		result = make(map[int]bool)
	}
	fixed := device.LocationConfig.Region == contracts.US915 || device.LocationConfig.Region == contracts.AU915
	if fixed {
		if control > 7 || control == 5 || control == 4 && mask > 255 {
			return nil, false
		}
		for i := 0; i < 72; i++ {
			switch {
			case control <= 4 && i/16 == control:
				result[i] = mask&(1<<uint(i%16)) == 0
			case control >= 6:
				if i < 64 {
					result[i] = control == 7
				} else {
					result[i] = mask&(1<<uint(i-64)) == 0
				}
			}
		}
	} else {
		if control != 0 && control != 6 {
			return nil, false
		}
		for i := 0; i < 16; i++ {
			_, exists := table[i]
			enabled := control == 6 && exists || control == 0 && mask&(1<<uint(i)) != 0
			if enabled && !exists {
				return nil, false
			}
			result[i] = !enabled
		}
	}
	for index, ch := range table {
		if ch.Frequency > 0 && !result[index] {
			return result, true
		}
	}
	return nil, false
}

func uplinkMACReserve(session *types.DeviceSession) int {
	if session == nil {
		return 0
	}
	if pending := session.PendingUplink; pending != nil {
		return pending.ReservedFOpts
	}
	size := 0
	for _, response := range session.PendingMACResponses {
		if size+len(response) > 15 {
			break
		}
		size += len(response)
	}
	return size
}
func applicationPayloadCapacity(device contracts.Device, dataRate int, sessions ...*types.DeviceSession) int {
	reserved := 0
	if len(sessions) > 0 {
		reserved = uplinkMACReserve(sessions[0])
	}
	return max(1, maximumPayloadSize(device.LocationConfig.Region, dataRate)-reserved)
}

func applyExternalLinkADR(device contracts.Device, session *types.DeviceSession, c types.MACCommand) byte {
	status := byte(7)
	dr, power := *c.DataRate, *c.TxPower
	if dr != 15 && !regional.SupportsUplinkDataRate(device.LocationConfig.Region, dr) {
		status &^= 2
	}
	if power != 15 && power > 7 {
		status &^= 4
	}
	disabled, ok := externalChannelMask(device, session, *c.ChannelMask, *c.ChannelMaskControl)
	if !ok {
		status &^= 1
	}
	if status == 7 {
		session.DisabledChannels = disabled
		if dr != 15 {
			session.CurrentDataRate = dr
			session.CurrentSpreadingFactor = spreadingFactorForDataRate(device.LocationConfig.Region, dr)
		}
		if power != 15 {
			session.CurrentTxPower = 16 - 2*power
		}
		session.UnconfirmedRepetitions = max(1, *c.NbTrans)
		session.ExternalADR = true
	}
	return status
}

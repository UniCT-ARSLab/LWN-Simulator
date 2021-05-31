package regional_parameters

import (
	"errors"
	"fmt"

	c "github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
	models "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters/models_rp"
	"github.com/brocaar/lorawan"
)

const (
	LenChMask  = 16
	Code_Eu868 = iota
	Code_Us915
	Code_Cn779
	Code_Eu433
	Code_Au915
	Code_Cn470
	Code_As923
	Code_Kr920
	Code_In865
	Code_Ru864
)

const (
	ChMaskCntlChannel = iota
	ChMaskCntlGroup
)

type Region interface {
	Setup()
	GetDataRate(uint8) (string, string)
	FrequencySupported(uint32) error
	DataRateSupported(uint8) error
	RX1DROffsetSupported(uint8) error
	GetCode() int
	GetChannels() []c.Channel
	GetMinDataRate() uint8
	GetMaxDataRate() uint8
	GetNbReservedChannels() int
	GetFrequencyBeacon() uint32
	GetDataRateBeacon() uint8
	GetCodR(uint8) string
	SetupInfoRequest(int) (string, int)
	LinkAdrReq(uint8, lorawan.ChMask, uint8, *[]c.Channel) ([]bool, []error)
	SetupRX1(uint8, uint8, int, lorawan.DwellTime) (uint8, int)
	GetPayloadSize(uint8, lorawan.DwellTime) (int, int)
	GetParameters() models.Parameters
}

type regionInfo struct {
	info func() Region
}

var regionRegistry = map[int]regionInfo{
	Code_Eu868: {func() Region { return &Eu868{} }},
	Code_Us915: {func() Region { return &Us915{} }},
	Code_Cn779: {func() Region { return &Cn779{} }},
	Code_Eu433: {func() Region { return &Eu433{} }},
	Code_Au915: {func() Region { return &Au915{} }},
	Code_Cn470: {func() Region { return &Cn470{} }},
	Code_As923: {func() Region { return &As923{} }},
	Code_Kr920: {func() Region { return &Kr920{} }},
	Code_In865: {func() Region { return &In865{} }},
	Code_Ru864: {func() Region { return &Ru864{} }},
}

func GetRegionalParameters(Code int) Region {

	r := regionRegistry[Code]
	return r.info()

}

func GetInfo(Code int) models.Informations {

	region := GetRegionalParameters(Code)
	region.Setup()

	param := region.GetParameters()

	//values datarate
	var valuesDatarate [14]int
	var payloadSize [14][2]int
	var payloadSizeDT [14][2]int
	var cofiguration [14]string
	for i := param.MinDataRate; i < 14; i++ {

		modu, dr := region.GetDataRate(i)
		if dr != "" {
			valuesDatarate[i] = int(i)
			cofiguration[i] = modu + ": " + dr
		} else {
			valuesDatarate[i] = -1
		}

		payloadSize[i][0], payloadSize[i][1] = region.GetPayloadSize(i, lorawan.DwellTimeNoLimit)

	}

	if param.Code == Code_As923 || param.Code == Code_Au915 {
		for i := param.MinDataRate; i < param.MaxDataRate; i++ {
			payloadSizeDT[i][0], payloadSizeDT[i][1] = region.GetPayloadSize(i, lorawan.DwellTime400ms)
		}
	}

	info := models.Informations{
		MaxRX1DROffset:     param.MaxRX1DROffset + 1,
		DataRate:           valuesDatarate,
		Configuration:      cofiguration,
		FrequencyRX2:       param.FrequencyRX2,
		DataRateRX2:        param.DataRateRX2,
		MinFrequency:       param.MinFrequency,
		MaxFrequency:       param.MaxFrequency,
		TablePayloadSize:   payloadSize,
		TablePayloadSizeDT: payloadSizeDT,
	}

	return info
}

func linkADRReqForChannels(region Region, ChMaskCntl uint8, ChMask lorawan.ChMask,
	newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	var errs []error

	chMaskTmp := ChMask
	channelsCopy := *channels
	channelsInactive := 0
	acks := []bool{false, false, false}

	switch ChMaskCntl {

	case 0: //only 0 in mask

		for _, enable := range ChMask {

			if !enable {
				channelsInactive++
			} else {
				break
			}
		}

		if channelsInactive == LenChMask { // all channels inactive
			errs = append(errs, errors.New("MAc Command disables all channels"))
		}

	case 6:

		for i, _ := range chMaskTmp {
			chMaskTmp[i] = true
		}

	}

	for i := region.GetNbReservedChannels(); i < LenChMask; i++ { //i primi 3 channel sono riservati

		if chMaskTmp[i] {

			if i >= len(channelsCopy) {
				errs = append(errs, errors.New("Unable to configure an undefined channel"))
				break
			}

			if !channelsCopy[i].Active { // can't enable uplink channel

				msg := fmt.Sprintf("ChMask can't enable an inactive channel[%v]", i)
				errs = append(errs, errors.New(msg))

				break

			} else { //channel active, check datarate

				err := channelsCopy[i].IsSupportedDR(newDataRate)
				if err == nil { //at least one channel supports DataRate
					acks[1] = true //ackDr
				}

			}
			channelsCopy[i].EnableUplink = chMaskTmp[i]

		}
	}
	acks[0] = true //ackMask

	//datarate
	if err := region.DataRateSupported(newDataRate); err != nil {

		acks[1] = false
		errs = append(errs, err)

	}

	acks[2] = true //txack

	if acks[0] && acks[1] && acks[2] {
		channels = &channelsCopy
	}

	return acks, errs
}

func linkADRReqForGroupOfChannels(region Region, ChMaskCntl uint8, ChMask lorawan.ChMask,
	newDataRate uint8, channels *[]c.Channel, nbGroup int) ([]bool, []error) {

	var errs []error
	channelsCopy := *channels
	lenMask := LenChMask
	offset := ChMaskCntl
	acks := []bool{false, false, false}

	switch ChMaskCntl {

	case 0, 1, 2, 3:
		offset = ChMaskCntl * 16
		break

	case 4:
		offset = ChMaskCntl * 16
		lenMask = LenChMask / 2
		break

	case 5:

		offset = 64
		lenMask = LenChMask / 2
		groupIndex := uint(0)

		for i, bit := range ChMask { //get nb from ChMask

			if bit {
				value := uint(1)
				mask := value << uint(i)
				groupIndex = groupIndex | mask
			}

		}

		if int(groupIndex)+(lenMask-1) < len(channelsCopy) &&
			int(offset)+int(groupIndex) < len(channelsCopy) {

			for i := 0; i < lenMask; i++ {

				if !channelsCopy[i+int(groupIndex)].Active { // can't enable uplink channel

					msg := fmt.Sprintf("ChMask Error: channel[%v] is inactive so it can't enable to send a uplink", i)
					errs = append(errs, errors.New(msg))

					acks[0] = false

					break

				}

				channelsCopy[i+int(groupIndex)].EnableUplink = ChMask[i]

			}

			if !channelsCopy[int(offset)+int(groupIndex)].Active { // can't enable uplink channel

				msg := fmt.Sprintf("ChMask Error: channel[%v] is inactive so it can't enable to send a uplink", int(offset)+int(groupIndex))
				errs = append(errs, errors.New(msg))

				acks[0] = false

				break

			}

			channelsCopy[int(offset)+int(groupIndex)].EnableUplink = ChMask[lenMask-1]

		} else {
			errs = append(errs, errors.New("ChMask value is too large"))
		}

		return acks, errs

	case 6:
		offset = 64
		lenMask = LenChMask / 2

		for i := 0; i < nbGroup; i++ {

			if !channelsCopy[i].Active { // can't enable uplink channel

				msg := fmt.Sprintf("ChMask Error: channel[%v] is inactive so it can't enable to send a uplink", i)
				errs = append(errs, errors.New(msg))

				acks[0] = false

				break

			} else { //channel active, check datarate

				err := channelsCopy[i].IsSupportedDR(newDataRate)
				if err == nil { //at least one channel supports DataRate
					acks[1] = true //ackDr
				}

			}

			channelsCopy[i].EnableUplink = true

		}

		break

	case 7:
		offset = 64
		lenMask = LenChMask / 2

		for i := 0; i < nbGroup; i++ {

			if !channelsCopy[i].Active { // can't enable uplink channel

				msg := fmt.Sprintf("ChMask Error: channel[%v] is inactive so it can't enable to send a uplink", i)
				errs = append(errs, errors.New(msg))

				acks[0] = false

				break

			} else { //channel active, check datarate

				err := channelsCopy[i].IsSupportedDR(newDataRate)
				if err == nil { //at least one channel supports DataRate
					acks[1] = true //ackDr
				}

			}

			channelsCopy[i].EnableUplink = false

		}

	}

	for i := int(offset); i < lenMask; i++ {

		if !channelsCopy[i].Active { // can't enable uplink channel

			msg := fmt.Sprintf("ChMask Error: channel[%v] is inactive so it can't enable to send a uplink", i)
			errs = append(errs, errors.New(msg))

			acks[0] = false

			break

		} else { //channel active, check datarate

			err := channelsCopy[i].IsSupportedDR(newDataRate)
			if err == nil { //at least one channel supports DataRate
				acks[1] = true //ackDr
			}

		}

		channelsCopy[i].EnableUplink = ChMask[i]

	}

	if len(errs) == 0 { //no error chMask
		acks[0] = true //ackMask
	}

	//datarate
	if err := region.DataRateSupported(newDataRate); err != nil {

		acks[1] = false
		errs = append(errs, err)

	} else {
		acks[1] = true
	}

	acks[2] = true //txack

	if acks[0] && acks[1] && acks[2] {
		channels = &channelsCopy
	}

	return acks, errs
}

func DecrementDataRate(region Region, datarate uint8) uint8 {

	datarateNEW := int(datarate) - 1
	minDR := region.GetMinDataRate() - 1

	for datarateNEW > int(minDR) {

		_, drString := region.GetDataRate(datarate)
		if drString != "" {
			return uint8(datarateNEW)
		}

		datarateNEW--
	}

	return region.GetMinDataRate()
}

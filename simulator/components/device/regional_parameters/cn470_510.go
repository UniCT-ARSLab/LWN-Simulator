package regional_parameters

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	c "github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
	models "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters/models_rp"
	"github.com/brocaar/lorawan"
)

type Cn470 struct {
	Info models.Parameters
}

//manca un setup
func (cn *Cn470) Setup() {
	cn.Info.Code = Code_Cn470
	cn.Info.MinFrequency = 470000000
	cn.Info.MaxFrequency = 510000000
	cn.Info.FrequencyRX2 = 505300000
	cn.Info.DataRateRX2 = 0
	cn.Info.MinDataRate = 0
	cn.Info.MaxDataRate = 5
	cn.Info.MinRX1DROffset = 0
	cn.Info.MaxRX1DROffset = 5
	cn.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   470300000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 96,
		},
		{
			EnableUplink:       true,
			InitialFrequency:   500300000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 48,
		},
	}
	cn.Info.InfoClassB.Setup(508300000, 508300000, 2, cn.Info.MinDataRate, cn.Info.MaxDataRate)
}

func (cn *Cn470) GetDataRate(datarate uint8) (string, string) {

	switch datarate {
	case 0, 1, 2, 3, 4, 5:
		r := fmt.Sprintf("SF%vBW125", 12-datarate)
		return "LORA", r

	}
	return "", ""
}

func (cn *Cn470) FrequencySupported(frequency uint32) error {

	if frequency < cn.Info.MinFrequency || frequency > cn.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (cn *Cn470) DataRateSupported(datarate uint8) error {

	if datarate < cn.Info.MinDataRate || datarate > cn.Info.MaxDataRate {
		return errors.New("Invalid Data Rate")
	}

	return nil
}

func (cn *Cn470) GetCode() int {
	return Code_Cn470
}

func (cn *Cn470) GetChannels() []c.Channel {
	var channels []c.Channel

	for _, group := range cn.Info.InfoGroupChannels {
		for i := 0; i < group.NbReservedChannels; i++ {
			frequency := group.InitialFrequency + group.OffsetFrequency*uint32(i)
			Active := true
			if (i >= 6 && i <= 38) || (i >= 45 && i <= 77) {
				Active = false
			}
			ch := c.Channel{
				Active:            Active,
				EnableUplink:      group.EnableUplink,
				FrequencyUplink:   frequency,
				FrequencyDownlink: frequency,
				MinDR:             group.MinDataRate,
				MaxDR:             group.MaxDataRate,
			}
			channels = append(channels, ch)
		}

	}

	return channels
}

func (cn *Cn470) GetMinDataRate() uint8 {
	return cn.Info.MinDataRate
}

func (cn *Cn470) GetMaxDataRate() uint8 {
	return cn.Info.MaxDataRate
}

func (cn *Cn470) GetNbReservedChannels() int {
	return cn.Info.InfoGroupChannels[0].NbReservedChannels
}

func (cn *Cn470) GetCodR(datarate uint8) string {
	return "4/5"
}

func (cn *Cn470) RX1DROffsetSupported(offset uint8) error {
	if offset >= cn.Info.MinRX1DROffset && offset <= cn.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (cn *Cn470) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask,
	newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	var errs []error
	chMaskTmp := ChMask
	channelsCopy := *channels
	offset := ChMaskCntl
	acks := []bool{false, false, false}

	switch ChMaskCntl {

	case 0, 1, 2, 3, 4, 5:

		offset = ChMaskCntl * 16

		for i := int(offset); i < len(chMaskTmp); i++ {

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

	case 6:
		offset = 0

		for i := int(offset); i < len(channelsCopy); i++ {

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
	if err := cn.DataRateSupported(newDataRate); err != nil || !acks[1] {

		acks[1] = false
		errs = append(errs, err)

	}

	acks[2] = true //txack
	acks[2] = true //txack

	if acks[0] && acks[1] && acks[2] {
		channels = &channelsCopy
	}

	return acks, errs
}

func (cn *Cn470) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {
	newIndexChannel := indexChannel % 48

	DataRateRx1 := uint8(0)
	if datarate > rx1offset { //set data rate RX1
		DataRateRx1 = datarate - rx1offset
	}

	return DataRateRx1, newIndexChannel
}

func (cn *Cn470) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	if indexChannel >= cn.GetNbReservedChannels() {
		indexChannel = rand.Int() % cn.GetNbReservedChannels()
	}

	_, drString := cn.GetDataRate(5)
	return drString, indexChannel

}

func (cn *Cn470) GetFrequencyBeacon() uint32 {
	return cn.Info.InfoClassB.FrequencyBeacon
}

func (cn *Cn470) GetDataRateBeacon() uint8 {
	return cn.Info.InfoClassB.DataRate
}

func (cn *Cn470) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

	switch datarate {
	case 0, 1, 2:
		return 59, 51
	case 3:
		return 123, 115
	case 4, 5:
		return 230, 222
	}

	return 0, 0

}

func (cn *Cn470) GetParameters() models.Parameters {
	return cn.Info
}

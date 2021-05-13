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

type Us915 struct {
	Info models.Parameters
}

//manca un setup
func (us *Us915) Setup() {
	us.Info.Code = Code_Us915
	us.Info.MinFrequency = 902000000
	us.Info.MaxFrequency = 928000000
	us.Info.FrequencyRX2 = 923300000
	us.Info.DataRateRX2 = 8
	us.Info.MinDataRate = 0
	us.Info.MaxDataRate = 13
	us.Info.MinRX1DROffset = 0
	us.Info.MaxRX1DROffset = 3
	us.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   902300000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        3,
			NbReservedChannels: 64,
		},
		{
			EnableUplink:       true,
			InitialFrequency:   903000000,
			OffsetFrequency:    1600000,
			MinDataRate:        4,
			MaxDataRate:        4,
			NbReservedChannels: 8,
		},
		{
			EnableUplink:       false,
			InitialFrequency:   923300000,
			OffsetFrequency:    600000,
			MinDataRate:        8,
			MaxDataRate:        13,
			NbReservedChannels: 8,
		},
	}
	us.Info.InfoClassB.Setup(923300000, 923300000, 8, us.Info.MinDataRate, us.Info.MaxDataRate)

}

func (us *Us915) GetDataRate(datarate uint8) (string, string) {

	switch datarate {
	case 0, 1, 2, 3:
		r := fmt.Sprintf("SF%vBW125", 10-datarate)
		return "LORA", r
	case 4:
		return "LORA", "SF8BW500"

	case 8, 9, 10, 11, 12, 13:
		r := fmt.Sprintf("SF%vBW500", 20-datarate)
		return "LORA", r

	default:
		return "", ""

	}

}

func (us *Us915) FrequencySupported(frequency uint32) error {

	if frequency < us.Info.MinFrequency || frequency > us.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (us *Us915) DataRateSupported(datarate uint8) error {

	if _, dr := us.GetDataRate(datarate); dr == "" {
		return errors.New("Invalid Data Rate or RFU")
	}

	return nil
}

func (us *Us915) GetCode() int {
	return Code_Us915
}

func (us *Us915) GetChannels() []c.Channel {
	var channels []c.Channel

	for _, group := range us.Info.InfoGroupChannels {
		for i := 0; i < group.NbReservedChannels; i++ {
			frequency := group.InitialFrequency + group.OffsetFrequency*uint32(i)
			ch := c.Channel{
				Active:            true,
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

func (us *Us915) GetMinDataRate() uint8 {
	return us.Info.MinDataRate
}

func (us *Us915) GetMaxDataRate() uint8 {
	return us.Info.MaxDataRate
}

func (us *Us915) GetNbReservedChannels() int {
	return us.Info.InfoGroupChannels[0].NbReservedChannels + us.Info.InfoGroupChannels[1].NbReservedChannels
}

func (us *Us915) GetCodR(datarate uint8) string {
	return "4/5"
}

func (us *Us915) RX1DROffsetSupported(offset uint8) error {

	if offset >= us.Info.MinRX1DROffset && offset <= us.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (us *Us915) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask, newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	return linkADRReqForGroupOfChannels(us, ChMaskCntl, ChMask, newDataRate, channels, us.Info.InfoGroupChannels[0].NbReservedChannels)
}

func (us *Us915) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {

	newIndexChannel := (indexChannel % 8) + 72
	DataRateRx1 := uint8(0)

	switch datarate {
	case 0:
		DataRateRx1 = 10 - rx1offset
	case 1:
		DataRateRx1 = 11 - rx1offset
	case 2:
		DataRateRx1 = 12 - rx1offset
	case 3:
		DataRateRx1 = 13 - rx1offset
	case 4:
		if rx1offset == 0 {
			DataRateRx1 = 13
		} else {
			DataRateRx1 = 14 - rx1offset
		}

	}

	if DataRateRx1 < 8 {
		DataRateRx1 = 8
	}

	return DataRateRx1, newIndexChannel
}

func (us *Us915) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	datarate := uint8(0)
	indexChannel = rand.Int() % us.GetNbReservedChannels()
	if indexChannel >= us.Info.InfoGroupChannels[0].NbReservedChannels {
		datarate = uint8(4)
	}

	_, drString := us.GetDataRate(datarate)
	return drString, indexChannel

}

func (us *Us915) GetFrequencyBeacon() uint32 {
	return us.Info.InfoClassB.FrequencyBeacon
}

func (us *Us915) GetDataRateBeacon() uint8 {
	return us.Info.InfoClassB.DataRate
}

func (us *Us915) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

	switch datarate {

	case 0:
		return 19, 11

	case 1:
		return 61, 53

	case 2:
		return 133, 125

	case 3, 4:
		return 250, 242

	case 8:
		return 41, 33

	case 9:
		return 117, 109

	case 10, 11, 12, 13:
		return 230, 222

	default:
		return 0, 0

	}

}

func (us *Us915) GetParameters() models.Parameters {
	return us.Info
}

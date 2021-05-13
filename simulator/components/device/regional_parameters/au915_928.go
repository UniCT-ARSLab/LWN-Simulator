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

type Au915 struct {
	Info models.Parameters
}

//manca un setup
func (au *Au915) Setup() {
	au.Info.Code = Code_Au915
	au.Info.MinFrequency = 915000000
	au.Info.MaxFrequency = 928000000
	au.Info.FrequencyRX2 = 923300000
	au.Info.DataRateRX2 = 8
	au.Info.MinDataRate = 0
	au.Info.MaxDataRate = 13
	au.Info.MinRX1DROffset = 0
	au.Info.MaxRX1DROffset = 5
	au.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   915200000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 64,
		},
		{
			EnableUplink:       true,
			InitialFrequency:   915900000,
			OffsetFrequency:    1600000,
			MinDataRate:        6,
			MaxDataRate:        6,
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
	au.Info.InfoClassB.Setup(923300000, 923300000, 8, au.Info.MinDataRate, au.Info.MaxDataRate)

}

func (au *Au915) GetDataRate(datarate uint8) (string, string) {

	switch datarate {

	case 0, 1, 2, 3, 4, 5:
		r := fmt.Sprintf("SF%vBW125", 12-datarate)
		return "LORA", r

	case 6:
		return "LORA", "SF8BW500"

	case 8, 9, 10, 11, 12, 13:
		r := fmt.Sprintf("SF%vBW500", 20-datarate)
		return "LORA", r

	default:
		return "", ""

	}

}

func (au *Au915) FrequencySupported(frequency uint32) error {

	if frequency < au.Info.MinFrequency || frequency > au.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (au *Au915) DataRateSupported(datarate uint8) error {

	if _, dr := au.GetDataRate(datarate); dr == "" {
		return errors.New("Invalid Data Rate or RFU")
	}

	return nil
}

func (au *Au915) GetCode() int {
	return Code_Au915
}

func (au *Au915) GetChannels() []c.Channel {
	var channels []c.Channel

	for _, group := range au.Info.InfoGroupChannels {
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

func (au *Au915) GetMinDataRate() uint8 {
	return au.Info.MinDataRate
}

func (au *Au915) GetMaxDataRate() uint8 {
	return au.Info.MaxDataRate
}

func (au *Au915) GetNbReservedChannels() int {
	return au.Info.InfoGroupChannels[0].NbReservedChannels + au.Info.InfoGroupChannels[1].NbReservedChannels
}

func (au *Au915) GetCodR(datarate uint8) string {
	return "4/5"
}

func (au *Au915) RX1DROffsetSupported(offset uint8) error {
	if offset >= au.Info.MinRX1DROffset && offset <= au.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (au *Au915) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask,
	newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	return linkADRReqForGroupOfChannels(au, ChMaskCntl, ChMask, newDataRate, channels, au.Info.InfoGroupChannels[0].NbReservedChannels)
}

func (au *Au915) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {

	newIndexChannel := indexChannel % 8
	DataRateRx1 := uint8(0)

	if datarate > rx1offset { //set data rate RX1
		DataRateRx1 = datarate - rx1offset + 8
	}

	if DataRateRx1 > 13 {
		DataRateRx1 = 13
	}

	return DataRateRx1, newIndexChannel

}

func (au *Au915) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	datarate := uint8(2)

	indexChannel = rand.Int() % au.GetNbReservedChannels()
	if indexChannel >= au.Info.InfoGroupChannels[0].NbReservedChannels {
		datarate = uint8(6)
	}

	_, drString := au.GetDataRate(datarate)

	return drString, indexChannel

}

func (au *Au915) GetFrequencyBeacon() uint32 {
	return au.Info.InfoClassB.FrequencyBeacon
}

func (au *Au915) GetDataRateBeacon() uint8 {
	return au.Info.InfoClassB.DataRate
}

func (au *Au915) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

	if dTime == lorawan.DwellTimeNoLimit {
		switch datarate {
		case 0, 1, 2:
			return 59, 51
		case 3:
			return 123, 115
		case 4, 5, 6:
			return 230, 222
		case 8:
			return 41, 33
		case 9:
			return 117, 109
		case 10, 11, 12, 13:
			return 230, 222
		}
	} else {
		switch datarate {
		case 0, 1:
			return 0, 0
		case 2:
			return 19, 11
		case 3:
			return 61, 53
		case 4:
			return 133, 125
		case 5, 6:
			return 250, 242
		case 8:
			return 41, 33
		case 9:
			return 117, 109
		case 10, 11, 12, 13:
			return 230, 222
		}
	}

	return 0, 0

}

func (au *Au915) GetParameters() models.Parameters {
	return au.Info
}

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

type In865 struct {
	Info models.Parameters
}

//manca un setup
func (in *In865) Setup() {
	in.Info.Code = Code_In865
	in.Info.MinFrequency = 865000000
	in.Info.MaxFrequency = 867000000
	in.Info.FrequencyRX2 = 866550000
	in.Info.DataRateRX2 = 2
	in.Info.MinDataRate = 0
	in.Info.MaxDataRate = 7
	in.Info.MinRX1DROffset = 0
	in.Info.MaxRX1DROffset = 7
	in.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   865062500,
			OffsetFrequency:    340000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 2,
		},
		{
			EnableUplink:       true,
			InitialFrequency:   865985000,
			OffsetFrequency:    0,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 1,
		},
	}

	in.Info.InfoClassB.Setup(866550000, 866550000, 4, in.Info.MinDataRate, in.Info.MaxDataRate)

}

func (in *In865) GetDataRate(datarate uint8) (string, string) {

	switch datarate {

	case 0, 1, 2, 3, 4, 5:
		r := fmt.Sprintf("SF%vBW125", 12-datarate)
		return "LORA", r

	case 7:
		return "FSK", "50000"

	default:
		return "", ""

	}

}

func (in *In865) FrequencySupported(frequency uint32) error {

	if frequency < in.Info.MinFrequency || frequency > in.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (in *In865) DataRateSupported(datarate uint8) error {

	if _, dr := in.GetDataRate(datarate); dr == "" {
		return errors.New("Invalid Data Rate or RFU")
	}

	return nil
}

func (in *In865) GetCode() int {
	return Code_In865
}

func (in *In865) GetChannels() []c.Channel {
	var channels []c.Channel

	for _, group := range in.Info.InfoGroupChannels {
		for i := 0; i < group.NbReservedChannels; i++ {
			frequency := in.Info.InfoGroupChannels[0].InitialFrequency + in.Info.InfoGroupChannels[0].OffsetFrequency*uint32(i)
			ch := c.Channel{
				Active:            true,
				EnableUplink:      in.Info.InfoGroupChannels[0].EnableUplink,
				FrequencyUplink:   frequency,
				FrequencyDownlink: frequency,
				MinDR:             0,
				MaxDR:             5,
			}
			channels = append(channels, ch)
		}
	}

	return channels
}

func (in *In865) GetMinDataRate() uint8 {
	return in.Info.MinDataRate
}

func (in *In865) GetMaxDataRate() uint8 {
	return in.Info.MaxDataRate
}

func (in *In865) GetNbReservedChannels() int {
	return in.Info.InfoGroupChannels[0].NbReservedChannels + in.Info.InfoGroupChannels[1].NbReservedChannels
}

func (in *In865) GetCodR(datarate uint8) string {
	return "4/5"
}

func (in *In865) RX1DROffsetSupported(offset uint8) error {
	if offset >= in.Info.MinRX1DROffset && offset <= in.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (in *In865) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask, newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	return linkADRReqForChannels(in, ChMaskCntl, ChMask, newDataRate, channels)
}

func (in *In865) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {

	DataRateRx1 := 5

	minDR := 0

	effectiveOffset := int(rx1offset)
	if effectiveOffset > 5 { //set data rate RX1
		effectiveOffset = 5 - int(rx1offset)
	}
	dr := int(datarate) - effectiveOffset

	if dr >= minDR {
		if dr < DataRateRx1 {
			DataRateRx1 = dr
		}
	} else {
		if minDR < DataRateRx1 {
			DataRateRx1 = minDR
		}
	}

	return uint8(DataRateRx1), indexChannel
}

func (in *In865) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	if indexChannel >= in.GetNbReservedChannels() {
		indexChannel = rand.Int() % in.GetNbReservedChannels()
	}

	_, drString := in.GetDataRate(5)
	return drString, indexChannel

}

func (in *In865) GetFrequencyBeacon() uint32 {
	return in.Info.InfoClassB.FrequencyBeacon
}

func (in *In865) GetDataRateBeacon() uint8 {
	return in.Info.InfoClassB.DataRate
}

func (in *In865) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

	switch datarate {
	case 0, 1, 2:
		return 59, 51
	case 3:
		return 123, 115
	case 4, 5, 6, 7:
		return 230, 222
	}

	return 0, 0

}

func (in *In865) GetParameters() models.Parameters {
	return in.Info
}

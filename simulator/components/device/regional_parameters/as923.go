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

type As923 struct {
	Info models.Parameters
}

//manca un setup
func (as *As923) Setup() {
	as.Info.Code = Code_As923
	as.Info.MinFrequency = 923000000
	as.Info.MaxFrequency = 923500000
	as.Info.FrequencyRX2 = 923200000
	as.Info.DataRateRX2 = 2
	as.Info.MinDataRate = 0
	as.Info.MaxDataRate = 7
	as.Info.MinRX1DROffset = 0
	as.Info.MaxRX1DROffset = 7
	as.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   923200000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 2,
		},
	}
	as.Info.InfoClassB.Setup(923400000, 923400000, 3, as.Info.MinDataRate, as.Info.MaxDataRate)

}

func (as *As923) GetDataRate(datarate uint8) (string, string) {

	switch datarate {
	case 0, 1, 2, 3, 4, 5:
		r := fmt.Sprintf("SF%vBW125", 12-datarate)
		return "LORA", r

	case 6:
		return "LORA", "SF7BW250"
	case 7:
		return "FSK", "50000"
	}
	return "", ""
}

func (as *As923) FrequencySupported(frequency uint32) error {

	if frequency < as.Info.MinFrequency || frequency > as.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (as *As923) DataRateSupported(datarate uint8) error {

	if datarate < as.Info.MinDataRate || datarate > as.Info.MaxDataRate {
		return errors.New("Invalid Data Rate")
	}

	return nil
}

func (as *As923) GetCode() int {
	return Code_As923
}

func (as *As923) GetChannels() []c.Channel {
	var channels []c.Channel

	for i := 0; i < as.Info.InfoGroupChannels[0].NbReservedChannels; i++ {
		frequency := as.Info.InfoGroupChannels[0].InitialFrequency + as.Info.InfoGroupChannels[0].OffsetFrequency*uint32(i)
		ch := c.Channel{
			Active:            true,
			EnableUplink:      as.Info.InfoGroupChannels[0].EnableUplink,
			FrequencyUplink:   frequency,
			FrequencyDownlink: frequency,
			MinDR:             0,
			MaxDR:             5,
		}
		channels = append(channels, ch)
	}

	return channels
}

func (as *As923) GetMinDataRate() uint8 {
	return as.Info.MinDataRate
}

func (as *As923) GetMaxDataRate() uint8 {
	return as.Info.MaxDataRate
}

func (as *As923) GetNbReservedChannels() int {
	return as.Info.InfoGroupChannels[0].NbReservedChannels
}

func (as *As923) GetCodR(datarate uint8) string {
	return "4/5"
}

func (as *As923) RX1DROffsetSupported(offset uint8) error {
	if offset >= as.Info.MinRX1DROffset && offset <= as.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (as *As923) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask,
	newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	return linkADRReqForChannels(as, ChMaskCntl, ChMask, newDataRate, channels)
}

func (as *As923) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {

	DataRateRx1 := 5

	minDR := 0

	if dtime == lorawan.DwellTime400ms {
		minDR = 2
	}

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

func (as *As923) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	if indexChannel >= as.GetNbReservedChannels() {
		indexChannel = rand.Int() % as.GetNbReservedChannels()
	}

	_, datarate := as.GetDataRate(5)
	return datarate, indexChannel
}

func (as *As923) GetFrequencyBeacon() uint32 {
	return as.Info.InfoClassB.FrequencyBeacon
}

func (as *As923) GetDataRateBeacon() uint8 {
	return as.Info.InfoClassB.DataRate
}

func (as *As923) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

	if dTime == lorawan.DwellTimeNoLimit {

		switch datarate {
		case 0, 1, 2:
			return 59, 51
		case 3:
			return 123, 115
		case 4, 5, 6, 7:
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
		case 5, 6, 7:
			return 250, 242

		}
	}

	return 0, 0

}

func (as *As923) GetParameters() models.Parameters {
	return as.Info
}

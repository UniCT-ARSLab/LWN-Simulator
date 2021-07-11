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

type Ru864 struct {
	Info models.Parameters
}

//manca un setup
func (ru *Ru864) Setup() {
	ru.Info.Code = Code_Ru864
	ru.Info.MinFrequency = 864000000
	ru.Info.MaxFrequency = 870000000
	ru.Info.FrequencyRX2 = 869100000
	ru.Info.DataRateRX2 = 0
	ru.Info.MinDataRate = 0
	ru.Info.MaxDataRate = 7
	ru.Info.MinRX1DROffset = 0
	ru.Info.MaxRX1DROffset = 5
	ru.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   868900000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 2,
		},
	}
	ru.Info.InfoClassB.Setup(869100000, 868900000, 3, ru.Info.MinDataRate, ru.Info.MaxDataRate)
}

func (ru *Ru864) GetDataRate(datarate uint8) (string, string) {

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

func (ru *Ru864) FrequencySupported(frequency uint32) error {

	if frequency < ru.Info.MinFrequency || frequency > ru.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (ru *Ru864) DataRateSupported(datarate uint8) error {

	if datarate < ru.Info.MinDataRate || datarate > ru.Info.MaxDataRate {
		return errors.New("Invalid Data Rate")
	}

	return nil
}

func (ru *Ru864) GetCode() int {
	return Code_Ru864
}

func (ru *Ru864) GetChannels() []c.Channel {
	var channels []c.Channel

	for i := 0; i < ru.Info.InfoGroupChannels[0].NbReservedChannels; i++ {
		frequency := ru.Info.InfoGroupChannels[0].InitialFrequency + ru.Info.InfoGroupChannels[0].OffsetFrequency*uint32(i)
		ch := c.Channel{
			Active:            true,
			EnableUplink:      ru.Info.InfoGroupChannels[0].EnableUplink,
			FrequencyUplink:   frequency,
			FrequencyDownlink: frequency,
			MinDR:             0,
			MaxDR:             5,
		}
		channels = append(channels, ch)
	}

	return channels
}

func (ru *Ru864) GetMinDataRate() uint8 {
	return ru.Info.MinDataRate
}

func (ru *Ru864) GetMaxDataRate() uint8 {
	return ru.Info.MaxDataRate
}

func (ru *Ru864) GetNbReservedChannels() int {
	return ru.Info.InfoGroupChannels[0].NbReservedChannels
}

func (ru *Ru864) GetCodR(datarate uint8) string {
	return "4/5"
}

func (ru *Ru864) RX1DROffsetSupported(offset uint8) error {
	if offset >= ru.Info.MinRX1DROffset && offset <= ru.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (ru *Ru864) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask, newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	return linkADRReqForChannels(ru, ChMaskCntl, ChMask, newDataRate, channels)
}

func (ru *Ru864) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {

	DataRateRx1 := uint8(0)
	if datarate > rx1offset { //set data rate RX1
		DataRateRx1 = datarate - rx1offset
	}

	return DataRateRx1, indexChannel
}

func (ru *Ru864) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	if indexChannel >= ru.GetNbReservedChannels() {
		indexChannel = rand.Int() % ru.GetNbReservedChannels()
	}

	_, drString := ru.GetDataRate(5)
	return drString, indexChannel

}

func (ru *Ru864) GetFrequencyBeacon() uint32 {
	return ru.Info.InfoClassB.FrequencyBeacon
}

func (ru *Ru864) GetDataRateBeacon() uint8 {
	return ru.Info.InfoClassB.DataRate
}

func (ru *Ru864) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

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

func (ru *Ru864) GetParameters() models.Parameters {
	return ru.Info
}

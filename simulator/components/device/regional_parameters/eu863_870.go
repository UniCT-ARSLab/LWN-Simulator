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

type Eu868 struct {
	Info models.Parameters
}

//manca un setup
func (eu *Eu868) Setup() {
	eu.Info.Code = Code_Eu868
	eu.Info.MinFrequency = 863000000
	eu.Info.MaxFrequency = 870000000
	eu.Info.FrequencyRX2 = 869525000
	eu.Info.DataRateRX2 = 0
	eu.Info.MinDataRate = 0
	eu.Info.MaxDataRate = 7
	eu.Info.MinRX1DROffset = 0
	eu.Info.MaxRX1DROffset = 5
	eu.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   868100000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 3,
		},
	}
	eu.Info.InfoClassB.Setup(869525000, 869525000, 3, eu.Info.MinDataRate, eu.Info.MaxDataRate)

}

func (eu *Eu868) GetDataRate(datarate uint8) (string, string) {

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

func (eu *Eu868) FrequencySupported(frequency uint32) error {

	if frequency < eu.Info.MinFrequency || frequency > eu.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (eu *Eu868) DataRateSupported(datarate uint8) error {

	if datarate < eu.Info.MinDataRate || datarate > eu.Info.MaxDataRate {
		return errors.New("Invalid Data Rate")
	}

	return nil
}

func (eu *Eu868) GetCode() int {
	return Code_Eu868
}

func (eu *Eu868) GetChannels() []c.Channel {
	var channels []c.Channel

	for i := 0; i < eu.Info.InfoGroupChannels[0].NbReservedChannels; i++ {
		frequency := eu.Info.InfoGroupChannels[0].InitialFrequency + eu.Info.InfoGroupChannels[0].OffsetFrequency*uint32(i)
		ch := c.Channel{
			Active:            true,
			EnableUplink:      eu.Info.InfoGroupChannels[0].EnableUplink,
			FrequencyUplink:   frequency,
			FrequencyDownlink: frequency,
			MinDR:             0,
			MaxDR:             5,
		}
		channels = append(channels, ch)
	}

	return channels
}

func (eu *Eu868) GetMinDataRate() uint8 {
	return eu.Info.MinDataRate
}

func (eu *Eu868) GetMaxDataRate() uint8 {
	return eu.Info.MaxDataRate
}

func (eu *Eu868) GetNbReservedChannels() int {
	return eu.Info.InfoGroupChannels[0].NbReservedChannels
}

func (eu *Eu868) GetCodR(datarate uint8) string {
	return "4/5"
}

func (eu *Eu868) RX1DROffsetSupported(offset uint8) error {
	if offset >= eu.Info.MinRX1DROffset && offset <= eu.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (eu *Eu868) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask,
	newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	return linkADRReqForChannels(eu, ChMaskCntl, ChMask, newDataRate, channels)
}

func (eu *Eu868) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {

	DataRateRx1 := uint8(0)
	if datarate > rx1offset { //set data rate RX1
		DataRateRx1 = datarate - rx1offset
	}

	return DataRateRx1, indexChannel
}

func (eu *Eu868) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	if indexChannel >= eu.GetNbReservedChannels() {
		indexChannel = rand.Int() % eu.GetNbReservedChannels()
	}

	_, datarate := eu.GetDataRate(5)
	return datarate, indexChannel
}

func (eu *Eu868) GetFrequencyBeacon() uint32 {
	return eu.Info.InfoClassB.FrequencyBeacon
}

func (eu *Eu868) GetDataRateBeacon() uint8 {
	return eu.Info.InfoClassB.DataRate
}

func (eu *Eu868) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

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

func (eu *Eu868) GetParameters() models.Parameters {
	return eu.Info
}

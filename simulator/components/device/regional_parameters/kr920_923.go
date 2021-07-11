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

type Kr920 struct {
	Info models.Parameters
}

//manca un setup
func (kr *Kr920) Setup() {
	kr.Info.Code = Code_Kr920
	kr.Info.MinFrequency = 920900000
	kr.Info.MaxFrequency = 923300000
	kr.Info.FrequencyRX2 = 921900000
	kr.Info.DataRateRX2 = 0
	kr.Info.MinDataRate = 0
	kr.Info.MaxDataRate = 5
	kr.Info.MinRX1DROffset = 0
	kr.Info.MaxRX1DROffset = 5
	kr.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   922100000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 3,
		},
		{
			EnableUplink:       true,
			InitialFrequency:   920900000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 6,
		},
		{
			EnableUplink:       true,
			InitialFrequency:   922700000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 4,
		},
	}

	kr.Info.InfoClassB.Setup(923100000, 923100000, 3, kr.Info.MinDataRate, kr.Info.MaxDataRate)

}

func (kr *Kr920) GetDataRate(datarate uint8) (string, string) {

	switch datarate {
	case 0, 1, 2, 3, 4, 5:
		r := fmt.Sprintf("SF%vBW125", 12-datarate)
		return "LORA", r
	}
	return "", ""
}

func (kr *Kr920) FrequencySupported(frequency uint32) error {

	if frequency < kr.Info.MinFrequency || frequency > kr.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (kr *Kr920) DataRateSupported(datarate uint8) error {

	if datarate < kr.Info.MinDataRate || datarate > kr.Info.MaxDataRate {
		return errors.New("Invalid Data Rate")
	}

	return nil
}

func (kr *Kr920) GetCode() int {
	return Code_Kr920
}

func (kr *Kr920) GetChannels() []c.Channel {
	var channels []c.Channel

	for _, group := range kr.Info.InfoGroupChannels {
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

func (kr *Kr920) GetMinDataRate() uint8 {
	return kr.Info.MinDataRate
}

func (kr *Kr920) GetMaxDataRate() uint8 {
	return kr.Info.MaxDataRate
}

func (kr *Kr920) GetNbReservedChannels() int {
	return kr.Info.InfoGroupChannels[0].NbReservedChannels
}

func (kr *Kr920) GetCodR(datarate uint8) string {
	return "4/5"
}

func (kr *Kr920) RX1DROffsetSupported(offset uint8) error {
	if offset >= kr.Info.MinRX1DROffset && offset <= kr.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (kr *Kr920) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask, newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	return linkADRReqForChannels(kr, ChMaskCntl, ChMask, newDataRate, channels)
}

func (kr *Kr920) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {

	DataRateRx1 := uint8(0)
	if datarate > rx1offset { //set data rate RX1
		DataRateRx1 = datarate - rx1offset
	}

	return DataRateRx1, indexChannel
}

func (kr *Kr920) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	if indexChannel >= kr.GetNbReservedChannels() {
		indexChannel = rand.Int() % kr.GetNbReservedChannels()
	}

	_, drString := kr.GetDataRate(5)
	return drString, indexChannel

}

func (kr *Kr920) GetFrequencyBeacon() uint32 {
	return kr.Info.InfoClassB.FrequencyBeacon
}

func (kr *Kr920) GetDataRateBeacon() uint8 {
	return kr.Info.InfoClassB.DataRate
}

func (kr *Kr920) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

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

func (kr *Kr920) GetParameters() models.Parameters {
	return kr.Info
}

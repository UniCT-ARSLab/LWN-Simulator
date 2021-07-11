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

type Cn779 struct {
	Info models.Parameters
}

//manca un setup
func (cn *Cn779) Setup() {
	cn.Info.Code = Code_Cn779
	cn.Info.MinFrequency = 779500000
	cn.Info.MaxFrequency = 786500000
	cn.Info.FrequencyRX2 = 786000000
	cn.Info.DataRateRX2 = 0
	cn.Info.MinDataRate = 0
	cn.Info.MaxDataRate = 7
	cn.Info.MinRX1DROffset = 0
	cn.Info.MaxRX1DROffset = 5
	cn.Info.InfoGroupChannels = []models.InfoGroupChannels{
		{
			EnableUplink:       true,
			InitialFrequency:   779500000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 3,
		},
		{
			EnableUplink:       true,
			InitialFrequency:   780500000,
			OffsetFrequency:    200000,
			MinDataRate:        0,
			MaxDataRate:        5,
			NbReservedChannels: 3,
		},
	}
	cn.Info.InfoClassB.Setup(785000000, 785000000, 3, cn.Info.MinDataRate, cn.Info.MaxDataRate)

}

func (cn *Cn779) GetDataRate(datarate uint8) (string, string) {
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

func (cn *Cn779) FrequencySupported(frequency uint32) error {

	if frequency < cn.Info.MinFrequency || frequency > cn.Info.MaxFrequency {
		return errors.New("Frequency not supported")
	}

	return nil
}

func (cn *Cn779) DataRateSupported(datarate uint8) error {

	if datarate < cn.Info.MinDataRate || datarate > cn.Info.MaxDataRate {
		return errors.New("Invalid Data Rate")
	}

	return nil
}

func (cn *Cn779) GetCode() int {
	return Code_Cn779
}

func (cn *Cn779) GetChannels() []c.Channel {
	var channels []c.Channel

	for i := 0; i < cn.Info.InfoGroupChannels[0].NbReservedChannels; i++ {
		frequency := cn.Info.InfoGroupChannels[0].InitialFrequency + cn.Info.InfoGroupChannels[0].OffsetFrequency*uint32(i)
		ch := c.Channel{
			Active:            true,
			EnableUplink:      cn.Info.InfoGroupChannels[0].EnableUplink,
			FrequencyUplink:   frequency,
			FrequencyDownlink: frequency,
			MinDR:             0,
			MaxDR:             5,
		}
		channels = append(channels, ch)
	}

	return channels
}

func (cn *Cn779) GetMinDataRate() uint8 {
	return cn.Info.MinDataRate
}

func (cn *Cn779) GetMaxDataRate() uint8 {
	return cn.Info.MaxDataRate
}

func (cn *Cn779) GetNbReservedChannels() int {
	return cn.Info.InfoGroupChannels[0].NbReservedChannels
}

func (cn *Cn779) GetCodR(datarate uint8) string {
	return "4/5"
}

func (cn *Cn779) RX1DROffsetSupported(offset uint8) error {
	if offset >= cn.Info.MinRX1DROffset && offset <= cn.Info.MaxRX1DROffset {
		return nil
	}

	return errors.New("Invalid RX1DROffset")
}

func (cn *Cn779) LinkAdrReq(ChMaskCntl uint8, ChMask lorawan.ChMask,
	newDataRate uint8, channels *[]c.Channel) ([]bool, []error) {

	return linkADRReqForChannels(cn, ChMaskCntl, ChMask, newDataRate, channels)
}

func (cn *Cn779) SetupRX1(datarate uint8, rx1offset uint8, indexChannel int, dtime lorawan.DwellTime) (uint8, int) {

	DataRateRx1 := uint8(0)
	if datarate > rx1offset { //set data rate RX1
		DataRateRx1 = datarate - rx1offset
	}

	return DataRateRx1, indexChannel
}

func (cn *Cn779) SetupInfoRequest(indexChannel int) (string, int) {

	rand.Seed(time.Now().UTC().UnixNano())

	if indexChannel >= cn.GetNbReservedChannels() {
		indexChannel = rand.Int() % cn.GetNbReservedChannels()
	}

	_, drString := cn.GetDataRate(5)

	return drString, indexChannel

}

func (cn *Cn779) GetFrequencyBeacon() uint32 {
	return cn.Info.InfoClassB.FrequencyBeacon
}

func (cn *Cn779) GetDataRateBeacon() uint8 {
	return cn.Info.InfoClassB.DataRate
}

func (cn *Cn779) GetPayloadSize(datarate uint8, dTime lorawan.DwellTime) (int, int) {

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

func (cn *Cn779) GetParameters() models.Parameters {
	return cn.Info
}

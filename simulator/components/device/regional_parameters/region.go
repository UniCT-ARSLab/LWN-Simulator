package regional_parameters

import (
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
	LinkAdrReq(uint8, lorawan.ChMask, uint8, *[]c.Channel) (int, []bool, error)
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

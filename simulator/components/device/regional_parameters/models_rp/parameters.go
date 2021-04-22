package models_rp

import c "github.com/arslab/lwnsimulator/simulator/components/device/classes/models_classes"

type Parameters struct {
	Code              int                 `json:"Code"`
	MinFrequency      uint32              `json:"MinFrequency"`
	MaxFrequency      uint32              `json:"MaxFrequency"`
	FrequencyRX2      uint32              `json:"FrequencyRX2"`
	DataRateRX2       uint32              `json:"DataRateRX2"`
	MinDataRate       uint8               `json:"MinDataRate"`
	MaxDataRate       uint8               `json:"MaxDataRate"`
	InfoGroupChannels []InfoGroupChannels `json:"InfoGroupChannels"`
	InfoClassB        c.InfoClassB        `json:"InfoClassB"`
	MinRX1DROffset    uint8               `json:"MinRX1DROffset"`
	MaxRX1DROffset    uint8               `json:"MaxRX1DROffset"`
}

type Informations struct {
	MaxRX1DROffset     uint8      `json:"MaxRX1DROffset"`
	DataRate           [14]int    `json:"DataRate"`
	Configuration      [14]string `json:"Configuration"`
	FrequencyRX2       uint32     `json:"FrequencyRX2"`
	DataRateRX2        uint32     `json:"DataRateRX2"`
	MinFrequency       uint32     `json:"MinFrequency"`
	MaxFrequency       uint32     `json:"MaxFrequency"`
	TablePayloadSize   [14][2]int `json:"PayloadSize"`
	TablePayloadSizeDT [14][2]int `json:"PayloadSizeDT"`
}

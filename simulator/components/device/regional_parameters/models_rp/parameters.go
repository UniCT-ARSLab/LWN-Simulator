package models_rp

import c "github.com/arslab/lwnsimulator/simulator/components/device/classes/models_classes"

type Parameters struct {
	Code              int                 `json:"code"`
	MinFrequency      uint32              `json:"minFrequency"`
	MaxFrequency      uint32              `json:"maxFrequency"`
	FrequencyRX2      uint32              `json:"frequencyRX2"`
	DataRateRX2       uint32              `json:"dataRateRX2"`
	MinDataRate       uint8               `json:"minDataRate"`
	MaxDataRate       uint8               `json:"maxDataRate"`
	InfoGroupChannels []InfoGroupChannels `json:"infoGroupChannels"`
	InfoClassB        c.InfoClassB        `json:"infoClassB"`
	MinRX1DROffset    uint8               `json:"minRX1DROffset"`
	MaxRX1DROffset    uint8               `json:"maxRX1DROffset"`
}

type Informations struct {
	MaxRX1DROffset     uint8      `json:"maxRX1DROffset"`
	DataRate           [14]int    `json:"dataRate"`
	Configuration      [14]string `json:"configuration"`
	FrequencyRX2       uint32     `json:"frequencyRX2"`
	DataRateRX2        uint32     `json:"dataRateRX2"`
	MinFrequency       uint32     `json:"minFrequency"`
	MaxFrequency       uint32     `json:"maxFrequency"`
	TablePayloadSize   [14][2]int `json:"payloadSize"`
	TablePayloadSizeDT [14][2]int `json:"payloadSizeDT"`
}

package models_rp

type InfoGroupChannels struct {
	EnableUplink       bool   `json:"enableUplink"`
	InitialFrequency   uint32 `json:"initialFrequency"`
	OffsetFrequency    uint32 `json:"offsetFrequency"`
	MinDataRate        uint8  `json:"minDataRate"`
	MaxDataRate        uint8  `json:"maxDataRate"`
	NbReservedChannels int    `json:"reserved"`
}

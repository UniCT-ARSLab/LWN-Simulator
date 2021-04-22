package models_rp

type InfoGroupChannels struct {
	EnableUplink       bool   `json:"EnableUplink"`
	InitialFrequency   uint32 `json:"InitialFrequency"`
	OffsetFrequency    uint32 `json:"OffsetFrequency"`
	MinDataRate        uint8  `json:"MinDataRate"`
	MaxDataRate        uint8  `json:"MaxDataRate"`
	NbReservedChannels int    `json:"Reserved"`
}

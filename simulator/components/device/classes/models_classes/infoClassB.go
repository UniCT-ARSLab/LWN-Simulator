package models_classes

import (
	"github.com/arslab/lwnsimulator/simulator/components/device/features"
	"github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
)

type InfoClassB struct {
	Periodicity uint8 `json:"periodicity"`

	DataRate        uint8  `json:"dataRate"`
	FrequencyBeacon uint32 `json:"frequencyBeacon"`

	PingSlot features.Window `json:"pingSlot"`
}

func (b *InfoClassB) Setup(freqBeacon uint32, freqPingSlot uint32, datarate uint8, minDr uint8, maxDr uint8) {

	b.FrequencyBeacon = freqBeacon //freq

	channel := channels.Channel{
		Active:            true,
		EnableUplink:      false,
		FrequencyUplink:   freqPingSlot,
		FrequencyDownlink: freqPingSlot,
		MinDR:             minDr,
		MaxDR:             maxDr,
	}

	b.PingSlot.Channel = channel
	b.PingSlot.Delay = 0
	b.PingSlot.DurationOpen = 30 //30 ms

}

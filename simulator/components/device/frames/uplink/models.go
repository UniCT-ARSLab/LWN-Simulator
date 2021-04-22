package uplink

import (
	"github.com/arslab/lwnsimulator/simulator/components/device/features/adr"
	mac "github.com/arslab/lwnsimulator/simulator/components/device/macCommands"
	"github.com/brocaar/lorawan"
)

type Uplink struct {
	DwellTime lorawan.DwellTime `json:"-"`

	ClassB bool              `json:"-"`
	FCnt   uint32            `json:"FCnt"`
	FOpts  []lorawan.Payload `json:"-"`

	FPort         *uint8            `json:"FPort"`
	ADR           adr.ADRInfo       `json:"-"`
	AckMacCommand mac.AckMacCommand `json:"-"` //to create new Uplink

}

type InfoFrame struct {
	MType   lorawan.MType
	Payload lorawan.Payload
}

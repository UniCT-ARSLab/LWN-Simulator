package classes

import (
	"time"

	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	"github.com/arslab/lwnsimulator/simulator/components/device/models"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/brocaar/lorawan"
)

const (
	ModeA = iota
	ModeB
	ModeC
)

type Class interface {
	Setup(*models.InformationDevice)
	SendData(rxpk pkt.RXPK)
	ReceiveWindows(time.Duration, time.Duration) *lorawan.PHYPayload
	RetransmissionCData(downlink *dl.InformationDownlink) error
	RetransmissionUnCData(downlink *dl.InformationDownlink) error
	GetMode() int
	ToString() string
	CloseRX2()
}

type ClassType struct {
	info func() Class
}

var ClassRegistry = map[int]ClassType{
	ModeA: {func() Class { return &ClassA{} }},
	ModeB: {func() Class { return &ClassB{} }},
	ModeC: {func() Class { return &ClassC{} }},
}

func GetClass(Code int) Class {
	r := ClassRegistry[Code]
	return r.info()
}

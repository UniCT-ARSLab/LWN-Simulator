package models

import (
	"github.com/arslab/lwnsimulator/simulator/resources/communication/buffer"
	loc "github.com/arslab/lwnsimulator/simulator/resources/location"
	"github.com/brocaar/lorawan"
)

type InfoDevice struct {
	DevEUI   lorawan.EUI64
	Location loc.Location
	Range    float64
}

type InfoGateway struct {
	MACAddress lorawan.EUI64
	Buffer     *buffer.BufferUplink
	Location   loc.Location
}

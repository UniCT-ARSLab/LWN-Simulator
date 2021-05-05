package models_uplink

import (
	"github.com/brocaar/lorawan"
)

type InfoFrame struct {
	MType   lorawan.MType
	Payload lorawan.Payload
}

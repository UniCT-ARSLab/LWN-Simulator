package types

import (
	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
)

const (
	CodeOK = iota
	CodeErrorName
	CodeErrorAddress
	CodeErrorDeviceActive
	CodeNoBridge
	CodeErrorGatewayActive
	CodeSaving
)

type AddressIP struct {
	Address string `json:"ServerIP"`
	Port    string `json:"Port"`
}

type Gateway struct {
	Gw    gw.Gateway `json:"Gateway"`
	Index int        `json:"Index"`
}

type Device struct {
	Dev   dev.Device `json:"Device"`
	Index int        `json:"Index"`
}

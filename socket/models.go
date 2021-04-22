package socket

import "github.com/brocaar/lorawan"

type ConsoleLog struct {
	Name string `json:"Name"`
	Msg  string `json:"Msg"`
}

type InfoStatus struct {
	DevEUI   lorawan.EUI64   `json:"DevEUI"`
	DevAddr  lorawan.DevAddr `json:"DevAddr"`
	NwkSKey  string          `json:"NwkSKey"`
	AppSKey  string          `json:"AppSKey"`
	FCntDown uint32          `json:"FCntDown"`
	FCnt     uint32          `json:"FCnt"`
}

type NewPayload struct {
	DevEUI  string `json:"DevEUI"`
	MType   string `json:"MType"`
	Payload string `json:"Payload"`
}

type NewLocation struct {
	DevEUI    string  `json:"DevEUI"`
	Latitude  float64 `json:"Latitude"`
	Longitude float64 `json:"Longitude"`
	Altitude  int32   `json:"Altitude"`
}

type MacCommand struct {
	DevEUI      lorawan.EUI64 `json:"DevEUI"`
	CID         string        `json:"CID"`
	Periodicity uint8         `json:"Periodicity"`
}

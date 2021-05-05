package socket

import "github.com/brocaar/lorawan"

type ConsoleLog struct {
	Name string `json:"name"`
	Msg  string `json:"message"`
}

type NewStatusDev struct {
	DevEUI   lorawan.EUI64   `json:"devEUI"`
	DevAddr  lorawan.DevAddr `json:"devAddr"`
	NwkSKey  string          `json:"nwkSKey"`
	AppSKey  string          `json:"appSKey"`
	FCntDown uint32          `json:"fcntDown"`
	FCnt     uint32          `json:"fcnt"`
}

type NewPayload struct {
	Id      int    `json:"id"`
	MType   string `json:"mtype"`
	Payload string `json:"payload"`
}

type NewLocation struct {
	Id        int     `json:"id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  int32   `json:"altitude"`
}

type MacCommand struct {
	Id          int    `json:"id"`
	CID         string `json:"cid"`
	Periodicity uint8  `json:"periodicity"`
}

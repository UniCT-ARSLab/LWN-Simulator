package models

import (
	"encoding/json"

	modelClass "github.com/arslab/lwnsimulator/simulator/components/device/classes/models_classes"
	"github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	up "github.com/arslab/lwnsimulator/simulator/components/device/frames/uplink"
	"github.com/brocaar/lorawan"
)

type Status struct {
	Active bool `json:"Active"`
	Joined bool `json:"-"`

	DataUplink    up.Uplink       `json:"InfoUplink"`
	MType         lorawan.MType   `json:"MType"`   // from UI
	Payload       lorawan.Payload `json:"Payload"` // from UI
	BufferUplinks []up.InfoFrame  `json:"-"`       // from socket

	DataDownlink dl.InformationDownlink `json:"-"`
	FCntDown     uint32                 `json:"FCntDown"`

	DataRate uint8 `json:"DataRate"`
	TXPower  uint8 `json:"TXPower"`
	Battery  uint8 `json:"Battery"`

	InfoClassB         modelClass.InfoClassB      `json:"-"`
	InfoClassC         modelClass.InfoClassC      `json:"-"`
	IndexchannelActive uint16                     `json:"-"`
	InfoChannelsUS915  channels.InfoChannelsUS915 `json:"-"`

	RetransmissionActive        bool          `json:"-"`
	CounterRepConfirmedDataUp   int           `json:"-"`
	CounterRepUnConfirmedDataUp uint8         `json:"-"`
	LastMType                   lorawan.MType `json:"-"`
	LastUplinks                 [][]byte      `json:"-"`
}

func (s *Status) MarshalJSON() ([]byte, error) {

	type Alias Status

	mtype := "UnConfirmedDataUp"
	if s.MType == lorawan.ConfirmedDataUp {
		mtype = "ConfirmedDataUp"
	}

	PayloadBytes, _ := s.Payload.MarshalBinary()

	return json.Marshal(&struct {
		MType   string `json:"MType"`
		Payload string `json:"Payload"`
		*Alias
	}{
		MType:   mtype,
		Payload: string(PayloadBytes),
		Alias:   (*Alias)(s),
	})

}

func (s *Status) UnmarshalJSON(data []byte) error {

	type Alias Status

	aux := &struct {
		MType   string `json:"MType"`
		Payload string `json:"Payload"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.MType == "ConfirmedDataUp" {
		s.MType = lorawan.ConfirmedDataUp
	} else {
		s.MType = lorawan.UnconfirmedDataUp
	}

	s.Payload = &lorawan.DataPayload{
		Bytes: []byte(aux.Payload),
	}

	return nil
}

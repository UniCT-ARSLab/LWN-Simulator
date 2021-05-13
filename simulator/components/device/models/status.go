package models

import (
	"encoding/json"

	modelClass "github.com/arslab/lwnsimulator/simulator/components/device/classes/models_classes"
	"github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	up "github.com/arslab/lwnsimulator/simulator/components/device/frames/uplink"
	mup "github.com/arslab/lwnsimulator/simulator/components/device/frames/uplink/models"
	"github.com/brocaar/lorawan"
)

type Status struct {
	Active bool `json:"active"`
	Joined bool `json:"-"`
	Mode   int  `json:"-"`

	DataUplink    up.InfoUplink   `json:"infoUplink"`
	MType         lorawan.MType   `json:"mtype"`   // from UI
	Payload       lorawan.Payload `json:"payload"` // from UI
	BufferUplinks []mup.InfoFrame `json:"-"`       // from socket

	DataDownlink dl.InformationDownlink `json:"-"`
	FCntDown     uint32                 `json:"fcntDown"`

	DataRate uint8 `json:"-"`
	TXPower  uint8 `json:"-"`
	Battery  uint8 `json:"-"`

	InfoClassB         modelClass.InfoClassB      `json:"-"`
	InfoClassC         modelClass.InfoClassC      `json:"-"`
	IndexchannelActive uint16                     `json:"-"`
	InfoChannelsUS915  channels.InfoChannelsUS915 `json:"-"`

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
		MType   string `json:"mtype"`
		Payload string `json:"payload"`
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
		MType   string `json:"mtype"`
		Payload string `json:"payload"`
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

package models

import (
	"encoding/json"
	"time"

	"github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
	rp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters"
)

// Configuration contains conf of device
type Configuration struct {
	Region rp.Region `json:"region"`

	SendInterval time.Duration `json:"sendInterval"` // interval to send data
	AckTimeout   time.Duration `json:"ackTimeout"`   // timer to wait ack frame

	Range float64 `json:"range"`

	DisableFCntDown bool `json:"disableFCntDown"`

	SupportedOtaa     bool `json:"supportedOtaa"`     //false not supported
	SupportedADR      bool `json:"supportedADR"`      //false not supported
	SupportedFragment bool `json:"supportedFragment"` //fragmentation true, false truncate
	SupportedClassB   bool `json:"supportedClassB"`   //false not supported
	SupportedClassC   bool `json:"supportedClassC"`   //false not supported

	//uplink
	DataRateInitial uint8 `json:"dataRate"`

	//RX1
	RX1DROffset uint8 `json:"rx1DROffset"`

	Channels []channels.Channel `json:"-"`

	NbRepConfirmedDataUp   int   `json:"nbRetransmission"` //Nb retrasmission of ConfirmedDataUp
	NbRepUnconfirmedDataUp uint8 `json:"-"`                // Nb retrasmission of UnconfirmedDataUp

}

func (c *Configuration) MarshalJSON() ([]byte, error) {
	type Alias Configuration

	return json.Marshal(&struct {
		Region       int `json:"region"`
		SendInterval int `json:"sendInterval"`
		AckTimeout   int `json:"ackTimeout"`

		*Alias
	}{
		Region:       c.Region.GetCode(),
		SendInterval: int(c.SendInterval / time.Second),
		AckTimeout:   int(c.AckTimeout / time.Second),

		Alias: (*Alias)(c),
	})

}

func (c *Configuration) UnmarshalJSON(data []byte) error {

	type Alias Configuration

	aux := &struct {
		Region       int `json:"region"`
		SendInterval int `json:"sendInterval"`
		AckTimeout   int `json:"ackTimeout"`

		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	c.Region = rp.GetRegionalParameters(aux.Region)
	c.SendInterval = time.Duration(aux.SendInterval) * time.Second
	c.AckTimeout = time.Duration(aux.AckTimeout) * time.Second

	return nil
}

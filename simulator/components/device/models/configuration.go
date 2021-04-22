package models

import (
	"encoding/json"
	"time"

	"github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
	rp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters"
)

//Configuration contains conf of device
type Configuration struct {
	Region rp.Region `json:"Region"`

	SendInterval time.Duration `json:"SendInterval"` // interval to send data
	AckTimeout   time.Duration `json:"AckTimeout"`   // timer to wait ack frame

	Range float64 `json:"Range"`

	DisableFCntDown bool `json:"DisableFCntDown"`

	SupportedOtaa     bool `json:"SupportedOtaa"`     //false not supported
	SupportedADR      bool `json:"SupportedADR"`      //false not supported
	SupportedFragment bool `json:"SupportedFragment"` //fragmentation true, false truncate
	SupportedClassB   bool `json:"SupportedClassB"`   //false not supported
	SupportedClassC   bool `json:"SupportedClassC"`   //false not supported

	//RX1
	RX1DROffset uint8 `json:"RX1DROffset"`

	Channels []channels.Channel `json:"Channels"`

	NbRepConfirmedDataUp   int   `json:"NbRetransmission"` //Nb retrasmission of ConfirmedDataUp
	NbRepUnconfirmedDataUp uint8 `json:"-"`                // Nb retrasmission of UnconfirmedDataUp

}

func (c *Configuration) MarshalJSON() ([]byte, error) {
	type Alias Configuration

	return json.Marshal(&struct {
		Region       int `json:"Region"`
		SendInterval int `json:"SendInterval"`
		AckTimeout   int `json:"AckTimeout"`

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
		Region       int `json:"Region"`
		SendInterval int `json:"SendInterval"`
		AckTimeout   int `json:"AckTimeout"`

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

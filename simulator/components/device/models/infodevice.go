package models

import (
	"encoding/hex"
	"encoding/json"

	"github.com/arslab/lwnsimulator/simulator/components/device/features"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	f "github.com/arslab/lwnsimulator/simulator/components/forwarder"
	"github.com/arslab/lwnsimulator/simulator/resources/location"
	"github.com/brocaar/lorawan"
)

type InformationDevice struct {
	Name      string            `json:"Name"`
	DevEUI    lorawan.EUI64     `json:"DevEUI"`
	DevAddr   lorawan.DevAddr   `json:"DevAddr"`
	NwkSKey   [16]byte          `json:"NwkSKey"`
	AppSKey   [16]byte          `json:"AppSKey"`
	AppKey    [16]byte          `json:"AppKey"`
	DevNonce  lorawan.DevNonce  `json:"-"`
	JoinNonce lorawan.JoinNonce `json:"-"`
	NetID     lorawan.NetID     `json:"-"`
	JoinEUI   lorawan.EUI64     `json:"-"`

	StateSimulator *uint8        `json:"-"`
	Status         Status        `json:"Status"`
	Configuration  Configuration `json:"Configuration"`

	Location location.Location `json:"Location"`
	RX       []features.Window `json:"RXs"` //RX[0] = rx1 RX[1] = rx2

	Forwarder        *f.Forwarder        `json:"-"`
	ReceivedDownlink dl.ReceivedDownlink `json:"-"`
}

func (d *InformationDevice) MarshalJSON() ([]byte, error) {

	type Alias InformationDevice

	return json.Marshal(&struct {
		DevEUI  string `json:"DevEUI"`
		DevAddr string `json:"DevAddr"`
		NwkSKey string `json:"NwkSKey"`
		AppSKey string `json:"AppSKey"`
		AppKey  string `json:"AppKey"`
		*Alias
	}{
		DevEUI:  hex.EncodeToString(d.DevEUI[:]),
		DevAddr: hex.EncodeToString(d.DevAddr[:]),
		NwkSKey: hex.EncodeToString(d.NwkSKey[:]),
		AppSKey: hex.EncodeToString(d.AppSKey[:]),
		AppKey:  hex.EncodeToString(d.AppKey[:]),
		Alias:   (*Alias)(d),
	})

}

func (d *InformationDevice) UnmarshalJSON(data []byte) error {

	type Alias InformationDevice

	aux := &struct {
		DevEUI  string `json:"DevEUI"`
		DevAddr string `json:"DevAddr"`
		NwkSKey string `json:"NwkSKey"`
		AppSKey string `json:"AppSKey"`
		AppKey  string `json:"AppKey"`

		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	DevEUITmp, _ := hex.DecodeString(aux.DevEUI)
	DevAddrTmp, _ := hex.DecodeString(aux.DevAddr)
	NwkSKeyTmp, _ := hex.DecodeString(aux.NwkSKey)
	AppSKeyTmp, _ := hex.DecodeString(aux.AppSKey)
	AppKeyTmp, _ := hex.DecodeString(aux.AppKey)

	copy(d.DevEUI[:8], DevEUITmp)
	copy(d.DevAddr[:4], DevAddrTmp)
	copy(d.NwkSKey[:16], NwkSKeyTmp)
	copy(d.AppSKey[:16], AppSKeyTmp)
	copy(d.AppKey[:16], AppKeyTmp)

	return nil
}

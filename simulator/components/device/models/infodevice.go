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
	Name      string            `json:"name"`
	DevEUI    lorawan.EUI64     `json:"devEUI"`
	DevAddr   lorawan.DevAddr   `json:"devAddr"`
	NwkSKey   [16]byte          `json:"nwkSKey"`
	AppSKey   [16]byte          `json:"appSKey"`
	AppKey    [16]byte          `json:"appKey"`
	DevNonce  lorawan.DevNonce  `json:"-"`
	JoinNonce lorawan.JoinNonce `json:"-"`
	NetID     lorawan.NetID     `json:"-"`
	JoinEUI   lorawan.EUI64     `json:"-"`

	Status        Status        `json:"status"`
	Configuration Configuration `json:"configuration"`

	Location location.Location `json:"location"`
	RX       []features.Window `json:"rxs"` //RX[0] = rx1 RX[1] = rx2

	Forwarder        *f.Forwarder        `json:"-"`
	ReceivedDownlink dl.ReceivedDownlink `json:"-"`
}

func (d *InformationDevice) MarshalJSON() ([]byte, error) {

	type Alias InformationDevice

	return json.Marshal(&struct {
		DevEUI  string `json:"devEUI"`
		DevAddr string `json:"devAddr"`
		NwkSKey string `json:"nwkSKey"`
		AppSKey string `json:"appSKey"`
		AppKey  string `json:"appKey"`
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
		DevEUI  string `json:"devEUI"`
		DevAddr string `json:"devAddr"`
		NwkSKey string `json:"nwkSKey"`
		AppSKey string `json:"appSKey"`
		AppKey  string `json:"appKey"`

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

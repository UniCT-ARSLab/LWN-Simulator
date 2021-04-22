package models

import (
	"encoding/hex"
	"encoding/json"
	"net"
	"time"

	loc "github.com/arslab/lwnsimulator/simulator/resources/location"
	"github.com/brocaar/lorawan"
)

type InfoGateway struct {
	Active        bool          `json:"Active"`
	TypeGateway   bool          `json:"TypeGateway"` //true real
	Name          string        `json:"Name"`
	MACAddress    lorawan.EUI64 `json:"MACAddress"`
	Location      loc.Location  `json:"Location"`
	KeepAlive     time.Duration `json:"KeepAlive"`
	Connection    *net.UDPConn  `json:"-"`
	AddrIP        string        `json:"Address"`
	Port          string        `json:"Port"`
	BridgeAddress *string       `json:"-"` //is a pointer
}

func (g *InfoGateway) MarshalJSON() ([]byte, error) {

	type Alias InfoGateway

	return json.Marshal(&struct {
		MACAddress string `json:"MACAddress"`
		KeepAlive  int    `json:"KeepAlive"`

		*Alias
	}{
		MACAddress: hex.EncodeToString(g.MACAddress[:]),
		KeepAlive:  int(g.KeepAlive / time.Second),

		Alias: (*Alias)(g),
	})

}

func (g *InfoGateway) UnmarshalJSON(data []byte) error {

	type Alias InfoGateway

	aux := &struct {
		MACAddress string `json:"MACAddress"`
		KeepAlive  int    `json:"KeepAlive"`
		*Alias
	}{
		Alias: (*Alias)(g),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	MACAddressTmp, _ := hex.DecodeString(aux.MACAddress)
	copy(g.MACAddress[:8], MACAddressTmp)

	g.KeepAlive = time.Duration(aux.KeepAlive) * time.Second

	return nil
}

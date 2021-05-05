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
	Active        bool          `json:"active"`
	TypeGateway   bool          `json:"typeGateway"` //true real
	Name          string        `json:"name"`
	MACAddress    lorawan.EUI64 `json:"macAddress"`
	Location      loc.Location  `json:"location"`
	KeepAlive     time.Duration `json:"keepAlive"`
	Connection    *net.UDPConn  `json:"-"`
	AddrIP        string        `json:"ip"`
	Port          string        `json:"port"`
	BridgeAddress *string       `json:"-"` //is a pointer
}

func (g *InfoGateway) MarshalJSON() ([]byte, error) {

	type Alias InfoGateway

	return json.Marshal(&struct {
		MACAddress string `json:"macAddress"`
		KeepAlive  int    `json:"keepAlive"`

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
		MACAddress string `json:"macAddress"`
		KeepAlive  int    `json:"keepAlive"`
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

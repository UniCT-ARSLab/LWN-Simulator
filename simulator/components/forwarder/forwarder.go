package forwarder

import (
	"sync"
	"time"

	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	m "github.com/arslab/lwnsimulator/simulator/components/forwarder/models"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/buffer"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	loc "github.com/arslab/lwnsimulator/simulator/resources/location"
	"github.com/brocaar/lorawan"
)

type Forwarder struct {
	DevToGw  map[lorawan.EUI64]map[lorawan.EUI64]*buffer.BufferUplink            // populates in setup and can update itself
	GwtoDev  map[uint32]map[lorawan.EUI64]map[lorawan.EUI64]*dl.ReceivedDownlink // populates with register/unRegister
	Devices  map[lorawan.EUI64]m.InfoDevice
	Gateways map[lorawan.EUI64]m.InfoGateway
	Mutex    sync.Mutex
}

// GPS offset compensates for the drift between UTC and GPS time
const GPSOffset = 18000

func createPacket(info pkt.RXPK) pkt.RXPK {

	tnow := time.Now()
	offset, _ := time.Parse(time.RFC3339, "1980-01-06T00:00:00Z")
	tmms := tnow.UnixMilli() - offset.UnixMilli() + GPSOffset

	rxpk := pkt.RXPK{

		Time:      tnow.Format(time.RFC3339),
		Tmms:      &tmms,
		Tmst:      uint32(tnow.Unix()),
		Channel:   info.Channel,
		RFCH:      0,
		Frequency: info.Frequency,
		Stat:      1,
		Modu:      info.Modu,
		DatR:      info.DatR,
		Brd:       0,
		CodR:      info.CodR,
		RSSI:      info.RSSI,
		LSNR:      7,
		Size:      info.Size,
		Data:      info.Data,
	}

	return rxpk
}

func inRange(d m.InfoDevice, g m.InfoGateway) bool {

	distance := loc.GetDistance(d.Location.Latitude, d.Location.Longitude,
		g.Location.Latitude, g.Location.Longitude)

	if distance <= (d.Range / 1000.0) {
		return true
	}

	return false
}

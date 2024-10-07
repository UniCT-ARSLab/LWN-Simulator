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

// Forwarder allows communication between devices and gateways
type Forwarder struct {
	DevToGw  map[lorawan.EUI64]map[lorawan.EUI64]*buffer.BufferUplink            // It populates within the setup and it gets updated
	GwtoDev  map[uint32]map[lorawan.EUI64]map[lorawan.EUI64]*dl.ReceivedDownlink // It populates within the event of register or deregister
	Devices  map[lorawan.EUI64]m.InfoDevice                                      // A list of device information indexed by DevEUI
	Gateways map[lorawan.EUI64]m.InfoGateway                                     // A list of gateway information indexed by GatewayID
	Mutex    sync.Mutex                                                          // Mutex for the forwarder
}

// GPSOffset compensates for the drift between UTC and GPS time
const GPSOffset = 18000

// createPacket creates a packet from the received packet
func createPacket(info pkt.RXPK) pkt.RXPK {
	// Calculate the time of the packet
	now := time.Now()
	offset, _ := time.Parse(time.RFC3339, "1980-01-06T00:00:00Z")
	tmms := now.UnixMilli() - offset.UnixMilli() + GPSOffset
	// Create the packet
	rxpk := pkt.RXPK{
		Time:      now.Format(time.RFC3339),
		Tmms:      &tmms,
		Tmst:      uint32(now.Unix()),
		Channel:   info.Channel,
		RFCH:      0,
		Frequency: info.Frequency,
		Stat:      1,
		Modu:      info.Modu,
		DatR:      info.DatR,
		Brd:       0,
		CodR:      info.CodR,
		RSSI:      -60, // TODO: Make it variable during the simulation
		LSNR:      7,
		Size:      info.Size,
		Data:      info.Data,
	}
	return rxpk
}

// inRange checks if the device is in range of the gateway
func inRange(d m.InfoDevice, g m.InfoGateway) bool {
	distance := loc.GetDistance(d.Location.Latitude, d.Location.Longitude,
		g.Location.Latitude, g.Location.Longitude)
	return distance <= (d.Range / 1000.0)
}

package forwarder

import (
	"fmt"
	"github.com/arslab/lwnsimulator/shared"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	m "github.com/arslab/lwnsimulator/simulator/components/forwarder/models"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/buffer"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/brocaar/lorawan"
)

// Setup initializes the Forwarder by initializing the maps and returning a pointer to the Forwarder
func Setup() *Forwarder {
	shared.DebugPrint("Init new Forwarder instance")
	f := Forwarder{
		DevToGw:  make(map[lorawan.EUI64]map[lorawan.EUI64]*buffer.BufferUplink),            //1[devEUI] 2 [macAddress]
		GwtoDev:  make(map[uint32]map[lorawan.EUI64]map[lorawan.EUI64]*dl.ReceivedDownlink), //1[fre1] 2 [macAddress] 3[devEUI]
		Devices:  make(map[lorawan.EUI64]m.InfoDevice),
		Gateways: make(map[lorawan.EUI64]m.InfoGateway),
	}
	return &f
}

// AddDevice adds a device to the Forwarder and update the DevToGw map
func (f *Forwarder) AddDevice(d m.InfoDevice) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	shared.DebugPrint(fmt.Sprintf("Add device %v to Forwarder", d.DevEUI))
	f.Devices[d.DevEUI] = d
	inner := make(map[lorawan.EUI64]*buffer.BufferUplink)
	f.DevToGw[d.DevEUI] = inner
	for _, g := range f.Gateways {
		if inRange(d, g) {
			shared.DebugPrint(fmt.Sprintf("Adding communication link with %s", g.MACAddress))
			f.DevToGw[d.DevEUI][g.MACAddress] = g.Buffer
		}
	}
}

// AddGateway adds a gateway to the Forwarder and update the DevToGw map
func (f *Forwarder) AddGateway(g m.InfoGateway) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	shared.DebugPrint(fmt.Sprintf("Add/Update gateway %v to Forwarder", g.MACAddress))
	f.Gateways[g.MACAddress] = g
	for _, d := range f.Devices {
		if inRange(d, g) {
			shared.DebugPrint(fmt.Sprintf("Adding communication link with %s", d.DevEUI))
			f.DevToGw[d.DevEUI][g.MACAddress] = g.Buffer
		}
	}
}

// DeleteDevice removes a device from the Forwarder and update the DevToGw map
func (f *Forwarder) DeleteDevice(DevEUI lorawan.EUI64) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	shared.DebugPrint(fmt.Sprintf("Delete device %v from Forwarder", DevEUI))
	clear(f.DevToGw[DevEUI])
	delete(f.DevToGw, DevEUI)
	delete(f.Devices, DevEUI)
}

// DeleteGateway removes a gateway from the Forwarder and update the DevToGw map
func (f *Forwarder) DeleteGateway(g m.InfoGateway) {
	f.Mutex.Lock()
	defer f.Mutex.Unlock()
	shared.DebugPrint(fmt.Sprintf("Delete gateway %v from Forwarder", g.MACAddress))
	for _, d := range f.Devices {
		shared.DebugPrint(fmt.Sprintf("Removing communication link with %s", d.DevEUI))
		delete(f.DevToGw[d.DevEUI], g.MACAddress)
	}
	delete(f.Gateways, g.MACAddress)
}

// UpdateDevice updates a device in the Forwarder
func (f *Forwarder) UpdateDevice(d m.InfoDevice) {
	f.AddDevice(d)
}

func (f *Forwarder) Register(freq uint32, devEUI lorawan.EUI64, rDownlink *dl.ReceivedDownlink) {

	f.Mutex.Lock()

	inner, ok := f.GwtoDev[freq]
	if !ok {
		inner = make(map[lorawan.EUI64]map[lorawan.EUI64]*dl.ReceivedDownlink)
		f.GwtoDev[freq] = inner
	}

	for key := range f.DevToGw[devEUI] {

		inner, ok := f.GwtoDev[freq][key]
		if !ok {
			inner = make(map[lorawan.EUI64]*dl.ReceivedDownlink)
			f.GwtoDev[freq][key] = inner
		}

		rDownlink.Open()
		f.GwtoDev[freq][key][devEUI] = rDownlink

	}

	f.Mutex.Unlock()

}

func (f *Forwarder) UnRegister(freq uint32, devEUI lorawan.EUI64) {

	f.Mutex.Lock()

	for key := range f.DevToGw[devEUI] {

		_, ok := f.GwtoDev[freq][key][devEUI]
		if ok {

			f.GwtoDev[freq][key][devEUI].Close()
			delete(f.GwtoDev[freq][key], devEUI)

		}

	}

	f.Mutex.Unlock()

}

func (f *Forwarder) Uplink(data pkt.RXPK, DevEUI lorawan.EUI64) {

	f.Mutex.Lock()

	rxpk := createPacket(data)

	for _, up := range f.DevToGw[DevEUI] {
		up.Push(rxpk)
	}

	f.Mutex.Unlock()

}

func (f *Forwarder) Downlink(data *lorawan.PHYPayload, freq uint32, macAddress lorawan.EUI64) {

	f.Mutex.Lock()

	for _, dl := range f.GwtoDev[freq][macAddress] {
		dl.Push(data)
	}

	f.Mutex.Unlock()

}

// Reset resets the Forwarder by creating a new instance of the Forwarder
func (f *Forwarder) Reset() {
	shared.DebugPrint("Reset Forwarder")
	clear(f.DevToGw)
	clear(f.GwtoDev)
	clear(f.Devices)
	clear(f.Gateways)
}

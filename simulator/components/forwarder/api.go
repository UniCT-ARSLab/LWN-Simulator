package forwarder

import (
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	m "github.com/arslab/lwnsimulator/simulator/components/forwarder/models"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/buffer"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/brocaar/lorawan"
)

func Setup() *Forwarder {

	f := Forwarder{
		DevToGw:  make(map[lorawan.EUI64]map[lorawan.EUI64]*buffer.BufferUplink),            //1[devEUI] 2 [macAddress]
		GwtoDev:  make(map[uint32]map[lorawan.EUI64]map[lorawan.EUI64]*dl.ReceivedDownlink), //1[fre1] 2 [macAddress] 3[devEUI]
		Devices:  make(map[lorawan.EUI64]m.InfoDevice),
		Gateways: make(map[lorawan.EUI64]m.InfoGateway),
	}

	return &f

}

func (f *Forwarder) AddDevice(d m.InfoDevice) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	f.Devices[d.DevEUI] = d

	inner := make(map[lorawan.EUI64]*buffer.BufferUplink)
	f.DevToGw[d.DevEUI] = inner

	for _, g := range f.Gateways {

		if inRange(d, g) {
			f.DevToGw[d.DevEUI][g.MACAddress] = g.Buffer
		}

	}

}

func (f *Forwarder) AddGateway(g m.InfoGateway) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	f.Gateways[g.MACAddress] = g

	for _, d := range f.Devices {

		if inRange(d, g) {
			f.DevToGw[d.DevEUI][g.MACAddress] = g.Buffer
		}

	}
}

func (f *Forwarder) DeleteDevice(DevEUI lorawan.EUI64) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	for key := range f.DevToGw[DevEUI] {
		delete(f.DevToGw[DevEUI], key)
	}

	delete(f.DevToGw, DevEUI)
	delete(f.Devices, DevEUI)

}

func (f *Forwarder) DeleteGateway(g m.InfoGateway) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	for _, d := range f.Devices {
		delete(f.DevToGw[d.DevEUI], g.MACAddress)
	}

	delete(f.Gateways, g.MACAddress)

}

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

func (f *Forwarder) Reset() {
	f = Setup()
}

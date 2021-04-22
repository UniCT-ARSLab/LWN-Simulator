package forwarder

import (
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	m "github.com/arslab/lwnsimulator/simulator/components/forwarder/models"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/buffer"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	loc "github.com/arslab/lwnsimulator/simulator/resources/location"
	"github.com/brocaar/lorawan"
)

func Setup(dev []m.InfoDevice, gw []m.InfoGateway) *Forwarder {
	f := Forwarder{
		DevToGw: make(map[lorawan.EUI64][]*buffer.BufferUplink),
		GwtoDev: make(map[uint32][]*dl.ReceivedDownlink),
	}

	f.devices = append(dev)
	f.gateways = append(gw)

	for _, d := range dev {

		for _, g := range gw {

			distance := loc.GetDistance(d.Location.Latitude, d.Location.Longitude,
				g.Location.Latitude, g.Location.Longitude)

			if distance <= (d.Range / 1000.0) {
				f.DevToGw[d.DevEUI] = append(f.DevToGw[d.DevEUI], g.Buf)
			}

		}

	}
	return &f
}

func (f *Forwarder) AddDevice(d m.InfoDevice) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	f.devices = append(f.devices, d)

	for _, g := range f.gateways {

		distance := loc.GetDistance(d.Location.Latitude, d.Location.Longitude,
			g.Location.Latitude, g.Location.Longitude)

		if distance <= (d.Range / 1000.0) {
			f.DevToGw[d.DevEUI] = append(f.DevToGw[d.DevEUI], g.Buf)
		}

	}
}

func (f *Forwarder) AddGateway(g m.InfoGateway) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	f.gateways = append(f.gateways, g)

	for _, d := range f.devices {

		distance := loc.GetDistance(d.Location.Latitude, d.Location.Longitude,
			g.Location.Latitude, g.Location.Longitude)

		if distance <= (d.Range / 1000.0) {
			f.DevToGw[d.DevEUI] = append(f.DevToGw[d.DevEUI], g.Buf)
		}

	}
}

func (f *Forwarder) DeleteDevice(d m.InfoDevice) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	for range f.DevToGw[d.DevEUI] {
		delete(f.DevToGw, d.DevEUI)
	}
	index := -1
	for i, dev := range f.devices {
		if dev.DevEUI == d.DevEUI {
			index = i
			break
		}
	}

	switch index {
	case 0:
		f.devices = f.devices[1:]
	case len(f.devices) - 1:
		f.devices = f.devices[:len(f.devices)-1]
	default:
		f.devices = append(f.devices[:index], f.devices[index+1:]...)
	}
}

func (f *Forwarder) DeleteGateway(g m.InfoGateway) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	for _, d := range f.devices {
		for i, buf := range f.DevToGw[d.DevEUI] {
			if buf == g.Buf {

				switch i {
				case 0:
					f.DevToGw[d.DevEUI] = f.DevToGw[d.DevEUI][1:]
				case len(f.DevToGw[d.DevEUI]) - 1:
					f.DevToGw[d.DevEUI] = f.DevToGw[d.DevEUI][:len(f.DevToGw[d.DevEUI])-1]
				default:
					f.DevToGw[d.DevEUI] = append(f.DevToGw[d.DevEUI][:i], f.DevToGw[d.DevEUI][i+1:]...)
				}

			}

		}
	}

	index := -1
	for i, gw := range f.gateways {
		if gw.Buf == g.Buf {
			index = i
			break
		}
	}

	switch index {
	case 0:
		f.gateways = f.gateways[1:]
	case len(f.gateways) - 1:
		f.gateways = f.gateways[:len(f.gateways)-1]
	default:
		f.gateways = append(f.gateways[:index], f.gateways[index+1:]...)
	}
}

func (f *Forwarder) UpdateDevice(info m.InfoDevice) {

	f.Mutex.Lock()
	defer f.Mutex.Unlock()

	for i, d := range f.devices {
		if d.DevEUI == info.DevEUI {
			f.devices[i] = info

			for range f.DevToGw[info.DevEUI] { //map empty
				delete(f.DevToGw, info.DevEUI)
			}

			for _, g := range f.gateways {

				distance := loc.GetDistance(info.Location.Latitude, info.Location.Longitude,
					g.Location.Latitude, g.Location.Longitude)

				if distance <= (d.Range / 1000.0) {
					f.DevToGw[d.DevEUI] = append(f.DevToGw[d.DevEUI], g.Buf)
				}

			}

			break
		}

	}

}

func (f *Forwarder) Register(freq uint32, DevEUI lorawan.EUI64, buf *dl.ReceivedDownlink) {
	f.Mutex.Lock()
	f.GwtoDev[freq] = append(f.GwtoDev[freq], buf)
	f.Mutex.Unlock()
}

func (f *Forwarder) UnRegister(freq uint32, buf *dl.ReceivedDownlink) {

	f.Mutex.Lock()

	for i := 0; i < len(f.GwtoDev[freq]); {

		dl := f.GwtoDev[freq][i]

		if dl == buf {

			switch i {
			case 0:

				if len(f.GwtoDev[freq]) == 1 {
					f.GwtoDev[freq] = f.GwtoDev[freq][:0]
				} else {
					f.GwtoDev[freq] = f.GwtoDev[freq][1:]
				}

				break

			case len(f.GwtoDev[freq]) - 1:

				f.GwtoDev[freq] = f.GwtoDev[freq][:len(f.GwtoDev[freq])-1]
				break

			default:

				f.GwtoDev[freq] = append(f.GwtoDev[freq][:i], f.GwtoDev[freq][i+1:]...)
				break

			}

		} else {
			i++
		}
	}

	f.Mutex.Unlock()

}

func (f *Forwarder) Uplink(data pkt.RXPK, index lorawan.EUI64) {

	rxpk := createPacket(data)
	f.Mutex.Lock()
	for _, up := range f.DevToGw[index] {
		up.Push(rxpk)
	}
	f.Mutex.Unlock()
}

func (f *Forwarder) Downlink(data *lorawan.PHYPayload, freq uint32) {

	f.Mutex.Lock()
	for _, dl := range f.GwtoDev[freq] {
		dl.Push(data)
	}
	f.Mutex.Unlock()

}

func (f *Forwarder) Reset() {
	f = &Forwarder{
		DevToGw: make(map[lorawan.EUI64][]*buffer.BufferUplink),
		GwtoDev: make(map[uint32][]*dl.ReceivedDownlink),
	}
}

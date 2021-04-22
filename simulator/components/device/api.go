package device

import (
	"errors"

	"github.com/brocaar/lorawan"

	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	up "github.com/arslab/lwnsimulator/simulator/components/device/frames/uplink"
	f "github.com/arslab/lwnsimulator/simulator/components/forwarder"
	res "github.com/arslab/lwnsimulator/simulator/resources"
	"github.com/arslab/lwnsimulator/simulator/util"
)

func (d *Device) Setup(Resources *res.Resources, StateSimulator *uint8, forwarder *f.Forwarder) {

	d.Info.StateSimulator = StateSimulator
	d.Info.JoinEUI = lorawan.EUI64{0, 0, 0, 0, 0, 0, 0, 0}
	d.Info.NetID = lorawan.NetID{0, 0, 0}

	if !d.Info.Configuration.SupportedOtaa { //ABP
		d.Info.Status.Joined = true
	} else {
		d.Info.Status.Joined = false
	}

	d.Info.Configuration.Region.Setup()

	d.Info.Status.DataUplink.ADR.Setup(d.Info.Configuration.SupportedADR)
	d.Info.Status.DataUplink.DwellTime = lorawan.DwellTime400ms
	d.Info.Status.Battery = util.ConnectedPowerSource
	d.Info.Status.InfoChannelsUS915.ListChanLastPass = [8]int{-1, -1, -1, -1, -1, -1, -1, -1}

	d.Info.Status.CounterRepUnConfirmedDataUp = 1
	d.Info.Configuration.NbRepUnconfirmedDataUp = 1

	//class C
	if d.Info.Configuration.SupportedClassC {
		d.Info.Status.InfoClassC.Setup()
	}

	d.Resources = Resources
	d.Info.Forwarder = forwarder

	d.Info.ReceivedDownlink = dl.ReceivedDownlink{
		Notify: make(chan struct{}),
	}

	d.Info.Configuration.Channels = d.Info.Configuration.Region.GetChannels()

	d.Mode = classes.GetClass(classes.ModeA)
	d.Mode.Setup(&d.Info)

	d.Print("Setup OK!", nil, util.PrintBoth)

}

func (d *Device) OnStart() {
	go d.Run()
}

func (d *Device) TurnOFF() {

	d.Mutex.Lock()

	d.Info.Status.Active = false

	d.Mutex.Unlock()

}

func (d *Device) TurnON() {

	d.Info.Status.Active = true

	go d.Run()

	d.Print("Turn ON", nil, util.PrintBoth)
}

func (d *Device) SendMACCommand(cid lorawan.CID, periodicity uint8) error {
	var command []lorawan.Payload

	if cid == lorawan.PingSlotInfoReq {

		if !d.Info.Configuration.SupportedClassB {
			return errors.New("Device don't support Class B")
		}

		if d.Mode.GetMode() == classes.ModeB {
			d.SwitchClass(classes.ModeA)
		}

		command = []lorawan.Payload{
			&lorawan.MACCommand{
				CID: cid,
				Payload: &lorawan.PingSlotInfoReqPayload{
					Periodicity: periodicity,
				},
			},
		}
		d.Info.Status.InfoClassB.Periodicity = periodicity

	} else {

		command = []lorawan.Payload{
			&lorawan.MACCommand{
				CID: cid,
			},
		}

	}

	d.newMACComands(command)

	return nil
}

func (d *Device) NewUplink(mtype lorawan.MType, payload string) {

	FRMPayload := &lorawan.DataPayload{
		Bytes: []byte(payload),
	}

	info := up.InfoFrame{
		MType:   mtype,
		Payload: FRMPayload,
	}

	d.Info.Status.BufferUplinks = append(d.Info.Status.BufferUplinks, info)

}

func (d *Device) ChangePayload(mtype lorawan.MType, payload lorawan.Payload) {

	d.Info.Status.MType = mtype
	d.Info.Status.Payload = payload

}

func (d *Device) ChangeLocation(lat float64, lng float64, alt int32) {

	d.Info.Location.Latitude = lat
	d.Info.Location.Longitude = lng
	d.Info.Location.Altitude = alt

}

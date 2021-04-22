package classes

import (
	"errors"
	"fmt"
	"time"

	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	"github.com/arslab/lwnsimulator/simulator/components/device/models"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/brocaar/lorawan"
)

const (
	//BeaconPeriod is a period beacons
	BeaconPeriod = 128
	//Pingslots is number of ping slots in one period beacon
	Pingslots = 4096
	//PingDuration is a duration of a slot (ms)
	PingDuration = 30
	//TimeoutClassB is a timeout by last dl (s)
	TimeoutClassB = 120
)

//ClassB è implementata come la classe A
type ClassB struct {
	Info *models.InformationDevice
}

func (b *ClassB) Setup(info *models.InformationDevice) {
	b.Info = info
}

func (b *ClassB) SendData(rxpk pkt.RXPK) {

	var indexChannelRX1 int

	b.Info.Forwarder.Uplink(rxpk, b.Info.DevEUI)

	b.Info.RX[0].DataRate, indexChannelRX1 = b.Info.Configuration.Region.SetupRX1(
		b.Info.Status.DataRate, b.Info.Configuration.RX1DROffset,
		int(b.Info.Status.IndexchannelActive), b.Info.Status.DataDownlink.DwellTime)

	b.Info.RX[0].Channel = b.Info.Configuration.Channels[indexChannelRX1]
}

func (b *ClassB) ReceiveWindows(delayRX1 time.Duration, delayRX2 time.Duration) *lorawan.PHYPayload {

	for i := 0; i < 2; i++ {

		var delay time.Duration
		if i == 0 {
			delay = delayRX1
		} else {
			delay = delayRX2
		}

		b.Info.Forwarder.Register(b.Info.RX[i].GetListeningFrequency(), b.Info.DevEUI, &b.Info.ReceivedDownlink)

		resp := b.Info.RX[i].OpenWindow(delay, &b.Info.ReceivedDownlink)

		b.Info.Forwarder.UnRegister(b.Info.RX[i].GetListeningFrequency(), &b.Info.ReceivedDownlink)

		if resp != nil {
			return resp
		}

	}

	return nil

}

func (b *ClassB) RetransmissionCData(downlink *dl.InformationDownlink) error {

	if b.Info.Status.CounterRepConfirmedDataUp < b.Info.Configuration.NbRepConfirmedDataUp {

		if downlink != nil { ///downlink ricevuto

			if downlink.ACK { // ACK ricevuto
				b.Info.Status.CounterRepConfirmedDataUp = 0
				b.Info.Status.RetransmissionActive = false
				return nil
			}

		}

		b.Info.Status.RetransmissionActive = true
		b.Info.Status.CounterRepConfirmedDataUp++
		//nessun ACK ricevuto
		return nil
	} else {

		b.Info.Status.RetransmissionActive = false
		b.Info.Status.CounterRepConfirmedDataUp = 0
		err := fmt.Sprintf("Last Uplink sent %v times", b.Info.Configuration.NbRepConfirmedDataUp)

		return errors.New(err)

	}

}

func (b *ClassB) RetransmissionUnCData(downlink *dl.InformationDownlink) error {

	if b.Info.Status.CounterRepUnConfirmedDataUp < b.Info.Configuration.NbRepUnconfirmedDataUp {

		b.Info.Status.RetransmissionActive = true
		b.Info.Status.CounterRepUnConfirmedDataUp++

		return nil

	} else {

		b.Info.Status.CounterRepUnConfirmedDataUp = 1

		if b.Info.Status.RetransmissionActive {

			b.Info.Status.RetransmissionActive = false

			err := fmt.Sprintf("Last Uplink sent %v times", b.Info.Configuration.NbRepUnconfirmedDataUp)
			return errors.New(err)

		}

		return nil

	}

}

func (b *ClassB) GetMode() int {
	return ModeB
}

func (b *ClassB) ToString() string {
	return "B"
}

func (b *ClassB) CloseRX2() {}

package classes

import (
	"errors"
	"fmt"
	"time"

	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/brocaar/lorawan"

	"github.com/arslab/lwnsimulator/simulator/components/device/models"
)

type TypeA struct {
	Info *models.InformationDevice
}

func (a *TypeA) Setup(info *models.InformationDevice) {
	a.Info = info
}

func (a *TypeA) SendData(rxpk pkt.RXPK) {

	var indexChannelRX1 int

	a.Info.Forwarder.Uplink(rxpk, a.Info.DevEUI)

	a.Info.RX[0].DataRate, indexChannelRX1 = a.Info.Configuration.Region.SetupRX1(
		a.Info.Status.DataRate, a.Info.Configuration.RX1DROffset,
		int(a.Info.Status.IndexchannelActive), a.Info.Status.DataDownlink.DwellTime)

	a.Info.RX[0].Channel = a.Info.Configuration.Channels[indexChannelRX1]

}

func (a *TypeA) ReceiveWindows(delayRX1 time.Duration, delayRX2 time.Duration) *lorawan.PHYPayload {

	for i := 0; i < 2; i++ {

		var delay time.Duration
		if i == 0 {
			delay = delayRX1
		} else {
			delay = delayRX2
		}

		a.Info.Forwarder.Register(a.Info.RX[i].GetListeningFrequency(), a.Info.DevEUI, &a.Info.ReceivedDownlink)

		resp := a.Info.RX[i].OpenWindow(delay, &a.Info.ReceivedDownlink)

		a.Info.Forwarder.UnRegister(a.Info.RX[i].GetListeningFrequency(), a.Info.DevEUI)

		if resp != nil {
			return resp
		}

	}

	return nil

}

func (a *TypeA) RetransmissionCData(downlink *dl.InformationDownlink) error {

	if a.Info.Status.CounterRepConfirmedDataUp < a.Info.Configuration.NbRepConfirmedDataUp {

		if downlink != nil { ///downlink ricevuto

			if downlink.ACK { // ACK ricevuto
				a.Info.Status.CounterRepConfirmedDataUp = 0
				a.Info.Status.Mode = util.Normal
				return nil
			}

		}

		a.Info.Status.Mode = util.Retransmission
		a.Info.Status.CounterRepConfirmedDataUp++
		//nessun ACK ricevuto
		return nil
	} else {

		a.Info.Status.Mode = util.Normal
		a.Info.Status.CounterRepConfirmedDataUp = 0
		err := fmt.Sprintf("Last Uplink sent %v times", a.Info.Configuration.NbRepConfirmedDataUp)

		return errors.New(err)

	}

}

func (a *TypeA) RetransmissionUnCData(downlink *dl.InformationDownlink) error {

	if a.Info.Status.CounterRepUnConfirmedDataUp < a.Info.Configuration.NbRepUnconfirmedDataUp {

		a.Info.Status.Mode = util.Retransmission
		a.Info.Status.CounterRepUnConfirmedDataUp++

		return nil

	}

	var err error
	err = nil

	if a.Info.Status.Mode == util.Retransmission {

		a.Info.Status.Mode = util.Normal
		err = errors.New(fmt.Sprintf("Last Uplink sent %v times", a.Info.Status.CounterRepUnConfirmedDataUp))

	}

	a.Info.Status.CounterRepUnConfirmedDataUp = 1

	return err

}

func (a *TypeA) GetClass() int {
	return ClassA
}

func (a *TypeA) ToString() string {
	return "A"
}

func (a *TypeA) CloseRX2() {}

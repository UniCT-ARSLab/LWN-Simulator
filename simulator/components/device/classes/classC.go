package classes

import (
	"errors"
	"fmt"
	"sync"
	"time"

	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	"github.com/arslab/lwnsimulator/simulator/components/device/models"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/brocaar/lorawan"
)

//ClassC mode
type ClassC struct {
	Info      *models.InformationDevice
	Supported bool `json:"Supported"`

	Mux      sync.Mutex `json:"-"`
	Open     bool       `json:"-"`
	CondOpen *sync.Cond `json:"-"`
}

func (c *ClassC) Setup(info *models.InformationDevice) {
	c.Info = info
	c.CondOpen = sync.NewCond(&c.Mux)
	go c.RX2()
}

func (c *ClassC) SendData(rxpk pkt.RXPK) {

	var indexChannelRX1 int

	c.CloseWindow()
	defer c.OpenWindow()

	c.Info.Forwarder.Uplink(rxpk, c.Info.DevEUI)

	c.Info.RX[0].DataRate, indexChannelRX1 = c.Info.Configuration.Region.SetupRX1(
		c.Info.Status.DataRate, c.Info.Configuration.RX1DROffset,
		int(c.Info.Status.IndexchannelActive), c.Info.Status.DataDownlink.DwellTime)

	c.Info.RX[0].Channel = c.Info.Configuration.Channels[indexChannelRX1]
}

func (c *ClassC) ReceiveWindows(delayRX1 time.Duration, delayRX2 time.Duration) *lorawan.PHYPayload {

	c.CloseWindow()
	defer c.OpenWindow()

	c.Info.Forwarder.Register(c.Info.RX[0].GetListeningFrequency(), c.Info.DevEUI, &c.Info.ReceivedDownlink)

	resp := c.Info.RX[0].OpenWindow(0, &c.Info.ReceivedDownlink)

	c.Info.Forwarder.UnRegister(c.Info.RX[0].GetListeningFrequency(), &c.Info.ReceivedDownlink)

	return resp

}

func (c *ClassC) RetransmissionCData(downlink *dl.InformationDownlink) error {

	if c.Info.Status.CounterRepConfirmedDataUp < c.Info.Configuration.NbRepConfirmedDataUp {

		if downlink != nil { ///downlink ricevuto

			if downlink.ACK { // ACK ricevuto
				c.Info.Status.CounterRepConfirmedDataUp = 0
				c.Info.Status.RetransmissionActive = false
				return nil
			}

		}

		c.Info.Status.RetransmissionActive = true
		c.Info.Status.CounterRepConfirmedDataUp++
		//nessun ACK ricevuto
		return nil

	} else {

		c.Info.Status.RetransmissionActive = false
		c.Info.Status.CounterRepConfirmedDataUp = 0
		err := fmt.Sprintf("Last Uplink sent %v times", c.Info.Configuration.NbRepConfirmedDataUp)

		return errors.New(err)

	}

}

func (c *ClassC) RetransmissionUnCData(downlink *dl.InformationDownlink) error {

	if c.Info.Status.CounterRepUnConfirmedDataUp < c.Info.Configuration.NbRepUnconfirmedDataUp {

		c.Info.Status.RetransmissionActive = true
		c.Info.Status.CounterRepConfirmedDataUp++
		//nessun ACK ricevuto
		return nil

	} else {

		var err string

		c.Info.Status.CounterRepConfirmedDataUp = 0

		if c.Info.Status.RetransmissionActive {

			c.Info.Status.RetransmissionActive = false
			err = fmt.Sprintf("Last Uplink sent %v times", c.Info.Configuration.NbRepConfirmedDataUp)

		}

		return errors.New(err)

	}

}

func (c *ClassC) GetMode() int {
	return ModeC
}

func (c *ClassC) ToString() string {
	return "C"
}

func (c *ClassC) RX2() {

	for {

		c.Info.Forwarder.Register(c.Info.RX[1].GetListeningFrequency(), c.Info.DevEUI, &c.Info.ReceivedDownlink)

		state := c.GetStateWindow()
		if !state {

			c.Info.Forwarder.UnRegister(c.Info.RX[1].GetListeningFrequency(), &c.Info.ReceivedDownlink)

			c.Mux.Lock()
			c.CondOpen.Wait()
			c.Mux.Unlock()

		} else {

			select {

			case <-c.Info.Status.InfoClassC.Exit:
				return

			case <-c.Info.ReceivedDownlink.Notify:

				state := c.GetStateWindow()
				if !state {

					c.Info.ReceivedDownlink.Notify <- struct{}{}
					c.Info.Forwarder.UnRegister(c.Info.RX[1].GetListeningFrequency(), &c.Info.ReceivedDownlink)

					c.Mux.Lock()
					c.CondOpen.Wait()
					c.Mux.Unlock()

					continue
				}

				phy := c.Info.ReceivedDownlink.Pull()
				if phy != nil { //response

					downlink, err := dl.GetDownlink(*phy, c.Info.Configuration.DisableFCntDown, c.Info.Status.FCntDown,
						c.Info.NwkSKey, c.Info.AppSKey)
					if err != nil {
						continue
					}

					c.Info.Status.InfoClassC.InsertDownlink(*downlink)
					c.Info.Status.InfoClassC.WaitClass()

				}

			}

		}

	}

}

func (c *ClassC) OpenWindow() {

	c.Mux.Lock()
	c.Open = true
	c.CondOpen.Signal()

	c.Mux.Unlock()

}

func (c *ClassC) CloseWindow() {

	c.Mux.Lock()
	c.Open = false
	c.Mux.Unlock()

}

func (c *ClassC) GetStateWindow() bool {

	c.Mux.Lock()
	defer c.Mux.Unlock()

	return c.Open
}

func (c *ClassC) CloseRX2() {
	c.Mux.Lock()
	c.CondOpen.Signal()
	c.Mux.Unlock()
}

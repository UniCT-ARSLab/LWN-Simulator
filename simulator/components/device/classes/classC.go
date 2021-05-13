package classes

import (
	"errors"
	"fmt"
	"sync"
	"time"

	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	"github.com/arslab/lwnsimulator/simulator/components/device/models"
	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/brocaar/lorawan"
)

const (
	Close = iota
	Open
	Exit
)

//TypeC mode
type TypeC struct {
	Info      *models.InformationDevice
	Supported bool `json:"supported"`

	Mutex    sync.Mutex `json:"-"`
	Open     int        `json:"-"`
	CondOpen *sync.Cond `json:"-"`
}

func (c *TypeC) Setup(info *models.InformationDevice) {
	c.Info = info
	c.CondOpen = sync.NewCond(&c.Mutex)
	go c.RX2()
}

func (c *TypeC) SendData(rxpk pkt.RXPK) {

	var indexChannelRX1 int

	c.CloseWindow()
	defer c.OpenWindow()

	c.Info.Forwarder.Uplink(rxpk, c.Info.DevEUI)

	c.Info.RX[0].DataRate, indexChannelRX1 = c.Info.Configuration.Region.SetupRX1(
		c.Info.Status.DataRate, c.Info.Configuration.RX1DROffset,
		int(c.Info.Status.IndexchannelActive), c.Info.Status.DataDownlink.DwellTime)

	c.Info.RX[0].Channel = c.Info.Configuration.Channels[indexChannelRX1]
}

func (c *TypeC) ReceiveWindows(delayRX1 time.Duration, delayRX2 time.Duration) *lorawan.PHYPayload {

	c.CloseWindow()
	defer c.OpenWindow()

	c.Info.Forwarder.Register(c.Info.RX[0].GetListeningFrequency(), c.Info.DevEUI, &c.Info.ReceivedDownlink)

	resp := c.Info.RX[0].OpenWindow(0, &c.Info.ReceivedDownlink)

	c.Info.Forwarder.UnRegister(c.Info.RX[0].GetListeningFrequency(), c.Info.DevEUI)

	return resp

}

func (c *TypeC) RetransmissionCData(downlink *dl.InformationDownlink) error {

	if c.Info.Status.CounterRepConfirmedDataUp < c.Info.Configuration.NbRepConfirmedDataUp {

		if downlink != nil { ///downlink ricevuto

			if downlink.ACK { // ACK ricevuto
				c.Info.Status.CounterRepConfirmedDataUp = 0
				c.Info.Status.Mode = util.Normal
				return nil
			}

		}

		c.Info.Status.Mode = util.Retransmission
		c.Info.Status.CounterRepConfirmedDataUp++
		//nessun ACK ricevuto
		return nil

	} else {

		c.Info.Status.Mode = util.Normal
		c.Info.Status.CounterRepConfirmedDataUp = 0
		err := fmt.Sprintf("Last Uplink sent %v times", c.Info.Configuration.NbRepConfirmedDataUp)

		return errors.New(err)

	}

}

func (c *TypeC) RetransmissionUnCData(downlink *dl.InformationDownlink) error {

	if c.Info.Status.CounterRepUnConfirmedDataUp < c.Info.Configuration.NbRepUnconfirmedDataUp {

		c.Info.Status.Mode = util.Retransmission
		c.Info.Status.CounterRepUnConfirmedDataUp++
		//nessun ACK ricevuto
		return nil

	}

	var err error
	err = nil

	if c.Info.Status.Mode == util.Retransmission {

		c.Info.Status.Mode = util.Normal
		err = errors.New(fmt.Sprintf("Last Uplink sent %v times", c.Info.Status.CounterRepUnConfirmedDataUp))

	}

	c.Info.Status.CounterRepUnConfirmedDataUp = 1

	return err

}

func (c *TypeC) GetClass() int {
	return ClassC
}

func (c *TypeC) ToString() string {
	return "C"
}

func (c *TypeC) RX2() {

	c.Info.Forwarder.Register(c.Info.RX[1].GetListeningFrequency(), c.Info.DevEUI, &c.Info.ReceivedDownlink)

	for {

		switch c.isOpenWindow() {
		case Exit:
			c.Info.Forwarder.UnRegister(c.Info.RX[1].GetListeningFrequency(), c.Info.DevEUI)
			return

		case Close:
			c.Info.Forwarder.UnRegister(c.Info.RX[1].GetListeningFrequency(), c.Info.DevEUI)

			c.Mutex.Lock()
			c.CondOpen.Wait()
			c.Mutex.Unlock()

			c.Info.Forwarder.Register(c.Info.RX[1].GetListeningFrequency(), c.Info.DevEUI, &c.Info.ReceivedDownlink)

			continue
		}

		c.Info.ReceivedDownlink.Wait()

		switch c.isOpenWindow() {
		case Exit:
			c.Info.Forwarder.UnRegister(c.Info.RX[1].GetListeningFrequency(), c.Info.DevEUI)
			return

		case Close:
			c.Info.Forwarder.UnRegister(c.Info.RX[1].GetListeningFrequency(), c.Info.DevEUI)

			c.Mutex.Lock()
			c.CondOpen.Wait()
			c.Mutex.Unlock()

			c.Info.Forwarder.Register(c.Info.RX[1].GetListeningFrequency(), c.Info.DevEUI, &c.Info.ReceivedDownlink)

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

func (c *TypeC) OpenWindow() {

	c.Mutex.Lock()
	c.Open = Open
	c.CondOpen.Broadcast()

	c.Mutex.Unlock()

}

func (c *TypeC) CloseWindow() {

	c.Mutex.Lock()
	c.Open = Close
	c.Mutex.Unlock()

}

func (c *TypeC) isOpenWindow() int {

	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	return c.Open
}

func (c *TypeC) CloseRX2() {
	c.Mutex.Lock()
	c.Open = Exit
	c.CondOpen.Broadcast()
	c.Mutex.Unlock()
}

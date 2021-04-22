package device

import (
	"math/rand"
	"time"

	"github.com/arslab/lwnsimulator/simulator/util"

	act "github.com/arslab/lwnsimulator/simulator/components/device/activation"
	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	"github.com/brocaar/lorawan"
)

const (
	JOINACCEPTDELAY1 = time.Duration(5 * time.Second)
	JOINACCEPTDELAY2 = time.Duration(1 * time.Second)
)

func (d *Device) OtaaActivation() {

	for !d.Info.Status.Joined {

		ok := d.CanExecute()
		if !ok { //stop simulator
			return
		}

		d.SwitchClass(classes.ModeA)

		d.SendJoinRequest()

		d.Print("Open RXs", nil, util.PrintOnlySocket)

		phy := d.Mode.ReceiveWindows(JOINACCEPTDELAY1, JOINACCEPTDELAY2)
		if phy != nil {

			d.Print("Downlink Received", nil, util.PrintOnlySocket)

			_, err := d.ProcessDownlink(*phy)
			if err != nil {
				d.Print("", err, util.PrintOnlySocket)

				timerAckTimeout := time.NewTimer(d.Info.Configuration.AckTimeout)
				<-timerAckTimeout.C

				d.Print("ACK Timeout", nil, util.PrintOnlySocket)
			}
		}

		if d.Info.Status.Joined {

			d.Print("Joined", nil, util.PrintBoth)
			return
		}

		d.Print("Unjoined", nil, util.PrintBoth)

	}

	return
}

func (d *Device) CreateJoinRequest() []byte {

	rand.Seed(time.Now().UTC().UnixNano())
	random := uint16(rand.Int())

	DevNonce := lorawan.DevNonce(random)
	d.Info.DevNonce = DevNonce

	phy := lorawan.PHYPayload{
		MHDR: lorawan.MHDR{
			MType: lorawan.JoinRequest,
			Major: lorawan.LoRaWANR1,
		},
		MACPayload: &lorawan.JoinRequestPayload{
			JoinEUI:  d.Info.JoinEUI, // appEUI
			DevEUI:   d.Info.DevEUI,
			DevNonce: d.Info.DevNonce,
		},
	}

	if err := phy.SetUplinkJoinMIC(d.Info.AppKey); err != nil {

		d.Print("", err, util.PrintBoth)

		return []byte{}
	}

	bytes, err := phy.MarshalBinary()
	if err != nil {

		d.Print("", err, util.PrintBoth)

		return []byte{}
	}

	return bytes

}

func (d *Device) ProcessJoinAccept(JoinAccPayload *lorawan.JoinAcceptPayload) (*dl.InformationDownlink, error) {

	var downlink dl.InformationDownlink
	var err error

	//setkeys
	d.Info.NwkSKey, err = act.GetKey(d.Info.NetID, JoinAccPayload.JoinNonce, d.Info.DevNonce, d.Info.AppKey, act.PadNwkSKey)
	d.Info.AppSKey, err = act.GetKey(d.Info.NetID, JoinAccPayload.JoinNonce, d.Info.DevNonce, d.Info.AppKey, act.PadAppSKey)

	if err != nil {
		return nil, err
	}

	d.Info.Status.Joined = true

	//cflist
	if JoinAccPayload.CFList != nil {

		cflist, err := JoinAccPayload.CFList.Payload.MarshalBinary()
		if err != nil {
			d.Print("", nil, util.PrintBoth)
		}

		if JoinAccPayload.CFList.CFListType == lorawan.CFListChannel { //list of channel

			var CFList lorawan.CFListChannelPayload

			err = CFList.UnmarshalBinary(false, cflist)
			if err != nil {
				d.Print("", nil, util.PrintBoth)
			}

			for i, c := range CFList.Channels {
				index := i + d.Info.Configuration.Region.GetNbReservedChannels()
				d.setChannel(uint8(index), c, 0, 5)
			}

		} else { //list of ChMask

			var CFList lorawan.CFListChannelMaskPayload
			err = CFList.UnmarshalBinary(false, cflist)
			if err != nil {
				d.Print("", nil, util.PrintBoth)
			}

			for i, mask := range CFList.ChannelMasks {
				for j, enable := range mask {
					index := j + i*16
					d.Info.Configuration.Channels[index].EnableUplink = enable
				}
			}

		}
	}

	d.Info.JoinNonce = JoinAccPayload.JoinNonce
	d.Info.DevAddr = JoinAccPayload.DevAddr
	d.Info.NetID = JoinAccPayload.HomeNetID
	d.Info.RX[0].Delay = time.Duration(JoinAccPayload.RXDelay)
	d.Info.RX[1].Delay = time.Duration(JoinAccPayload.RXDelay)
	d.Info.Configuration.RX1DROffset = JoinAccPayload.DLSettings.RX1DROffset
	d.Info.RX[1].DataRate = JoinAccPayload.DLSettings.RX2DataRate
	downlink.MType = lorawan.JoinAccept

	return &downlink, nil
}

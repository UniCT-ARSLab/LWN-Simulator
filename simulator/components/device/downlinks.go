package device

import (
	"github.com/arslab/lwnsimulator/simulator/util"

	act "github.com/arslab/lwnsimulator/simulator/components/device/activation"
	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	"github.com/brocaar/lorawan"
)

func (d *Device) ProcessDownlink(phy lorawan.PHYPayload) (*dl.InformationDownlink, error) {

	var payload *dl.InformationDownlink
	var err error

	mtype := phy.MHDR.MType
	err = nil

	switch mtype {

	case lorawan.JoinAccept:
		Ja, err := act.DecryptJoinAccept(phy, d.Info.DevNonce, d.Info.JoinEUI, d.Info.AppKey)
		if err != nil {
			return nil, err
		}

		return d.ProcessJoinAccept(Ja)

	case lorawan.UnconfirmedDataDown:

		payload, err = dl.GetDownlink(phy, d.Info.Configuration.DisableFCntDown, d.Info.Status.FCntDown,
			d.Info.NwkSKey, d.Info.AppSKey)
		if err != nil {
			return nil, err
		}

	case lorawan.ConfirmedDataDown: //ack

		payload, err = dl.GetDownlink(phy, d.Info.Configuration.DisableFCntDown, d.Info.Status.FCntDown,
			d.Info.NwkSKey, d.Info.AppSKey)
		if err != nil {
			return nil, err
		}

		d.SendAck()

	}

	d.Info.Status.FCntDown = (d.Info.Status.FCntDown + 1) % util.MAXFCNTGAP

	switch d.Class.GetClass() {

	case classes.ClassA:
		d.Info.Status.DataUplink.AckMacCommand.CleanFOptsRXParamSetupAns()
		d.Info.Status.DataUplink.AckMacCommand.CleanFOptsRXTimingSetupAns()
		break

	case classes.ClassC:
		d.Info.Status.InfoClassC.SetACK(false) //Reset

	}

	msg := d.Info.Status.DataUplink.ADR.Reset()
	if msg != "" {
		d.Print(msg, nil, util.PrintBoth)
	}

	d.Info.Status.DataUplink.AckMacCommand.CleanFOptsDLChannelAns()

	return payload, err
}

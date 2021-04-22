package device

import (
	act "github.com/arslab/lwnsimulator/simulator/components/device/activation"
	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	"github.com/brocaar/lorawan"
)

func (d *Device) ProcessDownlink(phy lorawan.PHYPayload) (*dl.InformationDownlink, error) {

	downlink := false
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
		downlink = true

	case lorawan.ConfirmedDataDown: //ack
		payload, err = dl.GetDownlink(phy, d.Info.Configuration.DisableFCntDown, d.Info.Status.FCntDown,
			d.Info.NwkSKey, d.Info.AppSKey)
		downlink = true

		d.SendAck()

	}

	if downlink {

		if d.Info.Configuration.SupportedClassC {
			d.Info.Status.InfoClassC.SetACK(false) //Reset
		}

		d.Info.Status.DataUplink.ADR.Reset()

		d.Info.Status.DataUplink.AckMacCommand.CleanFOptsDLChannelAns()

		if d.Mode.GetMode() == classes.ModeA {
			d.Info.Status.DataUplink.AckMacCommand.CleanFOptsRXParamSetupAns()
			d.Info.Status.DataUplink.AckMacCommand.CleanFOptsRXTimingSetupAns()
		}

	}

	return payload, err
}

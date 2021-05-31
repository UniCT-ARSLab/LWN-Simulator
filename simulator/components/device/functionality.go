package device

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	"github.com/arslab/lwnsimulator/simulator/components/device/features/adr"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	rp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/brocaar/lorawan"
)

func (d *Device) Execute() {

	var downlink *dl.InformationDownlink
	var err error

	err = nil
	downlink = nil

	d.SwitchChannel()

	uplinks := d.CreateUplink()
	for i := 0; i < len(uplinks); i++ {

		data := d.SetInfo(uplinks[i], false)
		d.Class.SendData(data)

		d.Print("Uplink sent", nil, util.PrintBoth)
	}

	d.Print("Open RXs", nil, util.PrintBoth)
	phy := d.Class.ReceiveWindows(0, 0)

	if phy != nil {

		d.Print("Downlink Received", nil, util.PrintBoth)

		downlink, err = d.ProcessDownlink(*phy)
		if err != nil {
			d.Print("", err, util.PrintBoth)
			return
		}

		if downlink != nil { //downlink ricevuto

			d.ExecuteMACCommand(*downlink)

			if d.Info.Status.Mode != util.Retransmission {
				d.FPendingProcedure(downlink)
			}

		}

	} else {

		d.Print("None downlinks Received", nil, util.PrintBoth)

		timerAckTimeout := time.NewTimer(d.Info.Configuration.AckTimeout)
		<-timerAckTimeout.C
		d.Print("ACK Timeout", nil, util.PrintBoth)

	}

	d.ADRProcedure()

	//retransmission
	switch d.Info.Status.LastMType {

	case lorawan.ConfirmedDataUp:

		if d.Class.GetClass() == classes.ClassC {
			if d.Info.Status.InfoClassC.GetACK() {
				return
			}
		}

		err := d.Class.RetransmissionCData(downlink)
		if err != nil {

			d.Print("", err, util.PrintBoth)

			d.UnJoined()

		}

		if d.Info.Status.Mode == util.Retransmission {

			d.Info.Status.DataRate = rp.DecrementDataRate(d.Info.Configuration.Region, d.Info.Status.DataRate)

		}

	case lorawan.UnconfirmedDataUp:

		err := d.Class.RetransmissionUnCData(downlink)
		if err != nil {
			d.Print("", err, util.PrintBoth)
		}
	}

}

func (d *Device) FPendingProcedure(downlink *dl.InformationDownlink) {

	var err error
	if !d.CanExecute() {
		return
	}

	startProcedure := 0 //per la print finale

	for downlink != nil {

		if downlink.FPending {

			d.Print("Fpending set", nil, util.PrintBoth)

			if startProcedure == 0 {
				d.Info.Status.Mode = util.FPending
				d.Print("Start FPending procedure", nil, util.PrintBoth)
				startProcedure = 1
			}

			if downlink.MType == lorawan.UnconfirmedDataDown {
				d.SendEmptyFrame()
			}
			//ack sent in resolveDownlinks ergo open Receive Windows

			d.Print("Open RXs", nil, util.PrintBoth)
			phy := d.Class.ReceiveWindows(0, 0)

			if !d.CanExecute() { //stop
				return
			}

			if phy != nil {

				d.Print("Downlink Received", nil, util.PrintBoth)

				downlink, err = d.ProcessDownlink(*phy)
				if err != nil {
					d.Print("", err, util.PrintBoth)

				}

				if downlink != nil { //downlink ricevuto

					d.ExecuteMACCommand(*downlink)

				}

			} else {

				downlink = nil

				d.Print("None downlinks Received", nil, util.PrintBoth)

				timerAckTimeout := time.NewTimer(d.Info.Configuration.AckTimeout)
				<-timerAckTimeout.C

				d.Print("ACK Timeout", nil, util.PrintBoth)

			}

			d.ADRProcedure()

		} else {
			d.Print("Fpending unset", nil, util.PrintBoth)
			break
		}

	}

	if startProcedure == 1 {
		d.Print("FPending procedure finished", nil, util.PrintBoth)
	}

	d.Info.Status.Mode = util.Normal

}

func (d *Device) ADRProcedure() {

	dr, code := d.Info.Status.DataUplink.ADR.ADRProcedure(d.Info.Status.DataRate, d.Info.Configuration.Region, d.Info.Configuration.SupportedADR)

	switch code {

	case adr.CodeNoneError:
		d.Info.Status.DataRate = dr
		break

	case adr.CodeADRFlagReqSet:
		d.Print("SET ADRACKReq flag", nil, util.PrintBoth)
		break

	case adr.CodeUnjoined:
		if UnJoined := d.UnJoined(); UnJoined {

			d.OtaaActivation()

			msg := d.Info.Status.DataUplink.ADR.Reset()
			if msg != "" {
				d.Print(msg, nil, util.PrintBoth)
			}

		}
	}

}

func (d *Device) SwitchChannel() {

	rand.Seed(time.Now().UTC().UnixNano())

	lenChannels := len(d.Info.Configuration.Channels)
	chanUsed := make(map[int]bool)
	lenTrue := 1

	var random int
	var indexGroup int
	regionCode := d.Info.Configuration.Region.GetCode()

	if regionCode == rp.Code_Us915 {

		indexGroup = int(d.Info.Status.IndexchannelActive / 8)

		switch indexGroup {

		case 0:

			if d.Info.Status.InfoChannelsUS915.FirstPass {
				d.Info.Status.InfoChannelsUS915.FirstPass = false
			} else {
				indexGroup++
			}

			break

		case 1, 2, 3, 4, 5, 6:
			indexGroup++
			break

		case 7:

			random = indexGroup + 64

			msg := fmt.Sprintf("Switch channel from %v to %v", d.Info.Status.IndexchannelActive, random)
			d.Print(msg, nil, util.PrintBoth)

			d.Info.Status.IndexchannelActive = uint16(random)
			return

		default:
			indexGroup = 0
			break
		}

		lenChannels = 8

	}

	for lenTrue != lenChannels {

		//random
		if regionCode == rp.Code_Us915 {

			random = (rand.Int() % 8) + indexGroup*8

			for random == d.Info.Status.InfoChannelsUS915.ListChannelsLastPass[indexGroup] {
				random = (rand.Int() % 8) + indexGroup*8
			}

		} else {
			random = rand.Int() % lenChannels
		}

		if !chanUsed[random] { //evita il loop infinito

			if d.Info.Configuration.Channels[random].Active &&
				d.Info.Configuration.Channels[random].EnableUplink { //attivo e enable Uplink

				if d.Info.Configuration.Channels[random].IsSupportedDR(d.Info.Status.DataRate) == nil {

					oldindex := d.Info.Status.IndexchannelActive

					if oldindex != uint16(random) {
						d.Info.Status.IndexchannelActive = uint16(random)

						msg := fmt.Sprintf("Switch channel from %v to %v", oldindex, random)
						d.Print(msg, nil, util.PrintBoth)

						d.Info.Status.InfoChannelsUS915.ListChannelsLastPass[indexGroup] = random // lo fa anche se region non è US_915 (no problem)

						return
					}

				}

			}

			chanUsed[random] = true
			lenTrue++

		}

	}

	if lenTrue == lenChannels { //nessun canale abilitato all'uplink supporta il DataRate

		var msg string
		oldindex := d.Info.Status.IndexchannelActive

		d.Print("No channels available", nil, util.PrintBoth)

		if regionCode == rp.Code_Us915 {

			d.Info.Status.InfoChannelsUS915.ListChannelsLastPass[indexGroup] = indexGroup * 8
			d.Info.Status.IndexchannelActive = uint16(indexGroup * 8)
			d.Info.Configuration.Channels[d.Info.Status.IndexchannelActive].EnableUplink = true

			msg = fmt.Sprintf("Channel %v enabled to send uplinks", d.Info.Status.IndexchannelActive)
			d.Print(msg, nil, util.PrintBoth)

		} else {
			d.Info.Status.IndexchannelActive = uint16(0)
		}

		d.Info.Status.DataRate = d.Info.Configuration.Channels[d.Info.Status.IndexchannelActive].MaxDR
		if oldindex == d.Info.Status.IndexchannelActive {
			msg = fmt.Sprintf("Use channel[%v] with dataRate %v", d.Info.Status.IndexchannelActive, d.Info.Status.DataRate)
		} else {
			msg = fmt.Sprintf("Switch channel from %v to %v with DataRate %v", oldindex, d.Info.Status.IndexchannelActive, d.Info.Status.DataRate)
		}

		d.Print(msg, nil, util.PrintBoth)

		return
	}

}

func (d *Device) SwitchClass(class int) {

	if class == d.Class.GetClass() {
		return
	}

	switch class {

	case classes.ClassA:
		d.Class = classes.GetClass(classes.ClassA)
		d.Class.Setup(&d.Info)

	case classes.ClassB:

		d.Class = classes.GetClass(classes.ClassB)
		d.Class.Setup(&d.Info)

	case classes.ClassC:

		d.Class = classes.GetClass(classes.ClassC)
		d.Class.Setup(&d.Info)
		go d.DownlinkReceivedRX2ClassC()

	default:
		d.Print("Class not Supported", nil, util.PrintBoth)

	}

	msg := fmt.Sprintf("Switch in class %v", d.Class.ToString())
	d.Print(msg, nil, util.PrintBoth)

}

//se il dispositivo non supporta OTAA non può essere unjoined
func (d *Device) UnJoined() bool {

	if d.Info.Configuration.SupportedOtaa {
		d.Info.Status.Joined = false
		return true //Otaa
	}
	return false //ABP

}

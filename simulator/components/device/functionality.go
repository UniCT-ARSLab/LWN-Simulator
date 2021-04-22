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

		ok := d.CanExecute()
		if !ok { //stop
			return
		}

		d.Mode.SendData(data)

		d.Print("Uplink sent", nil, util.PrintBoth)
	}

	d.Print("Open RXs", nil, util.PrintBoth)

	ok := d.CanExecute()
	if !ok { //stop
		return
	}

	phy := d.Mode.ReceiveWindows(0, 0)

	ok = d.CanExecute()
	if !ok { //stop
		return
	}

	if phy != nil {

		d.Print("Downlink Received", nil, util.PrintBoth)

		downlink, err = d.ProcessDownlink(*phy)
		if err != nil {
			d.Print("", err, util.PrintOnlySocket)
			return
		}

		if downlink != nil { //downlink ricevuto

			d.ExecuteMACCommand(*downlink)

			d.ADRProcedure()

			if !d.Info.Status.RetransmissionActive {
				d.FPendingProcedure(*downlink)
			}

		}

	} else {
		d.Print("None downlinks Received", nil, util.PrintBoth)
		timerAckTimeout := time.NewTimer(d.Info.Configuration.AckTimeout)
		<-timerAckTimeout.C

		d.Print("ACK Timeout", nil, util.PrintOnlySocket)

	}

	//retransmission
	switch d.Info.Status.LastMType {

	case lorawan.ConfirmedDataUp:

		if d.Mode.GetMode() == classes.ModeC {
			if d.Info.Status.InfoClassC.GetACK() {
				return
			}
		}

		err := d.Mode.RetransmissionCData(downlink)
		if err != nil {

			d.Print("", err, util.PrintOnlySocket)

			d.UnJoined()

		}

		if d.Info.Status.RetransmissionActive {
			//datarate Lower
			if d.Info.Status.DataRate > d.Info.Configuration.Region.GetMinDataRate() {

				drString := ""
				datarate := d.Info.Status.DataRate - 1

				for drString == "" || datarate > d.Info.Configuration.Region.GetMinDataRate()-1 {
					_, drString = d.Info.Configuration.Region.GetDataRate(datarate)
					if drString != "" {
						d.Info.Status.DataRate = datarate
						break
					}

					datarate--
				}

			}
		}

	case lorawan.UnconfirmedDataUp:

		err := d.Mode.RetransmissionUnCData(downlink)
		if err != nil {
			d.Print("", err, util.PrintOnlySocket)
		}
	}

}

//FPendingProcedure is a procedure for fPrending bit
func (d *Device) FPendingProcedure(downlink dl.InformationDownlink) {

	ok := d.CanExecute()
	if !ok {
		return
	}

	i := 0 //per la print

	for downlink.FPending {

		d.Print("Fpending set", nil, util.PrintOnlySocket)

		if downlink.MType == lorawan.UnconfirmedDataDown {
			d.SendEmptyFrame()
		}

		//ack is managed in resolveDownlinks

		phy := d.Mode.ReceiveWindows(0, 0)

		ok = d.CanExecute()
		if !ok { //stop
			return
		}

		if phy != nil {

			d.Print("Downlink Received", nil, util.PrintOnlySocket)

			downlink, err := d.ProcessDownlink(*phy)
			if err != nil {
				d.Print("", err, util.PrintOnlySocket)
			}

			if downlink != nil { //downlink ricevuto

				d.ExecuteMACCommand(*downlink)

				d.ADRProcedure()

			}

		} else {

			timerAckTimeout := time.NewTimer(d.Info.Configuration.AckTimeout)
			<-timerAckTimeout.C

			d.Print("ACK Timeout", nil, util.PrintOnlySocket)
			d.SendEmptyFrame()
		}

		i = 1

	}

	if i > 0 {
		d.Print("Fpending Unset", nil, util.PrintOnlySocket)
	}

}

//ADRProcedure per incrementare la qualità della rete
func (d *Device) ADRProcedure() {

	switch d.Info.Status.DataUplink.ADR.ADRACKCnt {

	case adr.ADRACKLIMIT, adr.ADRACKLIMIT + adr.ADRACKDELAY:

		if d.Info.Status.DataRate > d.Info.Configuration.Region.GetMinDataRate() && d.Info.Configuration.SupportedADR {
			d.Info.Status.DataUplink.ADR.ADRACKReq = true
		}

	}

	if d.Info.Status.DataUplink.ADR.ADRACKCnt%adr.ADRACKDELAY == 0 &&
		d.Info.Status.DataUplink.ADR.ADRACKCnt > adr.ADRACKLIMIT {

		if d.Info.Status.DataRate > d.Info.Configuration.Region.GetMinDataRate() {

			drString := ""
			datarate := d.Info.Status.DataRate - 1

			for drString == "" || datarate > d.Info.Configuration.Region.GetMinDataRate()-1 {

				_, drString = d.Info.Configuration.Region.GetDataRate(datarate)
				if drString != "" {
					d.Info.Status.DataRate = datarate
					break
				}
				datarate--
			}

		} else {

			if UnJoined := d.UnJoined(); UnJoined {

				d.OtaaActivation()

				d.Info.Status.DataUplink.ADR.Reset()

			}
		}

	}

}

//SwitchChannel cambia il canale casualmente
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

		if indexGroup == 7 {
			random = d.Info.Status.InfoChannelsUS915.Pass + 64
			d.Info.Status.InfoChannelsUS915.Pass = (d.Info.Status.InfoChannelsUS915.Pass + 1) % 8

			msg := fmt.Sprintf("Switch channel from %v to %v", d.Info.Status.IndexchannelActive, random)
			d.Print(msg, nil, util.PrintOnlySocket)

			d.Info.Status.IndexchannelActive = uint16(random)
			return
		} else if indexGroup > 7 {
			indexGroup = -1
		}

		lenChannels = 8
	}

	indexGroup++

	for lenTrue != lenChannels {

		if regionCode == rp.Code_Us915 {

			random = (rand.Int() % 8) + indexGroup*8

			for random == d.Info.Status.InfoChannelsUS915.ListChanLastPass[indexGroup] {
				random = (rand.Int() % 8) + indexGroup*8
			}

		} else {
			random = rand.Int() % lenChannels
		}

		if !chanUsed[random] { //evita il loop infinito
			chanUsed[random] = true
			lenTrue++
		}

		if !d.Info.Configuration.Channels[random].Active { //Incative
			continue
		}

		if d.Info.Configuration.Channels[random].EnableUplink { //attivo e enable Uplink
			if d.Info.Configuration.Channels[random].IsSupportedDR(d.Info.Status.DataRate) == nil {

				oldindex := d.Info.Status.IndexchannelActive

				if oldindex != uint16(random) {
					d.Info.Status.IndexchannelActive = uint16(random)

					msg := fmt.Sprintf("Switch channel from %v to %v", oldindex, random)
					d.Print(msg, nil, util.PrintOnlySocket)

					d.Info.Status.InfoChannelsUS915.ListChanLastPass[indexGroup] = random
					return
				}

			}

		}

	}

	if lenTrue == lenChannels { //nessun canale abilitato all'uplink supporta il DataRate
		oldindex := d.Info.Status.IndexchannelActive

		if regionCode == rp.Code_Us915 {
			d.Info.Status.InfoChannelsUS915.ListChanLastPass[indexGroup] = d.Info.Status.InfoChannelsUS915.Pass * 8
			d.Info.Status.IndexchannelActive = uint16(d.Info.Status.InfoChannelsUS915.Pass * 8)
		} else {
			d.Info.Status.IndexchannelActive = uint16(0)
		}

		msg := fmt.Sprintf("None channel supports DataRate %v", d.Info.Status.DataRate)
		d.Print(msg, nil, util.PrintOnlySocket)

		d.Info.Status.DataRate = d.Info.Configuration.Channels[d.Info.Status.IndexchannelActive].MaxDR

		msg = fmt.Sprintf("Switch channel from %v to %v with DataRate %v", oldindex, d.Info.Status.IndexchannelActive, d.Info.Status.DataRate)
		d.Print(msg, nil, util.PrintOnlySocket)

		return
	}

}

//SwitchClass switch class of device
func (d *Device) SwitchClass(class int) {

	if class == d.Mode.GetMode() {
		return
	}

	switch class {

	case classes.ModeA:
		d.Mode = classes.GetClass(classes.ModeA)
		d.Mode.Setup(&d.Info)

	case classes.ModeB:

		d.Mode = classes.GetClass(classes.ModeB)
		d.Mode.Setup(&d.Info)

	case classes.ModeC:

		d.Mode = classes.GetClass(classes.ModeC)
		d.Mode.Setup(&d.Info)
		go d.DownlinkReceivedRX2ClassC()

	default:
		d.Print("Class not Supported", nil, util.PrintOnlySocket)

	}

	msg := fmt.Sprintf("Switch in class %v", d.Mode.ToString())
	d.Print(msg, nil, util.PrintOnlySocket)

}

//se il dispositivo non supporta OTAA non può essere unjoined
func (d *Device) UnJoined() bool {

	if d.Info.Configuration.SupportedOtaa {
		d.Info.Status.Joined = false
		return true //Otaa
	}
	return false //ABP

}

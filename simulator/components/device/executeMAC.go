package device

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	"github.com/arslab/lwnsimulator/simulator/components/device/features"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	mac "github.com/arslab/lwnsimulator/simulator/components/device/macCommands"
	rp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/brocaar/lorawan"
)

const (
	//MaxMargin is max value for margin (DevStatusReq)
	MaxMargin = int8(64)
)

//***************** MANAGE EXECUTE MAC COMMAND ******************
//*********************Uplink***********************************
//uplink
func (d *Device) newMACComands(CmdS []lorawan.Payload) {

	nCommand := len(CmdS) + len(d.Info.Status.DataUplink.FOpts)
	if nCommand > 15 {

		msg := fmt.Sprintf("Insert %d MACCommands(max 15)", nCommand)
		d.Print(msg, nil, util.PrintBoth)

		return
	}

	d.Info.Status.DataUplink.FOpts = append(d.Info.Status.DataUplink.FOpts, CmdS...)

}

//*********************downlink***********************************

func (d *Device) ExecuteMACCommand(downlink dl.InformationDownlink) {

	if !d.CanExecute() {
		return
	}

	var LinkADRReqCommands [][]byte
	msg := ""

	if len(downlink.FOptsReceived) == 0 {
		msg = "None MAC Command"
	} else {
		msg = "Execute MAC Commands"
	}

	d.Print(msg, nil, util.PrintBoth)

	for _, cmd := range downlink.FOptsReceived {

		cid, payloadBytes, err := mac.ParseMACCommand(cmd, false)
		if err != nil {
			d.Print("", err, util.PrintBoth)
			return
		}

		switch cid {
		case lorawan.LinkCheckAns:
			d.executeLinkCheckAns(payloadBytes)
		case lorawan.LinkADRReq:
			LinkADRReqCommands = append(LinkADRReqCommands, payloadBytes)
		case lorawan.DutyCycleReq:
			d.executeDutyCycleReq(payloadBytes)
		case lorawan.RXParamSetupReq:
			d.executeRXParamSetupReq(payloadBytes)
		case lorawan.DevStatusReq:
			d.executeDevStatusReq()
		case lorawan.NewChannelReq:
			d.executeNewChannelReq(payloadBytes)
		case lorawan.RXTimingSetupReq:
			d.executeRXTimingSetupReq(payloadBytes)
		case lorawan.DLChannelReq:
			d.executeDLChannelReq(payloadBytes)
		case lorawan.TXParamSetupReq:
			d.executeTXParamSetupReq(payloadBytes)
		case lorawan.DeviceTimeAns:
			d.executeDeviceTimeAns(payloadBytes)
		case lorawan.PingSlotChannelReq:
			d.executePingSlotChannelReq(payloadBytes)
		case lorawan.PingSlotInfoAns:
			d.executePingSlotInfoAns(payloadBytes)
		case lorawan.BeaconFreqReq:
			d.executeBeaconFreqReq(payloadBytes)
		}

	}

	if len(LinkADRReqCommands) != 0 {
		d.executeLinkADRReq(LinkADRReqCommands)
	}

}

func (d *Device) executeLinkCheckAns(payload []byte) {

	c := lorawan.LinkCheckAnsPayload{}
	err := c.UnmarshalBinary(payload)
	if err != nil {
		d.Print("", err, util.PrintBoth)
		return
	}

	msg := fmt.Sprintf("LinkCheckAns | Margin[%v], GwCnt[%v] |", c.Margin, c.GwCnt)
	d.Print(msg, nil, util.PrintBoth)

}

func (d *Device) executeLinkADRReq(commands [][]byte) {

	var TXPower uint8
	var NbRep uint8

	result := true
	DataRate := -1
	channels := d.Info.Configuration.Channels

	for _, cmd := range commands {

		var response []lorawan.Payload

		c := lorawan.LinkADRReqPayload{}
		err := c.UnmarshalBinary(cmd)
		if err != nil {

			d.Print("", err, util.PrintBoth)
			return

		}

		acks, errs := d.Info.Configuration.Region.LinkAdrReq(c.Redundancy.ChMaskCntl,
			c.ChMask, c.DataRate, &channels)

		if len(errs) != 0 {

			for _, err := range errs {
				msg := PrintMACCommand("LinkADRReq", err.Error())
				d.Print(msg, nil, util.PrintBoth)
			}

		} else {
			msg := PrintMACCommand("LinkADRReq", "Command is valid")
			d.Print(msg, nil, util.PrintBoth)

			DataRate = int(c.DataRate)
			TXPower = c.TXPower
			NbRep = c.Redundancy.NbRep

		}

		response = []lorawan.Payload{
			&lorawan.MACCommand{
				CID: lorawan.LinkADRAns,
				Payload: &lorawan.LinkADRAnsPayload{
					ChannelMaskACK: acks[0],
					DataRateACK:    acks[1],
					PowerACK:       acks[2],
				},
			},
		}

		d.newMACComands(response)

		result = result && acks[0] && acks[1] && acks[2]

	}

	if result {

		d.Info.Status.DataRate = uint8(DataRate)
		msg := fmt.Sprintf("Set new Datarate: %v", d.Info.Status.DataRate)
		d.Print(msg, nil, util.PrintBoth)

		d.Info.Status.TXPower = TXPower
		msg = fmt.Sprintf("Set TX Power: %v", TXPower)
		d.Print(msg, nil, util.PrintBoth)

		d.Info.Configuration.NbRepUnconfirmedDataUp = NbRep
		msg = fmt.Sprintf("Set Nb Repetition UnconfirmedDataUp: %v", NbRep)
		d.Print(msg, nil, util.PrintBoth)

		d.Info.Configuration.Channels = channels
		msg = fmt.Sprintf("Configuration of channels is changed")
		d.Print(msg, nil, util.PrintBoth)

		msg = PrintMACCommand("LinkADRReq", "Executed successfully")
		d.Print(msg, nil, util.PrintBoth)

	} else {

		msg := PrintMACCommand("LinkADRReq", "Command refused")
		d.Print(msg, nil, util.PrintBoth)

	}

}

func (d *Device) executeDutyCycleReq(payload []byte) {

	c := lorawan.DutyCycleReqPayload{}

	err := c.UnmarshalBinary(payload)
	if err != nil {

		msg := fmt.Sprintf("UnmarshalBinary %v", err)
		errs := errors.New(msg)
		d.Print("", errs, util.PrintBoth)

		return
	}

	//invia i dati all'interfaccia
	aggregatedDC := 1 / math.Pow(2, float64(c.MaxDCycle))

	cont := fmt.Sprintf("Aggregated duty cycle is %v", aggregatedDC)
	msg := PrintMACCommand("DutyCycleReq", cont)
	d.Print(msg, nil, util.PrintBoth)

	//ack
	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID:     lorawan.DutyCycleAns,
			Payload: &lorawan.DevStatusAnsPayload{},
		},
	}

	d.newMACComands(response)

}

//require ack
func (d *Device) executeRXParamSetupReq(payload []byte) {

	c := lorawan.RXParamSetupReqPayload{}
	err := c.UnmarshalBinary(payload)
	if err != nil {
		d.Print("", err, util.PrintBoth)
		return
	}

	//RX[0]
	RX1DROffsetACK := false

	if err = d.Info.Configuration.Region.RX1DROffsetSupported(c.DLSettings.RX1DROffset); err != nil {
		msg := PrintMACCommand("RXParamSetupReq", err.Error())
		d.Print(msg, nil, util.PrintBoth)
	} else {
		RX1DROffsetACK = true
	}

	//RX[1]
	ChannelACK := false
	if err = d.isSupportedFrequency(c.Frequency); err != nil {
		msg := PrintMACCommand("RXParamSetupReq", err.Error())
		d.Print(msg, nil, util.PrintBoth)
	} else {
		ChannelACK = true
	}

	RX2DataRateACK := false
	if err = d.isSupportedDR(c.DLSettings.RX2DataRate); err != nil {
		msg := PrintMACCommand("RXParamSetupReq", err.Error())
		d.Print(msg, nil, util.PrintBoth)
	} else {
		RX2DataRateACK = true
	}

	if RX1DROffsetACK && ChannelACK && RX2DataRateACK {

		d.Info.Configuration.RX1DROffset = c.DLSettings.RX1DROffset //RX1DROffset ACK
		d.Info.RX[1].SetListeningFrequency(c.Frequency)             //Channel Frequency RX2
		d.Info.RX[1].DataRate = c.DLSettings.RX2DataRate            //RX2DataRate

		msg := PrintMACCommand("RXParamSetupReq", "Executed successfully")
		d.Print(msg, nil, util.PrintBoth)

	} else {
		msg := PrintMACCommand("RXParamSetupReq", "Command refused")
		d.Print(msg, nil, util.PrintBoth)
	}

	//ack
	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID: lorawan.RXParamSetupAns,
			Payload: &lorawan.RXParamSetupAnsPayload{
				ChannelACK:     ChannelACK,
				RX2DataRateACK: RX2DataRateACK,
				RX1DROffsetACK: RX1DROffsetACK,
			},
		},
	}

	d.Info.Status.DataUplink.AckMacCommand.SetRXParamSetupAns(response)

}

func (d *Device) executeDevStatusReq() {

	rand.Seed(time.Now().UTC().UnixNano())
	margin := int8(rand.Int()) % MaxMargin //range

	if margin <= 32 {
		margin = margin - 32
	} else {
		margin %= 32
	}

	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID: lorawan.DevStatusAns,
			Payload: &lorawan.DevStatusAnsPayload{
				Battery: d.Info.Status.Battery,
				Margin:  margin,
			},
		},
	}

	msg := PrintMACCommand("DevStatusReq", "Executed successfully")
	d.Print(msg, nil, util.PrintBoth)

	d.newMACComands(response)
}

func (d *Device) executeNewChannelReq(payload []byte) {

	switch d.Info.Configuration.Region.GetCode() {
	case rp.Code_Us915, rp.Code_Au915:

		msg := PrintMACCommand("NewChannelReq", "It's not implemented in this region")
		d.Print(msg, nil, util.PrintBoth)

		return

	}

	c := lorawan.NewChannelReqPayload{}
	err := c.UnmarshalBinary(payload)

	if err != nil {

		d.Print("", err, util.PrintBoth)
		return

	}

	DataRateOK, FreqOK := d.setChannel(c.ChIndex, c.Freq, c.MinDR, c.MaxDR)
	if DataRateOK && FreqOK {

		msg := PrintMACCommand("NewChannelReq", "Executed successfully")
		d.Print(msg, nil, util.PrintBoth)

	} else {

		msg := PrintMACCommand("NewChannelReq", "Command refused")
		d.Print(msg, nil, util.PrintBoth)

	}

	//response
	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID: lorawan.NewChannelAns,
			Payload: &lorawan.NewChannelAnsPayload{
				DataRateRangeOK:    DataRateOK,
				ChannelFrequencyOK: FreqOK,
			},
		},
	}

	d.newMACComands(response)

}

//require ack
func (d *Device) executeRXTimingSetupReq(payload []byte) {

	c := lorawan.RXTimingSetupReqPayload{}

	err := c.UnmarshalBinary(payload)
	if err != nil {

		d.Print("", err, util.PrintBoth)
		return

	}

	delay := c.Delay
	if delay == 0 {
		delay = features.ReceiveDelay
	}

	d.Info.RX[0].Delay = time.Duration(delay) * time.Second

	msg := PrintMACCommand("RXTimingSetupReq", "Executed successfully")
	d.Print(msg, nil, util.PrintBoth)
	//ack
	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID: lorawan.RXTimingSetupAns,
		},
	}

	d.Info.Status.DataUplink.AckMacCommand.SetRXTimingSetupAns(response)
}

//require ack
func (d *Device) executeDLChannelReq(payload []byte) {

	switch d.Info.Configuration.Region.GetCode() {
	case rp.Code_Us915, rp.Code_Au915:

		msg := PrintMACCommand("DLChannelReq", "Is not implemented in this region")
		d.Print(msg, nil, util.PrintBoth)

		return
	}

	c := lorawan.DLChannelReqPayload{}

	err := c.UnmarshalBinary(payload)
	if err != nil {

		msg := fmt.Sprintf("UnmarshalBinary %v", err)
		errs := errors.New(msg)
		d.Print("", errs, util.PrintBoth)

		return
	}

	FreqUpExists, FreqOk := false, false

	err = d.isSupportedFrequency(c.Freq)
	if err == nil {
		FreqUpExists = d.setFrequencyDownlink(c.ChIndex, c.Freq)
		FreqOk = true
	}

	//ack
	if FreqUpExists && FreqOk {

		msg := PrintMACCommand("DLChannelReq", "Executed successfully")
		d.Print(msg, nil, util.PrintBoth)

	} else {

		msg := PrintMACCommand("DLChannelReq", "Command refused")
		d.Print(msg, nil, util.PrintBoth)

	}

	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID: lorawan.DLChannelAns,
			Payload: &lorawan.DLChannelAnsPayload{
				ChannelFrequencyOK:    FreqOk,
				UplinkFrequencyExists: FreqUpExists,
			},
		},
	}

	d.Info.Status.DataUplink.AckMacCommand.SetDLChannelAns(response)

}

func (d *Device) executeDeviceTimeAns(payload []byte) {
	c := lorawan.DeviceTimeAnsPayload{}

	err := c.UnmarshalBinary(payload)
	if err != nil {

		d.Print("", err, util.PrintBoth)
		return

	}

	content := fmt.Sprintf("TimeSinceGPSEpoch[%v]", c.TimeSinceGPSEpoch)

	msg := PrintMACCommand("DeviceTimeAns", content)
	d.Print(msg, nil, util.PrintBoth)

}

func (d *Device) executeTXParamSetupReq(payload []byte) {

	switch d.Info.Configuration.Region.GetCode() {
	case rp.Code_Au915, rp.Code_As923:
	default:
		msg := PrintMACCommand("TXParamSetupReq", "Is not implemented in this region")
		d.Print(msg, nil, util.PrintBoth)
		return
	}

	c := lorawan.TXParamSetupReqPayload{}

	err := c.UnmarshalBinary(payload)
	if err != nil {

		d.Print("", err, util.PrintBoth)
		return

	}

	//c.MaxEIRP
	d.Info.Status.DataUplink.DwellTime = c.UplinkDwellTime
	d.Info.Status.DataDownlink.DwellTime = c.DownlinkDwelltime

	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID: lorawan.TXParamSetupAns,
		},
	}

	msg := PrintMACCommand("TXParamSetupReq", "Executed successfully")
	d.Print(msg, nil, util.PrintBoth)

	d.newMACComands(response)
}

/****************CLASS B MAC COMMAND****************/

func (d *Device) executePingSlotInfoAns(payload []byte) {

	if !d.Info.Configuration.SupportedClassB {
		return
	}

	d.SwitchClass(classes.ClassB)

}

func (d *Device) executeBeaconFreqReq(payload []byte) {

	command := lorawan.BeaconFreqReqPayload{}

	if !d.Info.Configuration.SupportedClassB {
		return
	}

	err := command.UnmarshalBinary(payload)
	if err != nil {
		d.Print("", err, util.PrintBoth)

		return
	}

	freqOk := false
	if command.Frequency == 0 {
		d.Info.Status.InfoClassB.FrequencyBeacon = d.Info.Configuration.Region.GetFrequencyBeacon()
	} else {
		err := d.isSupportedFrequency(command.Frequency)
		if err == nil {
			freqOk = true
			d.Info.Status.InfoClassB.FrequencyBeacon = command.Frequency
		}
	}

	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID: lorawan.BeaconFreqAns,
			Payload: &lorawan.BeaconFreqAnsPayload{
				BeaconFrequencyOK: freqOk,
			},
		},
	}

	d.newMACComands(response)
}

func (d *Device) executePingSlotChannelReq(payload []byte) {

	if !d.Info.Configuration.SupportedClassB {
		return
	}

	c := lorawan.PingSlotChannelReqPayload{}

	err := c.UnmarshalBinary(payload)
	if err != nil {

		d.Print("", err, util.PrintBoth)
		return

	}

	FreqOK, DataRateOK := false, false
	err = d.isSupportedFrequency(c.Frequency)
	if err == nil {
		FreqOK = true
	}

	err = d.isSupportedDR(c.DR)
	if err == nil {
		DataRateOK = true
	}

	if FreqOK && DataRateOK {
		d.Info.Status.InfoClassB.PingSlot.SetListeningFrequency(c.Frequency) //set frequency listen
		d.Info.Status.InfoClassB.PingSlot.DataRate = c.DR                    // set datarate
	}

	//response
	response := []lorawan.Payload{
		&lorawan.MACCommand{
			CID: lorawan.PingSlotChannelAns,
			Payload: &lorawan.PingSlotChannelAnsPayload{
				DataRateOK:         DataRateOK,
				ChannelFrequencyOK: FreqOK,
			},
		},
	}

	d.newMACComands(response)

}

func PrintMACCommand(cmd string, content string) string {
	return fmt.Sprintf("%v | %v |", cmd, content)
}

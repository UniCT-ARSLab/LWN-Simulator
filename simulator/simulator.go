package simulator

import (
	"errors"
	"fmt"
	"log"
	"time"

	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	f "github.com/arslab/lwnsimulator/simulator/components/forwarder"
	mfw "github.com/arslab/lwnsimulator/simulator/components/forwarder/models"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	res "github.com/arslab/lwnsimulator/simulator/resources"
	"github.com/arslab/lwnsimulator/simulator/util"
	e "github.com/arslab/lwnsimulator/socket"
	"github.com/arslab/lwnsimulator/types"
	"github.com/brocaar/lorawan"
)

//Simulator is a model
type Simulator struct {
	State           uint8         `json:"-"`
	Devices         []*dev.Device `json:"-"`
	DevicesInactive int           `json:"-"`
	Gateways        []*gw.Gateway `json:"-"`
	Forwarder       f.Forwarder   `json:"-"`
	BridgeAddress   string        `json:"BridgeAddress"`
	Resources       res.Resources `json:"-"`
}

func (s *Simulator) setupSimulator() {

	s.Print("SETUP", nil, util.PrintBoth)

	s.setupGateways()
	s.setupDevices()

	var infoD []mfw.InfoDevice
	var infoGW []mfw.InfoGateway

	for i := 0; i < len(s.Devices); i++ {
		tmp := mfw.InfoDevice{
			DevEUI:   s.Devices[i].Info.DevEUI,
			Location: s.Devices[i].Info.Location,
			Range:    s.Devices[i].Info.Configuration.Range,
		}
		infoD = append(infoD, tmp)
	}

	for i := 0; i < len(s.Gateways); i++ {
		tmp := mfw.InfoGateway{
			Buf:      &s.Gateways[i].BufferUplink,
			Location: s.Gateways[i].Info.Location,
		}
		infoGW = append(infoGW, tmp)
	}

	s.Forwarder = *f.Setup(infoD, infoGW)

}

func (s *Simulator) setupGateways() {

	var gws []gw.Gateway

	path, err := util.GetPath()
	if err != nil {
		s.Print("", err, util.PrintOnlyConsole)
		return
	}

	err = util.RecoverConfigFile(path+"/gateways.json", &gws)
	if err != nil {

		s.Print("", err, util.PrintOnlyConsole)
		return

	}

	for i := 0; i < len(gws); i++ {

		if gws[i].Info.Active {
			s.Gateways = append(s.Gateways, &gws[i])
		}

	}

	for i := 0; i < len(s.Gateways); i++ {
		s.Gateways[i].Setup(&s.BridgeAddress, &s.Resources, &s.State, &s.Forwarder)
	}

}

func (s *Simulator) setupDevices() {

	var devices []dev.Device

	path, err := util.GetPath()
	if err != nil {
		s.Print("", err, util.PrintOnlyConsole)
		return
	}

	err = util.RecoverConfigFile(path+"/devices.json", &devices)
	if err != nil {

		s.Print("", err, util.PrintOnlyConsole)
		return

	}

	//load only active devices
	for i := 0; i < len(devices); i++ {
		if devices[i].Info.Status.Active {
			s.Devices = append(s.Devices, &devices[i])
		}
	}

	for i := 0; i < len(s.Devices); i++ {
		s.Devices[i].Setup(&s.Resources, &s.State, &s.Forwarder)
	}

}

func searchName(Name string, index *int) (int, error) {

	gateways := GetGateways()
	devices := GetDevices()

	for i, g := range gateways {

		if g.Info.Name == Name {

			if index != nil { //update

				if *index != i { //different gw
					return types.CodeErrorName, errors.New("Error: Name already used")
				}

			} else {
				return types.CodeErrorName, errors.New("Error: Name already used")
			}

		}

	}

	for i, d := range devices {

		if d.Info.Name == Name {

			if index != nil { //update

				if *index != i { //different dev
					return types.CodeErrorName, errors.New("Error: Name already used")
				}

			} else {
				return types.CodeErrorName, errors.New("Error: Name already used")
			}

		}

	}

	return types.CodeOK, nil
}

func searchAddress(address lorawan.EUI64, index *int) (int, error) {

	gateways := GetGateways()
	devices := GetDevices()

	for i, g := range gateways {

		if g.Info.MACAddress == address {

			if index != nil { //update

				if *index != i { //different gw
					return types.CodeErrorAddress, errors.New("Error: DevEUI already used")
				}

			} else {
				return types.CodeErrorAddress, errors.New("Error: MAC Address already used")
			}

		}

	}

	for i, d := range devices {

		if d.Info.DevEUI == address {

			if index != nil { //update

				if *index != i { //different dev
					return types.CodeErrorAddress, errors.New("Error: DevEUI already used")
				}

			} else {
				return types.CodeErrorAddress, errors.New("Error: DevEUI already used")
			}

		}

	}

	return types.CodeOK, nil
}

func (s *Simulator) Print(content string, err error, printType int) {

	now := time.Now()
	message := ""
	messageLog := ""
	event := e.EventLog

	if err == nil {
		message = fmt.Sprintf("[ %s ] [SIM] : %s", now.Format(time.Stamp), content)
		messageLog = fmt.Sprintf("[SIM] : %s", content)
	} else {
		message = fmt.Sprintf("[ %s ] [SIM] [ERROR]: %s", now.Format(time.Stamp), err)
		messageLog = fmt.Sprintf("[SIM] [ERROR]: %s", err)
		event = e.EventError
	}

	data := e.ConsoleLog{
		Name: "SIM",
		Msg:  message,
	}

	switch printType {

	case util.PrintBoth:
		s.Resources.WebSocket.Emit(event, data)
		log.Println(messageLog)

	case util.PrintOnlySocket:
		s.Resources.WebSocket.Emit(event, data)

	case util.PrintOnlyConsole:
		log.Println(messageLog)

	}

}

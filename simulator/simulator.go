package simulator

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/arslab/lwnsimulator/codes"
	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	f "github.com/arslab/lwnsimulator/simulator/components/forwarder"
	mfw "github.com/arslab/lwnsimulator/simulator/components/forwarder/models"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	c "github.com/arslab/lwnsimulator/simulator/console"
	res "github.com/arslab/lwnsimulator/simulator/resources"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/arslab/lwnsimulator/socket"
	"github.com/brocaar/lorawan"
)

// Simulator is a model
type Simulator struct {
	State                 uint8               `json:"-"`
	Devices               map[int]*dev.Device `json:"-"`
	ActiveDevices         map[int]int         `json:"-"`
	ActiveGateways        map[int]int         `json:"-"`
	ComponentsInactiveTmp int                 `json:"-"`
	Gateways              map[int]*gw.Gateway `json:"-"`
	Forwarder             f.Forwarder         `json:"-"`
	NextIDDev             int                 `json:"nextIDDev"`
	NextIDGw              int                 `json:"nextIDGw"`
	BridgeAddress         string              `json:"bridgeAddress"`
	Resources             res.Resources       `json:"-"`
	Console               c.Console           `json:"-"`
}

func (s *Simulator) setup() {
	s.setupGateways()
	s.setupDevices()
	s.SetupConsole()

	s.Print("SETUP OK!", nil, util.PrintBoth)
}

func (s *Simulator) setupGateways() {

	for _, g := range s.Gateways {

		s.Gateways[g.Id].State = util.Stopped

		if g.Info.Active {
			s.ActiveGateways[g.Id] = g.Id
		}

	}
	s.Print("Setup gateways OK!", nil, util.PrintOnlySocket)
}

func (s *Simulator) setupDevices() {

	for _, d := range s.Devices {

		s.Devices[d.Id].State = util.Stopped

		if d.Info.Status.Active {
			s.ActiveDevices[d.Id] = d.Id
		}

	}
	s.Print("Setup devices OK!", nil, util.PrintOnlySocket)
}

func (s *Simulator) SetupConsole() {
	for _, d := range s.Devices {
		s.Devices[d.Id].SetConsole(&s.Console)
	}
	for _, g := range s.Gateways {
		s.Gateways[g.Id].SetConsole(&s.Console)
	}
}

func (s *Simulator) loadData() {

	path, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	err = util.RecoverConfigFile(path+"/simulator.json", &s)
	if err != nil {
		log.Fatal(err)
	}

	err = util.RecoverConfigFile(path+"/gateways.json", &s.Gateways)
	if err != nil {
		log.Fatal(err)
	}

	err = util.RecoverConfigFile(path+"/devices.json", &s.Devices)
	if err != nil {
		log.Fatal(err)
	}

}

func (s *Simulator) searchName(Name string, Id int, gwFlag bool) (int, error) {

	for _, g := range s.Gateways {

		if g.Info.Name == Name {

			if (gwFlag && g.Id != Id) || !gwFlag {
				return codes.CodeErrorName, errors.New("Error: Name already used")
			}

		}

	}

	for _, d := range s.Devices {

		if d.Info.Name == Name {
			if (!gwFlag && d.Id != Id) || gwFlag {
				return codes.CodeErrorName, errors.New("Error: Name already used")
			}

		}

	}

	return codes.CodeOK, nil
}

func (s *Simulator) searchAddress(address lorawan.EUI64, Id int, gwFlag bool) (int, error) {

	for _, g := range s.Gateways {

		if g.Info.MACAddress == address {

			if (gwFlag && g.Id != Id) || !gwFlag {
				return codes.CodeErrorAddress, errors.New("Error: MAC Address already used")
			}

		}

	}

	for _, d := range s.Devices {

		if d.Info.DevEUI == address {

			if (!gwFlag && d.Id != Id) || gwFlag {
				return codes.CodeErrorAddress, errors.New("Error: DevEUI already used")
			}

		}

	}

	return codes.CodeOK, nil
}

func (s *Simulator) saveComponent(path string, v interface{}) {

	bytes, err := json.MarshalIndent(&v, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = util.WriteConfigFile(path, bytes)
	if err != nil {
		log.Fatal(err)
	}

}

func (s *Simulator) saveStatus() {

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/simulator.json"
	s.saveComponent(path, &s)

	path = pathDir + "/devices.json"
	s.saveComponent(path, &s.Devices)

	path = pathDir + "/gateways.json"
	s.saveComponent(path, &s.Gateways)

	s.Print("Status saved", nil, util.PrintOnlyConsole)
}

func (s *Simulator) turnONDevice(Id int) {

	infoDev := mfw.InfoDevice{
		DevEUI:   s.Devices[Id].Info.DevEUI,
		Location: s.Devices[Id].Info.Location,
		Range:    s.Devices[Id].Info.Configuration.Range,
	}
	s.Forwarder.AddDevice(infoDev)

	s.Devices[Id].Setup(&s.Resources, &s.Forwarder)
	s.Devices[Id].TurnON()
	s.ActiveDevices[Id] = Id

	s.Console.PrintSocket(socket.EventResponseCommand, s.Devices[Id].Info.Name+" Turn ON")
}

func (s *Simulator) turnOFFDevice(Id int) {

	s.ComponentsInactiveTmp++
	s.Resources.ExitGroup.Add(1)

	s.Devices[Id].TurnOFF()

	s.Forwarder.DeleteDevice(s.Devices[Id].Info.DevEUI)

	s.Resources.ExitGroup.Wait()

	delete(s.ActiveDevices, Id)
	s.ComponentsInactiveTmp--

	status := socket.NewStatusDev{
		DevEUI:   s.Devices[Id].Info.DevEUI,
		DevAddr:  s.Devices[Id].Info.DevAddr,
		NwkSKey:  string(s.Devices[Id].Info.NwkSKey[:]),
		AppSKey:  string(s.Devices[Id].Info.AppSKey[:]),
		FCntDown: s.Devices[Id].Info.Status.FCntDown,
		FCnt:     s.Devices[Id].Info.Status.DataUplink.FCnt,
	}

	s.Console.PrintSocket(socket.EventSaveStatus, status)

	s.Console.PrintSocket(socket.EventResponseCommand, s.Devices[Id].Info.Name+" Turn OFF")
}

func (s *Simulator) turnONGateway(Id int) {
	infoGw := mfw.InfoGateway{
		MACAddress: s.Gateways[Id].Info.MACAddress,
		Buffer:     &s.Gateways[Id].BufferUplink,
		Location:   s.Gateways[Id].Info.Location,
	}

	s.Forwarder.AddGateway(infoGw)

	s.Gateways[Id].Setup(&s.BridgeAddress, &s.Resources, &s.Forwarder)
	s.Gateways[Id].TurnON()

	s.ActiveGateways[Id] = Id

	s.Console.PrintSocket(socket.EventResponseCommand, s.Gateways[Id].Info.Name+" Turn ON")
}

func (s *Simulator) turnOFFGateway(Id int) {

	s.ComponentsInactiveTmp++
	s.Resources.ExitGroup.Add(1)

	s.Gateways[Id].TurnOFF()

	s.Resources.ExitGroup.Wait()

	delete(s.ActiveGateways, Id)
	s.ComponentsInactiveTmp--

	infoGw := mfw.InfoGateway{
		Buffer:   &s.Gateways[Id].BufferUplink,
		Location: s.Gateways[Id].Info.Location,
	}

	s.Forwarder.DeleteGateway(infoGw)

	s.Console.PrintSocket(socket.EventResponseCommand, s.Gateways[Id].Info.Name+" Turn OFF")
}

func (s *Simulator) reset() {

	for key := range s.ActiveGateways {
		delete(s.ActiveGateways, key)
	}

	for key := range s.ActiveDevices {
		delete(s.ActiveDevices, key)
	}

	s.ActiveDevices = make(map[int]int)
	s.ActiveGateways = make(map[int]int)

	s.Print("Reset", nil, util.PrintOnlyConsole)

}

func (s *Simulator) Print(content string, err error, printType int) {

	now := time.Now()
	message := ""
	messageLog := ""
	event := socket.EventLog

	if err == nil {
		message = fmt.Sprintf("[ %s ] [SIM]: %s", now.Format(time.Stamp), content)
		messageLog = fmt.Sprintf("[SIM]: %s", content)
	} else {
		message = fmt.Sprintf("[ %s ] [SIM] [ERROR]: %s", now.Format(time.Stamp), err)
		messageLog = fmt.Sprintf("[SIM] [ERROR]: %s", err)
		event = socket.EventError
	}

	data := socket.ConsoleLog{
		Name: "SIM",
		Msg:  message,
	}

	switch printType {
	case util.PrintBoth:
		s.Console.PrintSocket(event, data)
		s.Console.PrintLog(messageLog)
	case util.PrintOnlySocket:
		s.Console.PrintSocket(event, data)
	case util.PrintOnlyConsole:
		s.Console.PrintLog(messageLog)
	}
}

package simulator

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/arslab/lwnsimulator/shared"
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
	State                 uint8               `json:"-"`             // Runtime state: Stop, Running
	Devices               map[int]*dev.Device `json:"-"`             // A collection of devices
	ActiveDevices         map[int]int         `json:"-"`             // A collection of active devices
	ActiveGateways        map[int]int         `json:"-"`             // A collection of active gateways
	ComponentsInactiveTmp int                 `json:"-"`             // Number of inactive components
	Gateways              map[int]*gw.Gateway `json:"-"`             // A collection of gateways
	Forwarder             f.Forwarder         `json:"-"`             // Forwarder instance used for communication between devices and gateways
	NextIDDev             int                 `json:"nextIDDev"`     // Next device ID used for creating a new device
	NextIDGw              int                 `json:"nextIDGw"`      // Next gateway ID used for creating a new gateway
	BridgeAddress         string              `json:"bridgeAddress"` // Bridge address used to connect to a network
	Resources             res.Resources       `json:"-"`             // Resources used for managing the simulator
	Console               c.Console           `json:"-"`             // Console instance, used for logging in the web terminal
}

// setup loads and initializes the simulator maps for gateways and devices. It also initializes the console
func (s *Simulator) setup() {
	s.setupGateways()
	s.setupDevices()
	s.SetupConsole()
	s.Print("SETUP OK!", nil, util.PrintBoth)
}

// setupGateways initializes the gateways by setting their state to Stopped and adding them to the ActiveGateways map if they are active
func (s *Simulator) setupGateways() {
	for _, g := range s.Gateways {
		s.Gateways[g.Id].State = util.Stopped
		if g.Info.Active {
			s.ActiveGateways[g.Id] = g.Id
		}
	}
	s.Print("Setup gateways OK!", nil, util.PrintOnlySocket)
}

// setupDevices initializes the devices by setting their state to Stopped and adding them to the ActiveDevices map if they are active
func (s *Simulator) setupDevices() {
	for _, d := range s.Devices {
		s.Devices[d.Id].State = util.Stopped
		if d.Info.Status.Active {
			s.ActiveDevices[d.Id] = d.Id
		}
	}
	s.Print("Setup devices OK!", nil, util.PrintOnlySocket)
}

// SetupConsole attach the simulator console to devices and gateways
func (s *Simulator) SetupConsole() {
	for _, d := range s.Devices {
		s.Devices[d.Id].SetConsole(&s.Console)
	}
	for _, g := range s.Gateways {
		s.Gateways[g.Id].SetConsole(&s.Console)
	}
}

// loadData retrieves the simulator configuration, devices, and gateways from the JSON files by populating the Simulator struct
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

// saveComponent saves a configuration of the provided interface to a JSON file
func (s *Simulator) saveComponent(path string, v interface{}) {
	shared.DebugPrint(fmt.Sprintf("Saving component %s on disk", path))
	bytes, err := json.MarshalIndent(&v, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = util.WriteConfigFile(path, bytes)
	if err != nil {
		log.Fatal(err)
	}

}

// saveStatus saves the simulator status, devices, and gateways to JSON files
func (s *Simulator) saveStatus() {
	shared.DebugPrint("Saving status on disk")
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

// turnONDevice activates a device by adding it to the Forwarder and turning it on
func (s *Simulator) turnONDevice(Id int) {
	infoDev := mfw.InfoDevice{
		DevEUI:   s.Devices[Id].Info.DevEUI,
		Location: s.Devices[Id].Info.Location,
		Range:    s.Devices[Id].Info.Configuration.Range,
	}
	s.Forwarder.AddDevice(infoDev)
	s.Devices[Id].Setup(&s.Resources, &s.Forwarder)
	s.Devices[Id].TurnON()
	s.Console.PrintSocket(socket.EventResponseCommand, s.Devices[Id].Info.Name+" Turn ON")
}

// turnOFFDevice deactivates a device by removing it from the Forwarder and turning it off
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

// turnONGateway activates a gateway by adding it to the Forwarder and turning it on
func (s *Simulator) turnONGateway(Id int) {
	infoGw := mfw.InfoGateway{
		MACAddress: s.Gateways[Id].Info.MACAddress,
		Buffer:     &s.Gateways[Id].BufferUplink,
		Location:   s.Gateways[Id].Info.Location,
	}
	s.Forwarder.AddGateway(infoGw)
	s.Gateways[Id].Setup(&s.BridgeAddress, &s.Resources, &s.Forwarder)
	s.Gateways[Id].TurnON()
	s.Console.PrintSocket(socket.EventResponseCommand, s.Gateways[Id].Info.Name+" Turn ON")
}

// turnOFFGateway deactivates a gateway by removing it from the Forwarder and turning it off
func (s *Simulator) turnOFFGateway(Id int) {
	s.ComponentsInactiveTmp++
	s.Resources.ExitGroup.Add(1)
	s.Gateways[Id].TurnOFF()
	s.Resources.ExitGroup.Wait()
	delete(s.ActiveGateways, Id)
	s.ComponentsInactiveTmp--
	infoGw := mfw.InfoGateway{
		MACAddress: s.Gateways[Id].Info.MACAddress,
		Buffer:     &s.Gateways[Id].BufferUplink,
		Location:   s.Gateways[Id].Info.Location,
	}
	s.Forwarder.DeleteGateway(infoGw)
	s.Console.PrintSocket(socket.EventResponseCommand, s.Gateways[Id].Info.Name+" Turn OFF")
}

// reset removes all devices and gateways from the ActiveDevices and ActiveGateways maps
func (s *Simulator) reset() {
	shared.DebugPrint("Resetting simulator")
	clear(s.ActiveDevices)
	clear(s.ActiveGateways)
	s.Print("Reset", nil, util.PrintOnlyConsole)
}

// Print logs messages to the console and the web terminal based on the printType
func (s *Simulator) Print(content string, err error, printType int) {
	// Get current time as a timestamp
	now := time.Now().Format(time.Stamp)
	message := ""
	messageLog := ""
	event := socket.EventLog
	if err == nil {
		message = fmt.Sprintf("[ %s ] [SIM]: %s", now, content)
		messageLog = fmt.Sprintf("[SIM]: %s", content)
	} else {
		message = fmt.Sprintf("[ %s ] [SIM] [ERROR]: %s", now, err)
		messageLog = fmt.Sprintf("[SIM] [ERROR]: %s", err)
		event = socket.EventError
	}
	// Create a new ConsoleLog struct
	data := socket.ConsoleLog{
		Name: "SIM",
		Msg:  message,
	}
	switch printType {
	case util.PrintOnlySocket:
		s.Console.PrintSocket(event, data)
	case util.PrintOnlyConsole:
		s.Console.PrintLog(messageLog)
	default: // util.PrintBoth
		s.Console.PrintSocket(event, data)
		s.Console.PrintLog(messageLog)
	}
}

package simulator

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/brocaar/lorawan"

	"github.com/arslab/lwnsimulator/codes"
	"github.com/arslab/lwnsimulator/models"

	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	f "github.com/arslab/lwnsimulator/simulator/components/forwarder"
	mfw "github.com/arslab/lwnsimulator/simulator/components/forwarder/models"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	c "github.com/arslab/lwnsimulator/simulator/console"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/arslab/lwnsimulator/socket"
	socketio "github.com/googollee/go-socket.io"
)

func GetIstance() *Simulator {

	var s Simulator

	s.State = util.Stopped

	s.loadData()

	s.ActiveDevices = make(map[int]int)
	s.ActiveGateways = make(map[int]int)

	s.Forwarder = *f.Setup()

	s.Console = c.Console{}

	return &s
}

func (s *Simulator) AddWebSocket(WebSocket *socketio.Conn) {
	s.Console.SetupWebSocket(WebSocket)
	s.Resources.AddWebSocket(WebSocket)
	s.SetupConsole()
}

func (s *Simulator) Run() {

	s.State = util.Running
	s.setup()

	s.Print("START", nil, util.PrintBoth)

	for _, id := range s.ActiveGateways {
		s.turnONGateway(id)
	}

	for _, id := range s.ActiveDevices {
		s.turnONDevice(id)
	}
}

func (s *Simulator) Stop() {

	s.State = util.Stopped
	s.Resources.ExitGroup.Add(len(s.ActiveGateways) + len(s.ActiveDevices) - s.ComponentsInactiveTmp)

	for _, id := range s.ActiveGateways {
		s.Gateways[id].TurnOFF()
	}

	for _, id := range s.ActiveDevices {
		s.Devices[id].TurnOFF()
	}

	s.Resources.ExitGroup.Wait()

	s.saveStatus()

	s.Forwarder.Reset()

	s.Print("STOPPED", nil, util.PrintBoth)

	s.reset()

}

func (s *Simulator) SaveBridgeAddress(remoteAddr models.AddressIP) error {

	s.BridgeAddress = fmt.Sprintf("%v:%v", remoteAddr.Address, remoteAddr.Port)

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/simulator.json"

	bytes, err := json.MarshalIndent(&s, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = util.WriteConfigFile(path, bytes)
	if err != nil {
		log.Fatal(err)
	}

	s.Print("Gateway Bridge Address saved", nil, util.PrintOnlyConsole)

	return nil
}

func (s *Simulator) GetBridgeAddress() models.AddressIP {

	var rServer models.AddressIP
	if s.BridgeAddress == "" {
		return rServer
	}

	parts := strings.Split(s.BridgeAddress, ":")

	rServer.Address = parts[0]
	rServer.Port = parts[1]

	return rServer
}

func (s *Simulator) GetGateways() []gw.Gateway {

	var gateways []gw.Gateway

	for _, g := range s.Gateways {
		gateways = append(gateways, *g)
	}

	return gateways

}

func (s *Simulator) GetDevices() []dev.Device {

	var devices []dev.Device

	for _, d := range s.Devices {
		devices = append(devices, *d)
	}

	return devices

}

func (s *Simulator) SetGateway(gateway *gw.Gateway, update bool) (int, int, error) {

	emptyAddr := lorawan.EUI64{0, 0, 0, 0, 0, 0, 0, 0}

	if gateway.Info.MACAddress == emptyAddr {

		s.Print("Error: MAC Address invalid", nil, util.PrintOnlyConsole)
		return codes.CodeErrorAddress, -1, errors.New("Error: MAC Address invalid")

	}

	if !update { //new

		gateway.Id = s.NextIDGw
		s.NextIDGw++

	} else {

		if s.Gateways[gateway.Id].IsOn() {
			return codes.CodeErrorDeviceActive, -1, errors.New("Gateway is running, unable update")
		}

	}

	code, err := s.searchName(gateway.Info.Name, gateway.Id, true)
	if err != nil {

		s.Print("Name already used", nil, util.PrintOnlyConsole)
		return code, -1, err

	}

	code, err = s.searchAddress(gateway.Info.MACAddress, gateway.Id, true)
	if err != nil {

		s.Print("DevEUI already used", nil, util.PrintOnlyConsole)
		return code, -1, err

	}

	if !gateway.Info.TypeGateway {

		if s.BridgeAddress == "" {
			return codes.CodeNoBridge, -1, errors.New("No gateway bridge configured")
		}

	}

	s.Gateways[gateway.Id] = gateway

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/gateways.json"
	s.saveComponent(path, &s.Gateways)
	path = pathDir + "/simulator.json"
	s.saveComponent(path, &s)

	s.Print("Gateway Saved", nil, util.PrintOnlyConsole)

	if gateway.Info.Active {

		s.ActiveGateways[gateway.Id] = gateway.Id

		if s.State == util.Running {
			s.Gateways[gateway.Id].Setup(&s.BridgeAddress, &s.Resources, &s.Forwarder)
			s.turnONGateway(gateway.Id)
		}

	} else {
		_, ok := s.ActiveGateways[gateway.Id]
		if ok {
			delete(s.ActiveGateways, gateway.Id)
		}
	}

	return codes.CodeOK, gateway.Id, nil
}

func (s *Simulator) DeleteGateway(Id int) bool {

	if s.Gateways[Id].IsOn() {
		return false
	}

	delete(s.Gateways, Id)
	delete(s.ActiveGateways, Id)

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/gateways.json"
	s.saveComponent(path, &s.Gateways)

	s.Print("Gateway Deleted", nil, util.PrintOnlyConsole)

	return true
}

func (s *Simulator) SetDevice(device *dev.Device, update bool) (int, int, error) {

	emptyAddr := lorawan.EUI64{0, 0, 0, 0, 0, 0, 0, 0}

	if device.Info.DevEUI == emptyAddr {

		s.Print("DevEUI invalid", nil, util.PrintOnlyConsole)
		return codes.CodeErrorAddress, -1, errors.New("Error: DevEUI invalid")

	}

	if !update { //new

		device.Id = s.NextIDDev
		s.NextIDDev++

	} else {

		if s.Devices[device.Id].IsOn() {
			return codes.CodeErrorDeviceActive, -1, errors.New("Device is running, unable update")
		}

	}

	code, err := s.searchName(device.Info.Name, device.Id, false)
	if err != nil {

		s.Print("Name already used", nil, util.PrintOnlyConsole)
		return code, -1, err

	}

	code, err = s.searchAddress(device.Info.DevEUI, device.Id, false)
	if err != nil {

		s.Print("DevEUI already used", nil, util.PrintOnlyConsole)
		return code, -1, err

	}

	s.Devices[device.Id] = device

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/devices.json"
	s.saveComponent(path, &s.Devices)
	path = pathDir + "/simulator.json"
	s.saveComponent(path, &s)

	s.Print("Device Saved", nil, util.PrintOnlyConsole)

	if device.Info.Status.Active {

		s.ActiveDevices[device.Id] = device.Id

		if s.State == util.Running {
			s.turnONDevice(device.Id)
		}

	} else {
		_, ok := s.ActiveDevices[device.Id]
		if ok {
			delete(s.ActiveDevices, device.Id)
		}
	}

	return codes.CodeOK, device.Id, nil
}

func (s *Simulator) DeleteDevice(Id int) bool {

	if s.Devices[Id].IsOn() {
		return false
	}

	delete(s.Devices, Id)
	delete(s.ActiveDevices, Id)

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/devices.json"
	s.saveComponent(path, &s.Devices)

	s.Print("Device Deleted", nil, util.PrintOnlyConsole)

	return true
}

func (s *Simulator) ToggleStateDevice(Id int) {

	if s.Devices[Id].State == util.Stopped {
		s.turnONDevice(Id)
	} else if s.Devices[Id].State == util.Running {
		s.turnOFFDevice(Id)
	}

}

func (s *Simulator) SendMACCommand(cid lorawan.CID, data socket.MacCommand) {

	if !s.Devices[data.Id].IsOn() {
		s.Console.PrintSocket(socket.EventResponseCommand, s.Devices[data.Id].Info.Name+" is turned off")
		return
	}

	err := s.Devices[data.Id].SendMACCommand(cid, data.Periodicity)
	if err != nil {
		s.Console.PrintSocket(socket.EventResponseCommand, "Unable to send command: "+err.Error())
	} else {
		s.Console.PrintSocket(socket.EventResponseCommand, "MACCommand will be sent to the next uplink")
	}

}

func (s *Simulator) ChangePayload(pl socket.NewPayload) (string, bool) {

	devEUIstring := hex.EncodeToString(s.Devices[pl.Id].Info.DevEUI[:])

	if !s.Devices[pl.Id].IsOn() {
		s.Console.PrintSocket(socket.EventResponseCommand, s.Devices[pl.Id].Info.Name+" is turned off")
		return devEUIstring, false
	}

	MType := lorawan.UnconfirmedDataUp
	if pl.MType == "ConfirmedDataUp" {
		MType = lorawan.ConfirmedDataUp
	}

	Payload := &lorawan.DataPayload{
		Bytes: []byte(pl.Payload),
	}

	s.Devices[pl.Id].ChangePayload(MType, Payload)

	s.Console.PrintSocket(socket.EventResponseCommand, s.Devices[pl.Id].Info.Name+": Payload changed")

	return devEUIstring, true
}

func (s *Simulator) SendUplink(pl socket.NewPayload) {

	if !s.Devices[pl.Id].IsOn() {
		s.Console.PrintSocket(socket.EventResponseCommand, s.Devices[pl.Id].Info.Name+" is turned off")
		return
	}

	MType := lorawan.UnconfirmedDataUp
	if pl.MType == "ConfirmedDataUp" {
		MType = lorawan.ConfirmedDataUp
	}

	s.Devices[pl.Id].NewUplink(MType, pl.Payload)

	s.Console.PrintSocket(socket.EventResponseCommand, "Uplink queued")
}

func (s *Simulator) ChangeLocation(l socket.NewLocation) bool {

	if !s.Devices[l.Id].IsOn() {
		return false
	}

	s.Devices[l.Id].ChangeLocation(l.Latitude, l.Longitude, l.Altitude)

	info := mfw.InfoDevice{
		DevEUI:   s.Devices[l.Id].Info.DevEUI,
		Location: s.Devices[l.Id].Info.Location,
		Range:    s.Devices[l.Id].Info.Configuration.Range,
	}

	s.Forwarder.UpdateDevice(info)

	return true
}

func (s *Simulator) ToggleStateGateway(Id int) {

	if s.Gateways[Id].State == util.Stopped {
		s.turnONGateway(Id)
	} else {
		s.turnOFFGateway(Id)
	}

}

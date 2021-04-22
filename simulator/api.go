package simulator

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/brocaar/lorawan"

	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	mfw "github.com/arslab/lwnsimulator/simulator/components/forwarder/models"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	res "github.com/arslab/lwnsimulator/simulator/resources"
	"github.com/arslab/lwnsimulator/socket"
	"github.com/arslab/lwnsimulator/types"

	"github.com/arslab/lwnsimulator/simulator/util"
	socketio "github.com/googollee/go-socket.io"
)

func Setup(WebSocket socketio.Conn) *Simulator {

	var s Simulator

	path, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	err = util.RecoverConfigFile(path+"/simulator.json", &s)
	if err != nil {
		log.Fatal(err)
	}

	s.State = util.Stopped
	s.Resources = *res.NewResource(WebSocket)

	return &s
}

func (s *Simulator) Run() {

	s.setupSimulator()

	s.State = util.Running

	s.Print("START", nil, util.PrintBoth)

	for i := 0; i < len(s.Gateways); i++ {
		s.Gateways[i].OnStart()
	}

	for i := 0; i < len(s.Devices); i++ {
		s.Devices[i].OnStart()
	}

}

func (s *Simulator) resetResources() {

	s.Devices = s.Devices[:0]
	s.Gateways = s.Gateways[:0]
	s.Forwarder.Reset()

	s.Print("Reset Resources", nil, util.PrintOnlyConsole)

}

func (s *Simulator) Stop() {

	s.State = util.Stopped

	s.Print("STOP", nil, util.PrintOnlySocket)

	for i := 0; i < len(s.Gateways); i++ {
		s.Gateways[i].OnStop()
	}

	s.Resources.ExitGroup.Add(len(s.Gateways) + len(s.Devices) - s.DevicesInactive)
	s.Resources.ExitGroup.Wait()

	s.resetResources()

	s.Print("STOPPED", nil, util.PrintBoth)

}

func (s *Simulator) SaveBridgeAddress(remoteAddr types.AddressIP) error {

	//remoteServer cambia anche in running
	s.BridgeAddress = fmt.Sprintf("%v:%v", remoteAddr.Address, remoteAddr.Port)

	simBytes, err := json.MarshalIndent(&s, "", "\t")
	if err != nil {

		s.Print("", err, util.PrintOnlyConsole)
		return errors.New("Error while saving data")

	}

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/simulator.json"

	err = util.WriteConfigFile(path, simBytes)
	if err != nil {

		s.Print("", err, util.PrintOnlyConsole)
		return errors.New("Error while saving data")

	}

	s.Print("Gateway Bridge Address saved", nil, util.PrintOnlyConsole)

	return nil
}

func (s *Simulator) GetBridgeAddress() types.AddressIP {

	//non c'è bisogno di leggere il file json in quanto
	//l'address è già caricato nel simulatore

	var rServer types.AddressIP
	if s.BridgeAddress == "" {
		return rServer
	}

	parts := strings.Split(s.BridgeAddress, ":")

	rServer.Address = parts[0]
	rServer.Port = parts[1]

	return rServer
}

func GetGateways() []gw.Gateway {

	var gateways []gw.Gateway

	path, err := util.GetPath()
	if err != nil {
		log.Println(err)
		return gateways
	}

	err = util.RecoverConfigFile(path+"/gateways.json", &gateways)
	if err != nil {

		log.Println(err)
		return gateways

	}

	return gateways
}

func GetDevices() []dev.Device {

	var devices []dev.Device

	path, err := util.GetPath()
	if err != nil {
		log.Println(err)
		return devices
	}

	err = util.RecoverConfigFile(path+"/devices.json", &devices)
	if err != nil {

		log.Println(err)
		return devices

	}

	return devices
}

func (s *Simulator) SetGateway(gateway *gw.Gateway, index *int) (int, error) {

	emptyAddr := lorawan.EUI64{0, 0, 0, 0, 0, 0, 0, 0}

	if gateway.Info.MACAddress == emptyAddr {

		s.Print("Error: MAC Address invalid", nil, util.PrintOnlyConsole)
		return types.CodeErrorAddress, errors.New("Error: MAC Address invalid")

	}

	//gateway in running?
	for _, g := range s.Gateways {

		if g.Info.MACAddress == gateway.Info.MACAddress {
			return types.CodeErrorGatewayActive, errors.New("Gateway is running, unable update")
		}

	}

	gateways := GetGateways()

	code, err := searchName(gateway.Info.Name, index)
	if err != nil {

		s.Print("Name already used", nil, util.PrintOnlyConsole)
		return code, err

	}

	code, err = searchAddress(gateway.Info.MACAddress, index)
	if err != nil {

		s.Print("DevEUI already used", nil, util.PrintOnlyConsole)
		return code, err

	}

	if !gateway.Info.TypeGateway {

		if s.BridgeAddress == "" {
			return types.CodeNoBridge, errors.New("No gateway bridge configured")
		}

	}

	if index == nil { //new gw
		gateways = append(gateways, *gateway)
	} else { //update
		gateways[*index] = *gateway
	}

	gwBytes, _ := json.MarshalIndent(&gateways, "", "\t")

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/gateways.json"

	err = util.WriteConfigFile(path, gwBytes)
	if err != nil {
		s.Print("", err, util.PrintOnlyConsole)
		return types.CodeSaving, nil
	}

	s.Print("Gateway Saved", nil, util.PrintOnlyConsole)

	if s.State == util.Running && gateway.Info.Active {
		s.TurnONGateway(gateway.Info.MACAddress)
	}

	return types.CodeOK, nil
}

func (s *Simulator) DeleteGateway(MACAddress lorawan.EUI64) bool {

	gateways := GetGateways()
	found := false

	//gateway in running?
	for _, g := range s.Gateways {

		if g.Info.MACAddress == MACAddress {
			return false
		}

	}

	for i, gateway := range gateways {

		if gateway.Info.MACAddress == MACAddress {

			switch i {

			case 0:
				if len(gateways) == 1 {
					gateways = gateways[:0]
				} else {
					gateways = gateways[1:]
				}

			case len(gateways) - 1:
				gateways = gateways[:len(gateways)-1]

			default:
				gateways = append(gateways[:i], gateways[i+1:]...)

			}

			found = true
			break

		}

	}

	if !found {

		s.Print("", errors.New("Unable to delete a gateway that does not exist"), util.PrintOnlyConsole)
		return false

	}

	gwBytes, _ := json.MarshalIndent(&gateways, "", "\t")

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/gateways.json"

	err = util.WriteConfigFile(path, gwBytes)
	if err != nil {
		s.Print("", err, util.PrintOnlyConsole)
		return false
	}

	s.Print("Gateway Deleted", nil, util.PrintOnlyConsole)

	return true
}

func (s *Simulator) SetDevice(device dev.Device, index *int) (int, error) {

	emptyAddr := lorawan.EUI64{0, 0, 0, 0, 0, 0, 0, 0}

	devices := GetDevices()

	if device.Info.DevEUI == emptyAddr {

		s.Print("DevEUI invalid", nil, util.PrintOnlyConsole)
		return types.CodeErrorAddress, errors.New("Error: DevEUI invalid")

	}

	//dispositivo in running?
	for _, d := range s.Devices {

		if d.Info.DevEUI == device.Info.DevEUI {
			return types.CodeErrorDeviceActive, errors.New("Device is running, unable update")
		}

	}

	code, err := searchName(device.Info.Name, index)
	if err != nil {

		s.Print("Name already used", nil, util.PrintOnlyConsole)
		return code, err

	}

	code, err = searchAddress(device.Info.DevEUI, index)
	if err != nil {

		s.Print("DevEUI already used", nil, util.PrintOnlyConsole)
		return code, err

	}

	if index == nil { //new dev
		devices = append(devices, device)
	} else { //update
		devices[*index] = device
	}

	devBytes, _ := json.MarshalIndent(&devices, "", "\t")

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/devices.json"

	err = util.WriteConfigFile(path, devBytes)
	if err != nil {
		s.Print("", err, util.PrintOnlyConsole)
		return types.CodeSaving, nil
	}

	s.Print("Device Saved", nil, util.PrintOnlyConsole)

	if s.State == util.Running && device.Info.Status.Active {
		s.TurnONDevice(device.Info.DevEUI)
	}

	return types.CodeOK, nil
}

func (s *Simulator) DeleteDevice(DevEUI lorawan.EUI64) bool {

	devices := GetDevices()
	found := false

	for _, d := range s.Devices {

		if d.Info.DevEUI == DevEUI {
			return false
		}

	}

	for i, device := range devices {

		if device.Info.DevEUI == DevEUI {

			switch i {

			case 0:
				if len(devices) == 1 {
					devices = devices[:0]
				} else {
					devices = devices[1:]
				}

			case len(devices) - 1:
				devices = devices[:len(devices)-1]

			default:
				devices = append(devices[:i], devices[i+1:]...)
			}

			found = true
			break

		}
	}

	if !found {

		s.Print("", errors.New("Unable to delete a device that does not exist"), util.PrintOnlyConsole)
		return false

	}

	devBytes, _ := json.MarshalIndent(&devices, "", "\t")

	pathDir, err := util.GetPath()
	if err != nil {
		log.Fatal(err)
	}

	path := pathDir + "/devices.json"

	err = util.WriteConfigFile(path, devBytes)
	if err != nil {
		s.Print("", err, util.PrintOnlyConsole)
		return false
	}

	s.Print("Device Deleted", nil, util.PrintOnlyConsole)

	return true
}

func (s *Simulator) TurnONDevice(DevEUI lorawan.EUI64) bool {

	devices := GetDevices()

	for i := 0; i < len(devices); i++ {

		if devices[i].Info.DevEUI == DevEUI {

			s.Devices = append(s.Devices, &devices[i])
			s.Devices[len(s.Devices)-1].Setup(&s.Resources, &s.State, &s.Forwarder)

			infoDev := mfw.InfoDevice{
				DevEUI:   s.Devices[len(s.Devices)-1].Info.DevEUI,
				Location: s.Devices[len(s.Devices)-1].Info.Location,
				Range:    s.Devices[len(s.Devices)-1].Info.Configuration.Range,
			}

			s.Forwarder.AddDevice(infoDev)

			s.Devices[len(s.Devices)-1].TurnON()

			return true
		}

	}

	return false
}

func (s *Simulator) TurnOFFDevice(DevEUI lorawan.EUI64) bool {

	for i := 0; i < len(s.Devices); i++ {

		if s.Devices[i].Info.DevEUI == DevEUI {

			s.Resources.ExitGroup.Add(1)
			s.DevicesInactive++

			s.Devices[i].TurnOFF()

			//aggiorno il forwarder
			infoDev := mfw.InfoDevice{
				DevEUI:   s.Devices[i].Info.DevEUI,
				Location: s.Devices[i].Info.Location,
				Range:    s.Devices[i].Info.Configuration.Range,
			}

			s.Forwarder.DeleteDevice(infoDev)

			s.Resources.ExitGroup.Wait()

			switch i {

			case 0:
				s.Devices = s.Devices[1:]
			case len(s.Devices) - 1:
				s.Devices = s.Devices[:len(s.Devices)-1]
			default:
				s.Devices = append(s.Devices[:i], s.Devices[i+1:]...)

			}

			s.DevicesInactive--

			return true
		}
	}

	s.Print("", errors.New("Unable to turn off a device that is already turned off"), util.PrintOnlyConsole)

	return false

}

func (s *Simulator) SendMACCommand(cid lorawan.CID, data socket.MacCommand) {

	for i, device := range s.Devices {

		if device.Info.DevEUI == data.DevEUI {

			err := s.Devices[i].SendMACCommand(cid, data.Periodicity)
			if err != nil {
				s.Resources.WebSocket.Emit(socket.EventResponseCommand, "Unable to send command: "+err.Error())
			} else {
				s.Resources.WebSocket.Emit(socket.EventResponseCommand, "MACCommand will be sent to the next uplink")
			}

			return
		}
	}

	s.Resources.WebSocket.Emit(socket.EventResponseCommand, "Unable to send command: device inactive")

}

func (s *Simulator) ChangePayload(pl socket.NewPayload) {

	//decodifico data
	var DevEUI lorawan.EUI64
	DevEUITmp, _ := hex.DecodeString(pl.DevEUI)
	copy(DevEUI[:8], DevEUITmp)

	MType := lorawan.UnconfirmedDataUp
	if pl.MType == "ConfirmedDataUp" {
		MType = lorawan.ConfirmedDataUp
	}

	Payload := &lorawan.DataPayload{
		Bytes: []byte(pl.Payload),
	}

	//Cerco il device tra quelli attivi
	for i, device := range s.Devices {

		if device.Info.DevEUI == DevEUI {

			s.Devices[i].ChangePayload(MType, Payload)

			s.Resources.WebSocket.Emit(socket.EventResponseCommand, "Command executed successfully")

			return
		}

	}

	s.Resources.WebSocket.Emit(socket.EventResponseCommand, "Unable to change payload: device inactive")

}

func (s *Simulator) SendUplinkDevice(pl socket.NewPayload) {

	var DevEUI lorawan.EUI64
	DevEUITmp, _ := hex.DecodeString(pl.DevEUI)
	copy(DevEUI[:8], DevEUITmp)

	MType := lorawan.UnconfirmedDataUp
	if pl.MType == "ConfirmedDataUp" {
		MType = lorawan.ConfirmedDataUp
	}

	//Cerco il device
	index := -1
	for i, device := range s.Devices {

		if device.Info.DevEUI == DevEUI {
			index = i
			s.Devices[i].NewUplink(MType, pl.Payload)
			s.Resources.WebSocket.Emit(socket.EventResponseCommand, "Uplink queued")
			break
		}

	}

	if index == -1 {
		s.Resources.WebSocket.Emit(socket.EventResponseCommand, "Device is not active")
	}

}

func (s *Simulator) ChangeLocation(l socket.NewLocation) bool {

	var DevEUI lorawan.EUI64
	DevEUITmp, _ := hex.DecodeString(l.DevEUI)
	copy(DevEUI[:8], DevEUITmp)

	for i, device := range s.Devices {

		if device.Info.DevEUI == DevEUI {

			s.Devices[i].ChangeLocation(l.Latitude, l.Longitude, l.Altitude)

			info := mfw.InfoDevice{
				DevEUI:   s.Devices[i].Info.DevEUI,
				Location: s.Devices[i].Info.Location,
				Range:    s.Devices[i].Info.Configuration.Range,
			}

			s.Forwarder.UpdateDevice(info)

			return true
		}

	}

	return false
}

func (s *Simulator) TurnONGateway(MACAddress lorawan.EUI64) bool {

	gateways := GetGateways()
	for i := range gateways {

		if gateways[i].Info.MACAddress == MACAddress {

			s.Gateways = append(s.Gateways, &gateways[i])

			s.Gateways[len(s.Gateways)-1].Setup(&s.BridgeAddress, &s.Resources, &s.State, &s.Forwarder)

			infoGw := mfw.InfoGateway{
				Buf:      &s.Gateways[len(s.Gateways)-1].BufferUplink,
				Location: s.Gateways[len(s.Gateways)-1].Info.Location,
			}

			s.Forwarder.AddGateway(infoGw)

			s.Gateways[len(s.Gateways)-1].TurnON()

			break
		}
	}

	return true
}

func (s *Simulator) TurnOFFGateway(MACAddress lorawan.EUI64) bool {

	index := -1
	for i, gateway := range s.Gateways {

		if gateway.Info.MACAddress == MACAddress {
			index = i
			break
		}

	}

	if index == -1 {

		s.Print("", errors.New("Unable to turn off a device that is already turned off"), util.PrintOnlyConsole)
		s.Resources.WebSocket.Emit(socket.EventResponseCommand, "Unable to turn off a device that is already turned off")

		return false
	}

	s.Resources.ExitGroup.Add(1)
	s.Gateways[index].TurnOFF()

	s.Resources.ExitGroup.Wait()

	infoGw := mfw.InfoGateway{
		Buf:      &s.Gateways[index].BufferUplink,
		Location: s.Gateways[index].Info.Location,
	}

	s.Forwarder.DeleteGateway(infoGw)

	switch index {

	case 0:
		s.Gateways = s.Gateways[1:]
	case len(s.Gateways) - 1:
		s.Gateways = s.Gateways[:len(s.Gateways)-1]
	default:
		s.Gateways = append(s.Gateways[:index], s.Gateways[index+1:]...)

	}

	return true
}

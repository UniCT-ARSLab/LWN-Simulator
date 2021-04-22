package repositories

import (
	"errors"

	"github.com/brocaar/lorawan"

	e "github.com/arslab/lwnsimulator/socket"

	"github.com/arslab/lwnsimulator/simulator"
	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	"github.com/arslab/lwnsimulator/simulator/util"
	types "github.com/arslab/lwnsimulator/types"
	socketio "github.com/googollee/go-socket.io"
)

//SimulatorRepository è il repository del simulatore
type SimulatorRepository interface {
	Run() bool
	Stop() bool
	Setup(socketio.Conn)
	SaveBridgeAddress(types.AddressIP) error
	GetBridgeAddress() types.AddressIP
	GetGateways() []gw.Gateway
	AddGateway(gw.Gateway) (int, error)
	UpdateGateway(gw.Gateway, int) (int, error)
	DeleteGateway(lorawan.EUI64) bool
	AddDevice(dev.Device) (int, error)
	GetDevices() []dev.Device
	UpdateDevice(dev.Device, int) (int, error)
	DeleteDevice(lorawan.EUI64) bool
	TurnONDevice(lorawan.EUI64) bool
	TurnOFFDevice(lorawan.EUI64) bool
	SendMACCommand(lorawan.CID, e.MacCommand)
	ChangePayload(e.NewPayload)
	SendUplink(e.NewPayload)
	ChangeLocation(e.NewLocation) bool
	TurnONGateway(lorawan.EUI64) bool
	TurnOFFGateway(lorawan.EUI64) bool
}

type simulatorRepository struct {
	sim *simulator.Simulator
}

//NewSimulatorRepository return repository del simulatore
func NewSimulatorRepository() SimulatorRepository {
	return &simulatorRepository{}
}

func (s *simulatorRepository) Setup(WebSocket socketio.Conn) {
	s.sim = simulator.Setup(WebSocket)
}

func (s *simulatorRepository) Run() bool {

	switch s.sim.State {

	case util.Running:
		s.sim.Print("", errors.New("Already run"), util.PrintOnlyConsole)
		return false

	case util.Stopped:

		s.sim.Run()
	}

	return true
}

func (s *simulatorRepository) Stop() bool {

	switch s.sim.State {

	case util.Stopped:
		s.sim.Print("", errors.New("Already Stopped"), util.PrintOnlyConsole)
		return false

	default: //running
		s.sim.Stop()
		return true
	}

}

func (s *simulatorRepository) SaveBridgeAddress(addr types.AddressIP) error {
	return s.sim.SaveBridgeAddress(addr)
}

func (s *simulatorRepository) GetBridgeAddress() types.AddressIP {
	return s.sim.GetBridgeAddress()
}

func (s *simulatorRepository) GetGateways() []gw.Gateway {
	return simulator.GetGateways()
}

func (s *simulatorRepository) AddGateway(gateway gw.Gateway) (int, error) {
	return s.sim.SetGateway(&gateway, nil)
}

func (s *simulatorRepository) UpdateGateway(gateway gw.Gateway, index int) (int, error) {
	return s.sim.SetGateway(&gateway, &index)
}

func (s *simulatorRepository) DeleteGateway(gateway lorawan.EUI64) bool {
	return s.sim.DeleteGateway(gateway)
}

func (s *simulatorRepository) AddDevice(device dev.Device) (int, error) {
	return s.sim.SetDevice(device, nil)
}

func (s *simulatorRepository) GetDevices() []dev.Device {
	return simulator.GetDevices()
}

func (s *simulatorRepository) UpdateDevice(device dev.Device, index int) (int, error) {
	return s.sim.SetDevice(device, &index)
}

func (s *simulatorRepository) DeleteDevice(device lorawan.EUI64) bool {
	return s.sim.DeleteDevice(device)
}

func (s *simulatorRepository) TurnONDevice(DevEUI lorawan.EUI64) bool {
	return s.sim.TurnONDevice(DevEUI)
}

func (s *simulatorRepository) TurnOFFDevice(DevEUI lorawan.EUI64) bool {
	return s.sim.TurnOFFDevice(DevEUI)
}

func (s *simulatorRepository) SendMACCommand(cid lorawan.CID, data e.MacCommand) {
	s.sim.SendMACCommand(cid, data)
}

func (s *simulatorRepository) ChangePayload(pl e.NewPayload) {
	s.sim.ChangePayload(pl)
}

func (s *simulatorRepository) SendUplink(pl e.NewPayload) {
	s.sim.SendUplinkDevice(pl)
}

func (s *simulatorRepository) ChangeLocation(loc e.NewLocation) bool {
	return s.sim.ChangeLocation(loc)
}

func (s *simulatorRepository) TurnONGateway(MACAddress lorawan.EUI64) bool {
	return s.sim.TurnONGateway(MACAddress)
}

func (s *simulatorRepository) TurnOFFGateway(MACAddress lorawan.EUI64) bool {
	return s.sim.TurnOFFGateway(MACAddress)
}

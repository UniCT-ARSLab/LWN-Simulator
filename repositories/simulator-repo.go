package repositories

import (
	"errors"

	"github.com/brocaar/lorawan"

	"github.com/arslab/lwnsimulator/models"
	e "github.com/arslab/lwnsimulator/socket"

	"github.com/arslab/lwnsimulator/simulator"
	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	"github.com/arslab/lwnsimulator/simulator/util"
	socketio "github.com/googollee/go-socket.io"
)

//SimulatorRepository è il repository del simulatore
type SimulatorRepository interface {
	Run() bool
	Stop() bool
	GetIstance()
	AddWebSocket(*socketio.Conn)
	SaveBridgeAddress(models.AddressIP) error
	GetBridgeAddress() models.AddressIP
	GetGateways() []gw.Gateway
	AddGateway(*gw.Gateway) (int, int, error)
	UpdateGateway(*gw.Gateway) (int, error)
	DeleteGateway(int) bool
	AddDevice(*dev.Device) (int, int, error)
	GetDevices() []dev.Device
	UpdateDevice(*dev.Device) (int, error)
	DeleteDevice(int) bool
	ToggleStateDevice(int)
	SendMACCommand(lorawan.CID, e.MacCommand)
	ChangePayload(e.NewPayload) (string, bool)
	SendUplink(e.NewPayload)
	ChangeLocation(e.NewLocation) bool
	ToggleStateGateway(int)
}

type simulatorRepository struct {
	sim *simulator.Simulator
}

//NewSimulatorRepository return repository del simulatore
func NewSimulatorRepository() SimulatorRepository {
	return &simulatorRepository{}
}

func (s *simulatorRepository) GetIstance() {
	s.sim = simulator.GetIstance()
}

func (s *simulatorRepository) AddWebSocket(socket *socketio.Conn) {
	s.sim.AddWebSocket(socket)
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

func (s *simulatorRepository) SaveBridgeAddress(addr models.AddressIP) error {
	return s.sim.SaveBridgeAddress(addr)
}

func (s *simulatorRepository) GetBridgeAddress() models.AddressIP {
	return s.sim.GetBridgeAddress()
}

func (s *simulatorRepository) GetGateways() []gw.Gateway {
	return s.sim.GetGateways()
}

func (s *simulatorRepository) AddGateway(gateway *gw.Gateway) (int, int, error) {
	return s.sim.SetGateway(gateway, false)
}

func (s *simulatorRepository) UpdateGateway(gateway *gw.Gateway) (int, error) {
	code, _, err := s.sim.SetGateway(gateway, true)
	return code, err
}

func (s *simulatorRepository) DeleteGateway(Id int) bool {
	return s.sim.DeleteGateway(Id)
}

func (s *simulatorRepository) AddDevice(device *dev.Device) (int, int, error) {
	return s.sim.SetDevice(device, false)
}

func (s *simulatorRepository) GetDevices() []dev.Device {
	return s.sim.GetDevices()
}

func (s *simulatorRepository) UpdateDevice(device *dev.Device) (int, error) {
	code, _, err := s.sim.SetDevice(device, true)
	return code, err
}

func (s *simulatorRepository) DeleteDevice(Id int) bool {
	return s.sim.DeleteDevice(Id)
}

func (s *simulatorRepository) ToggleStateDevice(Id int) {
	s.sim.ToggleStateDevice(Id)
}

func (s *simulatorRepository) SendMACCommand(cid lorawan.CID, data e.MacCommand) {
	s.sim.SendMACCommand(cid, data)
}

func (s *simulatorRepository) ChangePayload(pl e.NewPayload) (string, bool) {
	return s.sim.ChangePayload(pl)
}

func (s *simulatorRepository) SendUplink(pl e.NewPayload) {
	s.sim.SendUplink(pl)
}

func (s *simulatorRepository) ChangeLocation(loc e.NewLocation) bool {
	return s.sim.ChangeLocation(loc)
}

func (s *simulatorRepository) ToggleStateGateway(Id int) {
	s.sim.ToggleStateGateway(Id)
}

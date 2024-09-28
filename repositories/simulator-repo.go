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

// SimulatorRepository is the interface that defines the methods that the simulator repository must implement.
type SimulatorRepository interface {
	Run() bool                                 // Run the simulator
	Stop() bool                                // Stop the simulator
	Status() bool                              // Get the status of the simulator
	GetInstance()                              // Get the instance of the simulator
	AddWebSocket(*socketio.Conn)               // Add a websocket connection
	SaveBridgeAddress(models.AddressIP) error  // Save the bridge address
	GetBridgeAddress() models.AddressIP        // Get the bridge address
	GetGateways() []gw.Gateway                 // Get the gateways
	AddGateway(*gw.Gateway) (int, int, error)  // Add a gateway
	UpdateGateway(*gw.Gateway) (int, error)    // Update a gateway
	DeleteGateway(int) bool                    // Delete a gateway
	AddDevice(*dev.Device) (int, int, error)   // Add a device
	GetDevices() []dev.Device                  // Get the devices
	UpdateDevice(*dev.Device) (int, error)     // Update a device
	DeleteDevice(int) bool                     // Delete a device
	ToggleStateDevice(int)                     // Toggle the state of a device
	SendMACCommand(lorawan.CID, e.MacCommand)  // Send a MAC command
	ChangePayload(e.NewPayload) (string, bool) // Change the payload
	SendUplink(e.NewPayload)                   // Send an uplink
	ChangeLocation(e.NewLocation) bool         // Change the location
	ToggleStateGateway(int)                    // Toggle the state of a gateway
}

// simulatorRepository repository struct
type simulatorRepository struct {
	sim *simulator.Simulator
}

// NewSimulatorRepository create a new repository instance
func NewSimulatorRepository() SimulatorRepository {
	return &simulatorRepository{}
}

// --- Repository calls to Simulator, no need to comment them, they are self-explanatory ---
// Check the simulator methods to see what they do

func (s *simulatorRepository) GetInstance() {
	s.sim = simulator.GetInstance()
}

func (s *simulatorRepository) AddWebSocket(socket *socketio.Conn) {
	s.sim.AddWebSocket(socket)
}

// Run If the simulator is stopped, it starts it and returns True, otherwise it prints an error message and returns False.
func (s *simulatorRepository) Run() bool {
	switch s.sim.State {
	case util.Running:
		s.sim.Print("", errors.New("Already run"), util.PrintOnlyConsole)
		return false
	default: // State = util.Stopped
		s.sim.Run()
	}
	return true
}

// Stop If the simulator is running, it stops it and returns True, otherwise it prints an error message and returns False.
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

// Status returns True if the simulator is running, otherwise it returns False.
func (s *simulatorRepository) Status() bool {
	if s.sim.State == util.Running {
		return true
	}
	return false
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

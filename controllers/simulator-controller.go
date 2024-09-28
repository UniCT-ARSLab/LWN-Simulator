package controllers

import (
	"github.com/arslab/lwnsimulator/models"
	repo "github.com/arslab/lwnsimulator/repositories"

	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	e "github.com/arslab/lwnsimulator/socket"
	"github.com/brocaar/lorawan"
	socketio "github.com/googollee/go-socket.io"
)

// SimulatorController is the interface that defines the methods that the simulator controller must implement.
type SimulatorController interface {
	Run() bool                                 // Run the simulator
	Stop() bool                                // Stop the simulator
	Status() bool                              // Get the status of the simulator
	GetInstance()                              // Get the instance of the simulator repository
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

// simulatorController controller struct
type simulatorController struct {
	repo repo.SimulatorRepository
}

// NewSimulatorController create a new controller instance with the provided repository
func NewSimulatorController(repo repo.SimulatorRepository) SimulatorController {
	return &simulatorController{
		repo: repo,
	}
}

// --- Controller calls to Repository, no need to comment them, they are self-explanatory ---
// Check the repository methods to see what they do

func (c *simulatorController) GetInstance() {
	c.repo.GetInstance()
}
func (c *simulatorController) AddWebSocket(socket *socketio.Conn) {
	c.repo.AddWebSocket(socket)
}

func (c *simulatorController) Run() bool {
	return c.repo.Run()
}

func (c *simulatorController) Stop() bool {
	return c.repo.Stop()
}

func (c *simulatorController) Status() bool {
	return c.repo.Status()
}

func (c *simulatorController) SaveBridgeAddress(addr models.AddressIP) error {
	return c.repo.SaveBridgeAddress(addr)
}

func (c *simulatorController) GetBridgeAddress() models.AddressIP {
	return c.repo.GetBridgeAddress()
}

func (c *simulatorController) GetGateways() []gw.Gateway {
	return c.repo.GetGateways()
}

func (c *simulatorController) AddGateway(gateway *gw.Gateway) (int, int, error) {
	return c.repo.AddGateway(gateway)
}

func (c *simulatorController) UpdateGateway(gateway *gw.Gateway) (int, error) {
	return c.repo.UpdateGateway(gateway)
}

func (c *simulatorController) DeleteGateway(Id int) bool {
	return c.repo.DeleteGateway(Id)
}

func (c *simulatorController) AddDevice(device *dev.Device) (int, int, error) {
	return c.repo.AddDevice(device)
}

func (c *simulatorController) GetDevices() []dev.Device {
	return c.repo.GetDevices()
}

func (c *simulatorController) UpdateDevice(device *dev.Device) (int, error) {
	return c.repo.UpdateDevice(device)
}

func (c *simulatorController) DeleteDevice(Id int) bool {
	return c.repo.DeleteDevice(Id)
}

func (c *simulatorController) ToggleStateDevice(Id int) {
	c.repo.ToggleStateDevice(Id)
}

func (c *simulatorController) SendMACCommand(cid lorawan.CID, data e.MacCommand) {
	c.repo.SendMACCommand(cid, data)
}

func (c *simulatorController) ChangePayload(pl e.NewPayload) (string, bool) {
	return c.repo.ChangePayload(pl)
}

func (c *simulatorController) SendUplink(pl e.NewPayload) {
	c.repo.SendUplink(pl)
}

func (c *simulatorController) ChangeLocation(loc e.NewLocation) bool {
	return c.repo.ChangeLocation(loc)
}

func (c *simulatorController) ToggleStateGateway(Id int) {
	c.repo.ToggleStateGateway(Id)
}

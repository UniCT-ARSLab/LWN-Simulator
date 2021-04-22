package controllers

import (
	repo "github.com/arslab/lwnsimulator/repositories"
	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	e "github.com/arslab/lwnsimulator/socket"
	types "github.com/arslab/lwnsimulator/types"
	"github.com/brocaar/lorawan"
	socketio "github.com/googollee/go-socket.io"
)

//SimulatorController interfaccia controller
type SimulatorController interface {
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

type simulatorController struct {
	repo repo.SimulatorRepository
}

//NewSimulatorController return il controller
func NewSimulatorController(repo repo.SimulatorRepository) SimulatorController {
	return &simulatorController{
		repo: repo,
	}
}

func (c *simulatorController) Setup(WebSocket socketio.Conn) {
	c.repo.Setup(WebSocket)
}

func (c *simulatorController) Run() bool {
	return c.repo.Run()
}

func (c *simulatorController) Stop() bool {
	return c.repo.Stop()
}

func (c *simulatorController) SaveBridgeAddress(addr types.AddressIP) error {
	return c.repo.SaveBridgeAddress(addr)
}

func (c *simulatorController) GetBridgeAddress() types.AddressIP {
	return c.repo.GetBridgeAddress()
}

func (c *simulatorController) GetGateways() []gw.Gateway {
	return c.repo.GetGateways()
}

func (c *simulatorController) AddGateway(gateway gw.Gateway) (int, error) {
	return c.repo.AddGateway(gateway)
}

func (c *simulatorController) UpdateGateway(gateway gw.Gateway, index int) (int, error) {
	return c.repo.UpdateGateway(gateway, index)
}

func (c *simulatorController) DeleteGateway(gateway lorawan.EUI64) bool {
	return c.repo.DeleteGateway(gateway)
}

func (c *simulatorController) AddDevice(device dev.Device) (int, error) {
	return c.repo.AddDevice(device)
}

func (c *simulatorController) GetDevices() []dev.Device {
	return c.repo.GetDevices()
}

func (c *simulatorController) UpdateDevice(device dev.Device, index int) (int, error) {
	return c.repo.UpdateDevice(device, index)
}

func (c *simulatorController) DeleteDevice(device lorawan.EUI64) bool {
	return c.repo.DeleteDevice(device)
}

func (c *simulatorController) TurnONDevice(DevEUI lorawan.EUI64) bool {
	return c.repo.TurnONDevice(DevEUI)
}

func (c *simulatorController) TurnOFFDevice(DevEUI lorawan.EUI64) bool {
	return c.repo.TurnOFFDevice(DevEUI)
}

func (c *simulatorController) SendMACCommand(cid lorawan.CID, data e.MacCommand) {
	c.repo.SendMACCommand(cid, data)
}

func (c *simulatorController) ChangePayload(pl e.NewPayload) {
	c.repo.ChangePayload(pl)
}

func (c *simulatorController) SendUplink(pl e.NewPayload) {
	c.repo.SendUplink(pl)
}

func (c *simulatorController) ChangeLocation(loc e.NewLocation) bool {
	return c.repo.ChangeLocation(loc)
}

func (c *simulatorController) TurnONGateway(MACAddress lorawan.EUI64) bool {
	return c.repo.TurnONGateway(MACAddress)
}

func (c *simulatorController) TurnOFFGateway(MACAddress lorawan.EUI64) bool {
	return c.repo.TurnOFFGateway(MACAddress)
}

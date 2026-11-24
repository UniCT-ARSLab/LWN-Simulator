package registry

import (
	"sort"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/validation"
)

type Registry struct {
	devices  map[string]contracts.Device
	gateways map[string]contracts.Gateway
}

func New(devices []contracts.Device, gateways []contracts.Gateway) (*Registry, error) {
	if err := validation.ValidateHardware(devices, gateways); err != nil {
		return nil, err
	}

	registry := &Registry{
		devices:  make(map[string]contracts.Device, len(devices)),
		gateways: make(map[string]contracts.Gateway, len(gateways)),
	}

	for _, device := range devices {
		registry.devices[device.ID] = cloneDevice(device)
	}
	for _, gateway := range gateways {
		registry.gateways[gateway.ID] = cloneGateway(gateway)
	}

	return registry, nil
}

func (r *Registry) Device(id string) (contracts.Device, bool) {
	device, ok := r.devices[id]
	return cloneDevice(device), ok
}

func (r *Registry) Gateway(id string) (contracts.Gateway, bool) {
	gateway, ok := r.gateways[id]
	return cloneGateway(gateway), ok
}

func (r *Registry) Devices() []contracts.Device {
	ids := make([]string, 0, len(r.devices))
	for id := range r.devices {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	devices := make([]contracts.Device, 0, len(ids))
	for _, id := range ids {
		devices = append(devices, cloneDevice(r.devices[id]))
	}
	return devices
}

func (r *Registry) Gateways() []contracts.Gateway {
	ids := make([]string, 0, len(r.gateways))
	for id := range r.gateways {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	gateways := make([]contracts.Gateway, 0, len(ids))
	for _, id := range ids {
		gateways = append(gateways, cloneGateway(r.gateways[id]))
	}
	return gateways
}

func (r *Registry) ActiveDevices() []contracts.Device {
	devices := make([]contracts.Device, 0)
	for _, device := range r.Devices() {
		if device.Active {
			devices = append(devices, device)
		}
	}
	return devices
}

func (r *Registry) ActiveVirtualGateways() []contracts.Gateway {
	gateways := make([]contracts.Gateway, 0)
	for _, gateway := range r.Gateways() {
		if gateway.Active && gateway.Type == contracts.Virtual {
			gateways = append(gateways, gateway)
		}
	}
	return gateways
}

func (r *Registry) ActiveGateways() []contracts.Gateway {
	gateways := make([]contracts.Gateway, 0)
	for _, gateway := range r.Gateways() {
		if gateway.Active {
			gateways = append(gateways, gateway)
		}
	}
	return gateways
}

func (r *Registry) Snapshot() contracts.SimulationSnapshot {
	devices := r.Devices()
	runtimeDevices := make([]contracts.RuntimeDevice, 0, len(devices))
	for _, device := range devices {
		status := contracts.RuntimeStatusRegistered
		if device.Active {
			status = contracts.RuntimeStatusActive
		}
		runtimeDevices = append(runtimeDevices, contracts.RuntimeDevice{
			ID:                     device.ID,
			Name:                   device.Name,
			Active:                 device.Active,
			Status:                 status,
			CurrentDataRate:        initialDataRate(device),
			CurrentSpreadingFactor: regional.Profile(device.LocationConfig.Region, initialDataRate(device)).SpreadingFactor,
			LastFPort:              device.FrameConfig.FPort,
		})
	}

	gateways := r.Gateways()
	runtimeGateways := make([]contracts.RuntimeGateway, 0, len(gateways))
	for _, gateway := range gateways {
		status := contracts.RuntimeStatusRegistered
		if gateway.Active {
			status = contracts.RuntimeStatusActive
		}
		runtimeGateways = append(runtimeGateways, contracts.RuntimeGateway{
			ID:     gateway.ID,
			Name:   gateway.Name,
			Active: gateway.Active,
			Status: status,
		})
	}

	return contracts.SimulationSnapshot{
		Devices:  runtimeDevices,
		Gateways: runtimeGateways,
		Metrics: contracts.SimulationMetrics{
			ActiveDevices:  int64(len(r.ActiveDevices())),
			ActiveGateways: int64(len(r.ActiveGateways())),
		},
	}
}

func initialDataRate(device contracts.Device) int {
	if device.AdvancedConfig.UplinkDataRate != nil {
		return regional.ClampDataRate(device.LocationConfig.Region, *device.AdvancedConfig.UplinkDataRate)
	}
	return regional.MustPlan(device.LocationConfig.Region).DefaultDataRate
}

func cloneDevice(device contracts.Device) contracts.Device {
	cloned := device
	if device.ABPConfig != nil {
		value := *device.ABPConfig
		cloned.ABPConfig = &value
	}
	if device.OOTAConfig != nil {
		value := *device.OOTAConfig
		cloned.OOTAConfig = &value
	}
	cloned.LocationConfig.Latitude = clonePointer(device.LocationConfig.Latitude)
	cloned.LocationConfig.Longitude = clonePointer(device.LocationConfig.Longitude)
	cloned.LocationConfig.Altitude = clonePointer(device.LocationConfig.Altitude)
	cloned.RX1Config.Delay = clonePointer(device.RX1Config.Delay)
	cloned.RX1Config.Duration = clonePointer(device.RX1Config.Duration)
	cloned.RX1Config.DataRateOffset = clonePointer(device.RX1Config.DataRateOffset)
	cloned.RX2Config.Delay = clonePointer(device.RX2Config.Delay)
	cloned.RX2Config.Duration = clonePointer(device.RX2Config.Duration)
	cloned.RX2Config.DataRate = clonePointer(device.RX2Config.DataRate)
	cloned.FrameConfig.Retransmission = clonePointer(device.FrameConfig.Retransmission)
	cloned.FrameConfig.FCntUp = cloneOptionalCounter(device.FrameConfig.FCntUp)
	cloned.FrameConfig.FCntDown = cloneOptionalCounter(device.FrameConfig.FCntDown)
	cloned.AdvancedConfig.UplinkDataRate = clonePointer(device.AdvancedConfig.UplinkDataRate)
	return cloned
}

func cloneGateway(gateway contracts.Gateway) contracts.Gateway {
	cloned := gateway
	cloned.Altitude = clonePointer(gateway.Altitude)
	cloned.Latitude = clonePointer(gateway.Latitude)
	cloned.Longitude = clonePointer(gateway.Longitude)
	cloned.KeepAlive = clonePointer(gateway.KeepAlive)
	cloned.GatewayIPv4 = clonePointer(gateway.GatewayIPv4)
	cloned.GatewayPort = clonePointer(gateway.GatewayPort)
	return cloned
}

func clonePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneOptionalCounter(value **int) **int {
	if value == nil {
		return nil
	}
	if *value == nil {
		var empty *int
		return &empty
	}
	cloned := **value
	inner := &cloned
	return &inner
}

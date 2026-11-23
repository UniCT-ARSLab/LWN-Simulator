package registry

import (
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestRegistrySortsFiltersAndSnapshotsEntities(t *testing.T) {
	devices := []contracts.Device{
		{ID: "00000000-0000-0000-0000-000000000002", Name: "Inactive", Active: false, FrameConfig: contracts.FrameConfig{FPort: 1}},
		{ID: "00000000-0000-0000-0000-000000000001", Name: "Active", Active: true, FrameConfig: contracts.FrameConfig{FPort: 2}, AdvancedConfig: contracts.AdvancedConfig{UplinkDataRate: intPointer(5)}},
	}
	// Complete the shared validation fields without changing the ordering assertions.
	for index := range devices {
		completeRegistryDevice(&devices[index])
	}
	virtual := registryGateway("00000000-0000-0000-0000-000000000004", true, contracts.Virtual)
	real := registryGateway("00000000-0000-0000-0000-000000000003", true, contracts.Real)
	real.KeepAlive = nil
	real.GatewayIPv4 = stringPointer("127.0.0.1")
	real.GatewayPort = int32Pointer(1700)
	registry, err := New(devices, []contracts.Gateway{virtual, real})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	ordered := registry.Devices()
	if ordered[0].ID > ordered[1].ID || ordered[0].Name != "Active" {
		t.Fatalf("devices are not sorted deterministically: %+v", ordered)
	}
	gateways := registry.Gateways()
	if gateways[0].ID > gateways[1].ID {
		t.Fatalf("gateways are not sorted deterministically: %+v", gateways)
	}
	if len(registry.ActiveDevices()) != 1 || len(registry.ActiveGateways()) != 2 || len(registry.ActiveVirtualGateways()) != 1 {
		t.Fatalf("unexpected active filters")
	}
	if _, ok := registry.Device(devices[0].ID); !ok {
		t.Fatal("device lookup failed")
	}
	if _, ok := registry.Gateway(real.ID); !ok {
		t.Fatal("gateway lookup failed")
	}
	if _, ok := registry.Device("missing"); ok {
		t.Fatal("missing device unexpectedly found")
	}

	snapshot := registry.Snapshot()
	if len(snapshot.Devices) != 2 || len(snapshot.Gateways) != 2 || snapshot.Metrics.ActiveDevices != 1 || snapshot.Metrics.ActiveGateways != 2 {
		t.Fatalf("unexpected registry snapshot: %+v", snapshot)
	}
	if snapshot.Devices[0].CurrentDataRate != 5 || snapshot.Devices[0].CurrentSpreadingFactor != 7 {
		t.Fatalf("configured data rate was not clamped: %+v", snapshot.Devices[0])
	}
}

func TestRegistryRejectsInvalidHardware(t *testing.T) {
	if _, err := New([]contracts.Device{{ID: "invalid"}}, nil); err == nil {
		t.Fatal("expected registry creation to validate hardware")
	}
}

func TestRegistryOwnsDeepCopiesOfHardware(t *testing.T) {
	device := contracts.Device{ID: "00000000-0000-0000-0000-000000000001", Name: "Original", Active: true, FrameConfig: contracts.FrameConfig{FPort: 1}}
	completeRegistryDevice(&device)
	gateway := registryGateway("00000000-0000-0000-0000-000000000004", true, contracts.Virtual)
	registry, err := New([]contracts.Device{device}, []contracts.Gateway{gateway})
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	device.ABPConfig.AppSKey = "11111111111111111111111111111111"
	*gateway.Latitude = 0
	storedDevice, _ := registry.Device(device.ID)
	storedGateway, _ := registry.Gateway(gateway.ID)
	if storedDevice.ABPConfig.AppSKey == device.ABPConfig.AppSKey || *storedGateway.Latitude == 0 {
		t.Fatal("caller mutation leaked into registry")
	}
	storedDevice.ABPConfig.AppSKey = "22222222222222222222222222222222"
	storedAgain, _ := registry.Device(device.ID)
	if storedAgain.ABPConfig.AppSKey == storedDevice.ABPConfig.AppSKey {
		t.Fatal("lookup mutation leaked back into registry")
	}
}

func completeRegistryDevice(device *contracts.Device) {
	device.DevEUI = "70B3D57ED000000" + device.ID[len(device.ID)-1:]
	device.Class = contracts.ClassA
	device.Activation = contracts.ABP
	device.ABPConfig = &contracts.ABPConfig{DevAddr: "26011BDA", NwkSKey: "00000000000000000000000000000000", AppSKey: "00000000000000000000000000000000"}
	device.LocationConfig = contracts.LocationConfig{Latitude: float32Pointer(37.5), Longitude: float32Pointer(15.1), Altitude: float32Pointer(100), Region: contracts.EU868}
	device.RX1Config = contracts.RX1Config{Delay: intPointer(1000), Duration: intPointer(1000), DataRateOffset: intPointer(0)}
	device.RX2Config = contracts.RX2Config{Delay: intPointer(2000), Duration: intPointer(1000), ChannelFrequency: 869525000, DataRate: intPointer(5), ACKTimeout: 2000}
	device.FrameConfig.Retransmission = intPointer(0)
	device.PayloadConfig = contracts.PayloadConfig{UplinkInterval: 10, OversizedPayloadBehavior: contracts.Truncate, MType: contracts.UnconfirmedDataUp}
	device.AdvancedConfig.AntennaRange = 1000
}

func registryGateway(id string, active bool, gatewayType contracts.GatewayType) contracts.Gateway {
	return contracts.Gateway{ID: id, Active: active, Name: "Gateway", Type: gatewayType, GatewayEUI: "A8404100010001" + id[len(id)-2:], KeepAlive: int32Pointer(30), Latitude: float32Pointer(37.5), Longitude: float32Pointer(15.1), Altitude: float32Pointer(100)}
}

func intPointer(value int) *int             { return &value }
func int32Pointer(value int32) *int32       { return &value }
func float32Pointer(value float32) *float32 { return &value }
func stringPointer(value string) *string    { return &value }

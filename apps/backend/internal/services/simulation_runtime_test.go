package services

import (
	"context"
	"errors"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine"
	enginetypes "github.com/Giuseppe-Compagnone/lwn-engine/types"
	"lwn-simulator-backend/internal/apperrors"
)

type populatedSimulationDeviceSource struct{ devices []contracts.Device }

func (source populatedSimulationDeviceSource) GetDevices(contracts.GetDevicesRequest) (contracts.GetDevicesResponse, error) {
	return contracts.GetDevicesResponse{Devices: source.devices}, nil
}

type populatedSimulationGatewaySource struct{ gateways []contracts.Gateway }

func (source populatedSimulationGatewaySource) GetGateways(contracts.GetGatewaysRequest) (contracts.GetGatewaysResponse, error) {
	return contracts.GetGatewaysResponse{Gateways: source.gateways}, nil
}

func TestSimulationRuntimeCommandsAndEventPagination(t *testing.T) {
	clock := enginetypes.NewManualClock(0)
	device := validSimulationDevice("a1c6e32b-4f0d-4b50-9fc5-000000000081")
	gateway := validSimulationGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c81")
	service := NewSimulationService(
		populatedSimulationDeviceSource{devices: []contracts.Device{device}},
		populatedSimulationGatewaySource{gateways: []contracts.Gateway{gateway}},
		enginetypes.Options{Clock: clock, EventBuffer: 256},
	)
	if _, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatalf("start simulation: %v", err)
	}
	defer func() { _, _ = service.Stop(context.Background()) }()

	if err := clock.Advance(5 * time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	waitForSimulationFrame(t, service, device.ID)

	uplink, err := service.QueueUplink(contracts.SimulationUplinkRequest{DeviceID: device.ID})
	if err != nil || uplink.ID == "" {
		t.Fatalf("queue uplink: %+v, %v", uplink, err)
	}
	payload := []byte("downlink")
	downlink, err := service.QueueDownlink(contracts.SimulationDownlinkRequest{DeviceID: device.ID, Payload: &payload})
	if err != nil || downlink.ID == "" {
		t.Fatalf("queue downlink: %+v, %v", downlink, err)
	}
	dataRate := 3
	mac, err := service.QueueMACCommand(contracts.SimulationMACCommandRequest{
		DeviceID: device.ID,
		Command:  contracts.SimulationMACCommand{Type: contracts.LinkADRReq, DataRate: &dataRate},
	})
	if err != nil || mac.ID == "" {
		t.Fatalf("queue MAC command: %+v, %v", mac, err)
	}

	firstPage, err := service.Events(0, 2)
	if err != nil {
		t.Fatalf("get first event page: %v", err)
	}
	if len(firstPage.Events) != 2 || firstPage.LastSequence != firstPage.Events[1].Sequence {
		t.Fatalf("unexpected first page: %+v", firstPage)
	}
	secondPage, err := service.Events(firstPage.LastSequence, 100)
	if err != nil {
		t.Fatalf("get second event page: %v", err)
	}
	if len(secondPage.Events) == 0 || secondPage.Events[0].Sequence <= firstPage.LastSequence {
		t.Fatalf("unexpected second page: %+v", secondPage)
	}
	filtered, err := service.EventsFiltered(0, 100, SimulationEventFilters{
		DeviceID: device.ID,
		Types:    map[contracts.SimulationEventType]struct{}{contracts.DeviceUplinkTransmitted: {}},
	})
	if err != nil {
		t.Fatalf("get filtered events: %v", err)
	}
	if len(filtered.Events) == 0 {
		t.Fatal("expected at least one filtered uplink event")
	}
	for _, event := range filtered.Events {
		if event.DeviceID == nil || *event.DeviceID != device.ID || event.Type != contracts.DeviceUplinkTransmitted {
			t.Fatalf("filter returned unrelated event: %+v", event)
		}
	}
}

func TestSimulationRuntimeSynchronizesHardwareRegistry(t *testing.T) {
	service := NewSimulationService(simulationDeviceSource{}, simulationGatewaySource{}, enginetypes.Options{Clock: enginetypes.NewManualClock(0)})
	if _, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatalf("start simulation: %v", err)
	}
	defer func() { _, _ = service.Stop(context.Background()) }()

	device := validSimulationDevice("a1c6e32b-4f0d-4b50-9fc5-000000000082")
	gateway := validSimulationGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c82")
	if err := service.RegisterDevice(device); err != nil {
		t.Fatalf("register device: %v", err)
	}
	if err := service.RegisterGateway(gateway); err != nil {
		t.Fatalf("register gateway: %v", err)
	}
	snapshot, err := service.Snapshot()
	if err != nil || snapshot.State.DeviceCount != 2 || snapshot.State.GatewayCount != 1 {
		t.Fatalf("unexpected synchronized snapshot: %+v, %v", snapshot.State, err)
	}
	device.PayloadConfig.Payload = "Updated runtime payload"
	if err := service.UpdateDevice(device); err != nil {
		t.Fatalf("update device: %v", err)
	}
	if err := service.RemoveDevice(device.ID); err != nil {
		t.Fatalf("remove device: %v", err)
	}
	if err := service.RemoveDevice("a1c6e32b-4f0d-4b50-9fc5-000000000080"); err != nil {
		t.Fatalf("remove initial device: %v", err)
	}
	if err := service.RemoveGateway(gateway.ID); err != nil {
		t.Fatalf("remove gateway: %v", err)
	}
	snapshot, _ = service.Snapshot()
	if snapshot.State.DeviceCount != 0 || snapshot.State.GatewayCount != 0 {
		t.Fatalf("hardware was not removed from runtime: %+v", snapshot.State)
	}
}

func TestSimulationDownlinkAndMACCommandValidation(t *testing.T) {
	if _, err := simulationDownlink(contracts.SimulationDownlinkRequest{DeviceID: testDeviceID1}); err == nil {
		t.Fatal("empty downlink should be rejected")
	}
	invalidType := contracts.SimulationMACCommand{Type: "unsupported"}
	if _, err := simulationMACCommand(invalidType); err == nil {
		t.Fatal("unsupported MAC command should be rejected")
	}
	mask := int32(65535)
	delay := 2
	command, err := simulationMACCommand(contracts.SimulationMACCommand{
		Type: contracts.LinkADRReq, ChannelMask: &mask, DelaySeconds: &delay,
	})
	if err != nil {
		t.Fatalf("convert MAC command: %v", err)
	}
	if command.ChannelMask == nil || *command.ChannelMask != 65535 || command.Delay == nil || *command.Delay != 2*time.Second {
		t.Fatalf("MAC command fields were not converted: %+v", command)
	}
}

func TestWrapEngineCommandErrorClassifiesFailures(t *testing.T) {
	tests := []struct {
		err    error
		target error
	}{
		{engine.ErrInvalidTransition, apperrors.ErrConflict},
		{engine.ErrDeviceNotFound, apperrors.ErrNotFound},
		{errors.New("invalid payload"), apperrors.ErrInvalid},
	}
	for _, test := range tests {
		if got := wrapEngineCommandError("test", test.err); !errors.Is(got, test.target) {
			t.Errorf("wrapEngineCommandError(%v) = %v, want %v", test.err, got, test.target)
		}
	}
}

func waitForSimulationFrame(t *testing.T, service *SimulationService, deviceID string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := service.Snapshot()
		if err == nil {
			for _, device := range snapshot.Devices {
				if device.ID == deviceID && device.FrameCounterUp > 0 && !device.PendingConfirmedUplink {
					return
				}
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for the initial uplink to complete")
}

func validSimulationDevice(id string) contracts.Device {
	return contracts.Device{
		ID: id, Active: true, Name: "Runtime device", DevEUI: "70B3D57ED0000081",
		Class: contracts.ClassC, Activation: contracts.ABP,
		ABPConfig:      &contracts.ABPConfig{DevAddr: "26011BDA", NwkSKey: "00112233445566778899AABBCCDDEEFF", AppSKey: "FFEEDDCCBBAA99887766554433221100"},
		LocationConfig: contracts.LocationConfig{Latitude: serviceFloat32Ptr(37.5), Longitude: serviceFloat32Ptr(15.1), Altitude: serviceFloat32Ptr(100), Region: contracts.EU868},
		RX1Config:      contracts.RX1Config{Delay: serviceIntPtr(1000), Duration: serviceIntPtr(1000), DataRateOffset: serviceIntPtr(0)},
		RX2Config:      contracts.RX2Config{Delay: serviceIntPtr(2000), Duration: serviceIntPtr(1000), ChannelFrequency: 869525000, DataRate: serviceIntPtr(5), ACKTimeout: 2000},
		FrameConfig:    contracts.FrameConfig{FPort: 1, Retransmission: serviceIntPtr(0), FCntUp: serviceOptionalCounter(0), FCntDown: serviceOptionalCounter(0)},
		PayloadConfig:  contracts.PayloadConfig{UplinkInterval: 60, OversizedPayloadBehavior: contracts.Truncate, MType: contracts.UnconfirmedDataUp, Payload: "test"},
		AdvancedConfig: contracts.AdvancedConfig{AntennaRange: 1000, ADREnabled: true},
	}
}

func validSimulationGateway(id string) contracts.Gateway {
	return contracts.Gateway{
		ID: id, Active: true, Name: "Runtime gateway", Type: contracts.Virtual,
		GatewayEUI: "A840410001000181", KeepAlive: serviceInt32Ptr(30),
		Latitude: serviceFloat32Ptr(37.5), Longitude: serviceFloat32Ptr(15.1), Altitude: serviceFloat32Ptr(100),
	}
}

func serviceIntPtr(value int) *int             { return &value }
func serviceInt32Ptr(value int32) *int32       { return &value }
func serviceFloat32Ptr(value float32) *float32 { return &value }
func serviceOptionalCounter(value int) **int {
	inner := &value
	return &inner
}

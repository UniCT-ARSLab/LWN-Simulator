package services

import (
	"errors"
	"reflect"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"lwn-simulator-backend/internal/apperrors"
)

func TestLiveDeviceUpdatesAllowStatusPayloadEncodingAndPosition(t *testing.T) {
	for _, paused := range []bool{false, true} {
		t.Run(map[bool]string{false: "running", true: "paused"}[paused], func(t *testing.T) {
			device := validSimulationDevice(testDeviceID1)
			repo := &mockDeviceRepository{devices: []contracts.Device{device}}
			devices := NewDeviceService(repo)
			simulation := NewSimulationService(devices, simulationGatewaySource{}, types.Options{Clock: types.NewManualClock(0)})
			devices.SetRuntimeSynchronizer(simulation)
			if _, err := simulation.Start(t.Context(), contracts.SimulationConfig{Speed: 1}); err != nil {
				t.Fatal(err)
			}
			defer simulation.Stop(t.Context())
			if paused {
				if _, err := simulation.Pause(); err != nil {
					t.Fatal(err)
				}
			}
			updated := device
			updated.LocationConfig.Latitude = serviceFloat32Ptr(40)
			updated.LocationConfig.Longitude = serviceFloat32Ptr(16)
			updated.LocationConfig.Altitude = serviceFloat32Ptr(120)
			updated.PayloadConfig.Payload = "AQID"
			updated.PayloadConfig.Base64Encoded = true
			if _, err := devices.UpdateDevice(contracts.UpdateDeviceRequest{ID: device.ID, Device: updated}); err != nil {
				t.Fatal(err)
			}
			if repo.lastUpdatedDevice.PayloadConfig.Payload != "AQID" || !reflect.DeepEqual(repo.lastUpdatedDevice.LocationConfig, updated.LocationConfig) {
				t.Fatal("payload or position was not persisted")
			}
			for _, mutation := range []func(*contracts.Device){
				func(d *contracts.Device) { d.Name = "Changed" },
				func(d *contracts.Device) { d.Class = contracts.ClassA },
				func(d *contracts.Device) { d.PayloadConfig.UplinkInterval = 5 },
				func(d *contracts.Device) { d.PayloadConfig.MType = contracts.ConfirmedDataUp },
				func(d *contracts.Device) { d.FrameConfig.FPort = 2 },
				func(d *contracts.Device) { d.RX1Config.Delay = serviceIntPtr(2000) },
				func(d *contracts.Device) { d.LocationConfig.Region = contracts.US915 },
				func(d *contracts.Device) {
					d.PayloadConfig.PayloadCodec = &contracts.PayloadCodec{Enabled: true, Script: "function decodeUplink(input) { return {data:{}}; }"}
				},
			} {
				forbidden := updated
				mutation(&forbidden)
				writes := repo.updateCalls
				_, err := devices.UpdateDevice(contracts.UpdateDeviceRequest{ID: device.ID, Device: forbidden})
				if !errors.Is(err, apperrors.ErrInvalid) || repo.updateCalls != writes {
					t.Fatalf("forbidden live edit was persisted or returned wrong error: %v", err)
				}
			}
			updated.Active = false
			if _, err := devices.UpdateDevice(contracts.UpdateDeviceRequest{ID: device.ID, Device: updated}); err != nil {
				t.Fatal(err)
			}
			if _, err := simulation.Stop(t.Context()); err != nil {
				t.Fatal(err)
			}
			updated.Name = "After stop"
			if _, err := devices.UpdateDevice(contracts.UpdateDeviceRequest{ID: device.ID, Device: updated}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLiveGatewayUpdatesAllowPositionAndRejectOtherChanges(t *testing.T) {
	for _, phase := range []string{"running", "paused"} {
		t.Run(phase, func(t *testing.T) {
			gateway := validSimulationGateway("a1c6e32b-4f0d-4b50-9fc5-000000000083")
			repo := &mockGatewayRepository{gateways: []contracts.Gateway{gateway}}
			gateways := NewGatewayService(repo)
			devices := NewDeviceService(&mockDeviceRepository{devices: []contracts.Device{validSimulationDevice(testDeviceID1)}})
			simulation := NewSimulationService(devices, gateways, types.Options{Clock: types.NewManualClock(0)})
			gateways.SetRuntimeSynchronizer(simulation)
			if _, err := simulation.Start(t.Context(), contracts.SimulationConfig{Speed: 1}); err != nil {
				t.Fatal(err)
			}
			defer simulation.Stop(t.Context())
			if phase == "paused" {
				if _, err := simulation.Pause(); err != nil {
					t.Fatal(err)
				}
			}
			moved := gateway
			moved.Latitude = serviceFloat32Ptr(40)
			moved.Longitude = serviceFloat32Ptr(16)
			moved.Altitude = serviceFloat32Ptr(120)
			if _, err := gateways.UpdateGateway(contracts.UpdateGatewayRequest{ID: gateway.ID, Gateway: moved}); err != nil {
				t.Fatal(err)
			}
			forbidden := moved
			forbidden.Name = "Changed"
			if _, err := gateways.UpdateGateway(contracts.UpdateGatewayRequest{ID: gateway.ID, Gateway: forbidden}); !errors.Is(err, apperrors.ErrInvalid) {
				t.Fatalf("expected invalid update, got %v", err)
			}
			if !reflect.DeepEqual(repo.gateways[0], moved) {
				t.Fatal("repository did not retain the valid position update")
			}
		})
	}
}

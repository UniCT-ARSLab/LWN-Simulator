package services

import (
	"errors"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type fakeDeviceRuntimeSynchronizer struct {
	err        error
	registered []contracts.Device
	updated    []contracts.Device
	removed    []string
}

func (runtime *fakeDeviceRuntimeSynchronizer) AcquireHardwareMutation() func() { return func() {} }
func (runtime *fakeDeviceRuntimeSynchronizer) RegisterDeviceLocked(device contracts.Device) error {
	runtime.registered = append(runtime.registered, device)
	return runtime.err
}
func (runtime *fakeDeviceRuntimeSynchronizer) ValidateDeviceUpdateLocked(contracts.Device) error {
	return nil
}

func (runtime *fakeDeviceRuntimeSynchronizer) UpdateDeviceLocked(device contracts.Device) error {
	runtime.updated = append(runtime.updated, device)
	return runtime.err
}
func (runtime *fakeDeviceRuntimeSynchronizer) RemoveDeviceLocked(id string) error {
	runtime.removed = append(runtime.removed, id)
	return runtime.err
}

type fakeGatewayRuntimeSynchronizer struct {
	err        error
	registered []contracts.Gateway
	updated    []contracts.Gateway
	removed    []string
}

func (runtime *fakeGatewayRuntimeSynchronizer) AcquireHardwareMutation() func() { return func() {} }
func (runtime *fakeGatewayRuntimeSynchronizer) RegisterGatewayLocked(gateway contracts.Gateway) error {
	runtime.registered = append(runtime.registered, gateway)
	return runtime.err
}
func (*fakeGatewayRuntimeSynchronizer) ValidateGatewayUpdateLocked(contracts.Gateway) error {
	return nil
}

func (runtime *fakeGatewayRuntimeSynchronizer) UpdateGatewayLocked(gateway contracts.Gateway) error {
	runtime.updated = append(runtime.updated, gateway)
	return runtime.err
}
func (runtime *fakeGatewayRuntimeSynchronizer) RemoveGatewayLocked(id string) error {
	runtime.removed = append(runtime.removed, id)
	return runtime.err
}

func TestDeviceServiceSynchronizesAndRollsBackRuntimeMutations(t *testing.T) {
	t.Run("create is rolled back when runtime rejects it", func(t *testing.T) {
		repository := &mockDeviceRepository{}
		runtime := &fakeDeviceRuntimeSynchronizer{err: errors.New("runtime unavailable")}
		service := NewDeviceService(repository)
		service.SetRuntimeSynchronizer(runtime)
		_, err := service.CreateDevice(contracts.CreateDeviceRequest{Name: "device", DevEUI: "0102030405060708"})
		if err == nil || len(repository.devices) != 0 || len(runtime.registered) != 1 {
			t.Fatalf("create rollback failed: devices=%+v runtime=%+v err=%v", repository.devices, runtime.registered, err)
		}
	})

	t.Run("update restores persisted device when runtime rejects it", func(t *testing.T) {
		original := contracts.Device{ID: testDeviceID1, Name: "original", DevEUI: "0102030405060708"}
		repository := &mockDeviceRepository{devices: []contracts.Device{original}}
		runtime := &fakeDeviceRuntimeSynchronizer{err: errors.New("runtime unavailable")}
		service := NewDeviceService(repository)
		service.SetRuntimeSynchronizer(runtime)
		updated := original
		updated.Name = "updated"
		_, err := service.UpdateDevice(contracts.UpdateDeviceRequest{ID: original.ID, Device: updated})
		if err == nil || repository.devices[0].Name != original.Name || len(runtime.updated) != 1 {
			t.Fatalf("update rollback failed: devices=%+v runtime=%+v err=%v", repository.devices, runtime.updated, err)
		}
	})

	t.Run("delete restores device when runtime rejects it", func(t *testing.T) {
		original := contracts.Device{ID: testDeviceID1, Name: "original", DevEUI: "0102030405060708"}
		repository := &mockDeviceRepository{devices: []contracts.Device{original}}
		runtime := &fakeDeviceRuntimeSynchronizer{err: errors.New("runtime unavailable")}
		service := NewDeviceService(repository)
		service.SetRuntimeSynchronizer(runtime)
		_, err := service.DeleteDevice(contracts.DeleteDeviceRequest{ID: original.ID})
		if err == nil || len(repository.devices) != 1 || len(runtime.removed) != 1 {
			t.Fatalf("delete rollback failed: devices=%+v runtime=%+v err=%v", repository.devices, runtime.removed, err)
		}
	})
}

func TestGatewayServiceSynchronizesAndRollsBackCreate(t *testing.T) {
	repository := &mockGatewayRepository{}
	runtime := &fakeGatewayRuntimeSynchronizer{err: errors.New("runtime unavailable")}
	service := NewGatewayService(repository)
	service.SetRuntimeSynchronizer(runtime)
	_, err := service.CreateGateway(validCreateGatewayRequest())
	if err == nil || len(repository.gateways) != 0 || len(runtime.registered) != 1 {
		t.Fatalf("gateway create rollback failed: gateways=%+v runtime=%+v err=%v", repository.gateways, runtime.registered, err)
	}
}

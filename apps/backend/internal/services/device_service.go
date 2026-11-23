package services

import (
	"errors"
	"fmt"
	"sync"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/google/uuid"
	"lwn-simulator-backend/internal/apperrors"
	"lwn-simulator-backend/internal/database"
)

type DeviceRepository interface {
	GetAll() ([]contracts.Device, error)
	GetByID(id string) (contracts.Device, error)
	Save(devices []contracts.Device) error
	Update(device contracts.Device) error
	Delete(device contracts.Device) error
}

type DeviceRuntimeSynchronizer interface {
	AcquireHardwareMutation() func()
	RegisterDeviceLocked(contracts.Device) error
	UpdateDeviceLocked(contracts.Device) error
	ValidateDeviceUpdateLocked(contracts.Device) error
	RemoveDeviceLocked(string) error
}

type RealtimePublisher interface {
	Publish(contracts.RealtimeWebSocketMessage)
}

type DeviceService struct {
	repository DeviceRepository
	runtime    DeviceRuntimeSynchronizer
	realtime   RealtimePublisher
	mu         sync.Mutex
}

func (s *DeviceService) SetRealtimePublisher(publisher RealtimePublisher) {
	s.mu.Lock()
	s.realtime = publisher
	s.mu.Unlock()
}

func (s *DeviceService) SetRuntimeSynchronizer(runtime DeviceRuntimeSynchronizer) {
	s.mu.Lock()
	s.runtime = runtime
	s.mu.Unlock()
}

func NewDeviceService(
	repository DeviceRepository,
) *DeviceService {
	return &DeviceService{
		repository: repository,
	}
}

func (s *DeviceService) CreateDevice(
	req contracts.CreateDeviceRequest,
) (contracts.CreateDeviceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	releaseRuntime := acquireHardwareMutation(s.runtime)
	defer releaseRuntime()

	devices, err := s.repository.GetAll()
	if err != nil {
		return contracts.CreateDeviceResponse{}, fmt.Errorf("get devices: %w", err)
	}

	for _, device := range devices {
		if device.DevEUI == req.DevEUI {
			return contracts.CreateDeviceResponse{}, apperrors.Conflict(
				"device with DevEUI %q already exists",
				req.DevEUI,
			)
		}
	}

	device := contracts.Device{
		ID:               uuid.NewString(),
		Active:           true,
		DevEUI:           req.DevEUI,
		Name:             req.Name,
		Activation:       req.Activation,
		RadioProtocol:    req.RadioProtocol,
		RadioReceiveMode: req.RadioReceiveMode,
		Class:            req.Class,
		LocationConfig:   req.LocationConfig,
		ABPConfig:        req.ABPConfig,
		OOTAConfig:       req.OOTAConfig,
		RX1Config:        req.RX1Config,
		RX2Config:        req.RX2Config,
		AdvancedConfig:   req.AdvancedConfig,
		FrameConfig:      req.FrameConfig,
		PayloadConfig:    req.PayloadConfig,
	}

	devices = append(devices, device)

	if err := s.repository.Save(devices); err != nil {
		return contracts.CreateDeviceResponse{}, fmt.Errorf("save device: %w", err)
	}
	if s.runtime != nil {
		if err := s.runtime.RegisterDeviceLocked(device); err != nil {
			rollbackErr := s.repository.Delete(device)
			return contracts.CreateDeviceResponse{}, formatRollbackError("synchronize created device", err, rollbackErr)
		}
	}

	s.publishRealtime(contracts.RealtimeDeviceCreatedMessage, &device, nil, device.ID)

	return contracts.CreateDeviceResponse{
		Device: device,
	}, nil
}

func (s *DeviceService) GetDevice(
	req contracts.GetDeviceRequest,
) (contracts.GetDeviceResponse, error) {

	device, err := s.repository.GetByID(req.ID)

	if err != nil {
		return contracts.GetDeviceResponse{}, wrapDeviceRepositoryError(
			"get device by id",
			err,
		)
	}

	return contracts.GetDeviceResponse{
		Device: device,
	}, nil
}

func (s *DeviceService) GetDevices(
	req contracts.GetDevicesRequest,
) (contracts.GetDevicesResponse, error) {
	devices, err := s.repository.GetAll()

	if err != nil {
		return contracts.GetDevicesResponse{}, fmt.Errorf("get all devices: %w", err)
	}

	return contracts.GetDevicesResponse{
		Devices: devices,
	}, nil
}

// ImportDevices replaces the devices in a newly created profile. It is kept
// separate from CreateDevice so an imported profile can preserve device IDs
// referenced by its historical simulation events.
func (s *DeviceService) ImportDevices(devices []contracts.Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	releaseRuntime := acquireHardwareMutation(s.runtime)
	defer releaseRuntime()

	seenIDs := make(map[string]struct{}, len(devices))
	seenDevEUIs := make(map[string]struct{}, len(devices))
	for _, device := range devices {
		if _, err := uuid.Parse(device.ID); err != nil {
			return apperrors.Invalid("device %q has an invalid id", device.ID)
		}
		if len(device.DevEUI) != 16 {
			return apperrors.Invalid("device %q has an invalid DevEUI", device.ID)
		}
		if _, exists := seenIDs[device.ID]; exists {
			return apperrors.Conflict("duplicate device id %q", device.ID)
		}
		if _, exists := seenDevEUIs[device.DevEUI]; exists {
			return apperrors.Conflict("duplicate device DevEUI %q", device.DevEUI)
		}
		seenIDs[device.ID] = struct{}{}
		seenDevEUIs[device.DevEUI] = struct{}{}
	}

	if err := s.repository.Save(devices); err != nil {
		return fmt.Errorf("import devices: %w", err)
	}
	if s.runtime != nil {
		for _, device := range devices {
			if err := s.runtime.RegisterDeviceLocked(device); err != nil {
				return fmt.Errorf("synchronize imported device %q: %w", device.ID, err)
			}
		}
	}
	return nil
}

func (s *DeviceService) UpdateDevice(
	req contracts.UpdateDeviceRequest,
) (contracts.UpdateDeviceResponse, error) {
	return s.updateDevice(req.ID, func(device *contracts.Device) error {
		*device = req.Device
		return nil
	})
}

func (s *DeviceService) updateDevice(
	id string,
	apply func(*contracts.Device) error,
) (contracts.UpdateDeviceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	releaseRuntime := acquireHardwareMutation(s.runtime)
	defer releaseRuntime()

	previous, err := s.repository.GetByID(id)
	if err != nil {
		return contracts.UpdateDeviceResponse{}, wrapDeviceRepositoryError(
			"get device by id",
			err,
		)
	}

	device := previous
	if err := apply(&device); err != nil {
		return contracts.UpdateDeviceResponse{}, err
	}
	device.ID = id

	devices, err := s.repository.GetAll()
	if err != nil {
		return contracts.UpdateDeviceResponse{}, fmt.Errorf(
			"get devices for update: %w",
			err,
		)
	}

	for _, existing := range devices {
		if existing.ID != id && existing.DevEUI == device.DevEUI {
			return contracts.UpdateDeviceResponse{}, apperrors.Conflict(
				"device with DevEUI %q already exists",
				device.DevEUI,
			)
		}
	}

	if s.runtime != nil {
		if err := s.runtime.ValidateDeviceUpdateLocked(device); err != nil {
			return contracts.UpdateDeviceResponse{}, err
		}
	}
	if err := s.repository.Update(device); err != nil {
		return contracts.UpdateDeviceResponse{}, wrapDeviceRepositoryError(
			"update device",
			err,
		)
	}
	if s.runtime != nil {
		if err := s.runtime.UpdateDeviceLocked(device); err != nil {
			rollbackErr := s.repository.Update(previous)
			return contracts.UpdateDeviceResponse{}, formatRollbackError("synchronize updated device", err, rollbackErr)
		}
	}
	s.publishRealtime(contracts.RealtimeDeviceUpdatedMessage, &device, nil, device.ID)

	return contracts.UpdateDeviceResponse{
		Device: device,
	}, nil
}

func (s *DeviceService) DeleteDevice(
	req contracts.DeleteDeviceRequest,
) (contracts.DeleteDeviceResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	releaseRuntime := acquireHardwareMutation(s.runtime)
	defer releaseRuntime()
	device, err := s.repository.GetByID(req.ID)

	if err != nil {
		return contracts.DeleteDeviceResponse{}, wrapDeviceRepositoryError(
			"get device by id",
			err,
		)
	}

	if err := s.repository.Delete(device); err != nil {
		return contracts.DeleteDeviceResponse{}, wrapDeviceRepositoryError(
			"delete device",
			err,
		)
	}
	if s.runtime != nil {
		if err := s.runtime.RemoveDeviceLocked(device.ID); err != nil {
			devices, readErr := s.repository.GetAll()
			if readErr != nil {
				return contracts.DeleteDeviceResponse{}, formatRollbackError("synchronize deleted device", err, readErr)
			}
			devices = append(devices, device)
			rollbackErr := s.repository.Save(devices)
			return contracts.DeleteDeviceResponse{}, formatRollbackError("synchronize deleted device", err, rollbackErr)
		}
	}
	s.publishRealtime(contracts.RealtimeDeviceDeletedMessage, nil, nil, device.ID)

	return contracts.DeleteDeviceResponse{}, nil
}

func (s *DeviceService) publishRealtime(
	messageType contracts.RealtimeWebSocketMessageType,
	device *contracts.Device,
	gateway *contracts.Gateway,
	resourceID string,
) {
	publisher := s.realtime
	if publisher == nil {
		return
	}
	publisher.Publish(contracts.RealtimeWebSocketMessage{
		Type:                  messageType,
		TimestampMilliseconds: 0,
		ResourceID:            stringPointer(resourceID),
		Device:                device,
		Gateway:               gateway,
	})
}

func stringPointer(value string) *string { return &value }

func wrapDeviceRepositoryError(operation string, err error) error {
	if errors.Is(err, database.ErrNotFound) {
		return fmt.Errorf("%w: %s: %v", apperrors.ErrNotFound, operation, err)
	}

	return fmt.Errorf("%s: %w", operation, err)
}

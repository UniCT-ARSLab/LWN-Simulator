package services

import (
	"errors"
	"fmt"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"lwn-simulator-backend/internal/apperrors"
)

const (
	defaultSimulationEventLimit = 100
	maximumSimulationEventLimit = 1000
)

// SimulationEventFilters narrows the event stream without changing the
// sequence cursor semantics. Empty fields mean that the dimension is not
// filtered.
type SimulationEventFilters struct {
	DeviceID  string
	GatewayID string
	Types     map[contracts.SimulationEventType]struct{}
}

func (s *SimulationService) SetSpeed(speed float64) (contracts.SimulationSnapshot, error) {
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	if err := runtime.SetSpeed(speed); err != nil {
		return contracts.SimulationSnapshot{}, wrapEngineCommandError("change simulation speed", err)
	}
	snapshot := runtime.Snapshot()
	s.publishSnapshot(snapshot)
	return snapshot, nil
}

func (s *SimulationService) QueueUplink(req contracts.SimulationUplinkRequest) (contracts.SimulationActionResponse, error) {
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationActionResponse{}, err
	}
	id, err := runtime.QueueUplink(req.DeviceID)
	if err != nil {
		return contracts.SimulationActionResponse{}, wrapEngineCommandError("queue uplink", err)
	}
	return contracts.SimulationActionResponse{ID: id}, nil
}

func (s *SimulationService) QueueDownlink(req contracts.SimulationDownlinkRequest) (contracts.SimulationActionResponse, error) {
	downlink, err := simulationDownlink(req)
	if err != nil {
		return contracts.SimulationActionResponse{}, err
	}
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationActionResponse{}, err
	}
	id, err := runtime.QueueDownlink(downlink)
	if err != nil {
		return contracts.SimulationActionResponse{}, wrapEngineCommandError("queue downlink", err)
	}
	return contracts.SimulationActionResponse{ID: id}, nil
}

func (s *SimulationService) QueueMACCommand(req contracts.SimulationMACCommandRequest) (contracts.SimulationActionResponse, error) {
	command, err := simulationMACCommand(req.Command)
	if err != nil {
		return contracts.SimulationActionResponse{}, err
	}
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationActionResponse{}, err
	}
	id, err := runtime.QueueMACCommand(req.DeviceID, command)
	if err != nil {
		return contracts.SimulationActionResponse{}, wrapEngineCommandError("queue MAC command", err)
	}
	return contracts.SimulationActionResponse{ID: id}, nil
}

func (s *SimulationService) Events(afterSequence int64, limit int) (contracts.SimulationEventsResponse, error) {
	return s.EventsFiltered(afterSequence, limit, SimulationEventFilters{})
}

func (s *SimulationService) EventsFiltered(afterSequence int64, limit int, filters SimulationEventFilters) (contracts.SimulationEventsResponse, error) {
	if afterSequence < 0 {
		return contracts.SimulationEventsResponse{}, apperrors.Invalid("afterSequence cannot be negative")
	}
	if limit == 0 {
		limit = defaultSimulationEventLimit
	}
	if limit < 1 || limit > maximumSimulationEventLimit {
		return contracts.SimulationEventsResponse{}, apperrors.Invalid("limit must be between 1 and %d", maximumSimulationEventLimit)
	}
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationEventsResponse{}, err
	}
	log := runtime.EventLog()
	events := make([]contracts.SimulationEvent, 0, min(limit, len(log)))
	lastSequence := afterSequence
	for _, event := range log {
		if event.Sequence <= afterSequence {
			continue
		}
		if filters.DeviceID != "" && (event.DeviceID == nil || *event.DeviceID != filters.DeviceID) {
			continue
		}
		if filters.GatewayID != "" && (event.GatewayID == nil || *event.GatewayID != filters.GatewayID) {
			continue
		}
		if len(filters.Types) > 0 {
			if _, ok := filters.Types[event.Type]; !ok {
				continue
			}
		}
		events = append(events, event)
		lastSequence = event.Sequence
		if len(events) == limit {
			break
		}
	}
	return contracts.SimulationEventsResponse{Events: events, LastSequence: lastSequence}, nil
}

func simulationDownlink(req contracts.SimulationDownlinkRequest) (types.Downlink, error) {
	var payload []byte
	if req.Payload != nil {
		payload = append([]byte(nil), (*req.Payload)...)
	}
	commands := make([]types.MACCommand, 0)
	if req.MacCommands != nil {
		commands = make([]types.MACCommand, 0, len(*req.MacCommands))
		for index, contractCommand := range *req.MacCommands {
			command, err := simulationMACCommand(contractCommand)
			if err != nil {
				return types.Downlink{}, apperrors.Invalid("macCommands[%d]: %v", index, err)
			}
			commands = append(commands, command)
		}
	}
	if len(payload) == 0 && len(commands) == 0 {
		return types.Downlink{}, apperrors.Invalid("downlink requires a payload or at least one MAC command")
	}
	downlink := types.Downlink{DeviceID: req.DeviceID, Payload: payload, DataRate: -1, MACCommands: commands}
	if req.ID != nil {
		downlink.ID = *req.ID
	}
	if req.FPort != nil {
		downlink.FPort = *req.FPort
	}
	if req.DataRate != nil {
		downlink.DataRate = *req.DataRate
	}
	if req.Confirmed != nil {
		downlink.Confirmed = *req.Confirmed
	}
	if req.FPending != nil {
		downlink.FPending = *req.FPending
	}
	if req.ACK != nil {
		downlink.ACK = *req.ACK
	}
	return downlink, nil
}

func simulationMACCommand(command contracts.SimulationMACCommand) (types.MACCommand, error) {
	if !command.Type.Valid() {
		return types.MACCommand{}, apperrors.Invalid("unsupported MAC command type %q", command.Type)
	}
	converted := types.MACCommand{
		Type:                 types.MACCommandType(command.Type),
		DataRate:             command.DataRate,
		TxPower:              command.TxPower,
		NbTrans:              command.NbTrans,
		ChannelMaskControl:   command.ChannelMaskControl,
		MaxDutyCycleExponent: command.MaxDutyCycleExponent,
		RX1DataRateOffset:    command.RX1DataRateOffset,
		Frequency:            command.Frequency,
		ChannelIndex:         command.ChannelIndex,
		MinimumDataRate:      command.MinimumDataRate,
		MaximumDataRate:      command.MaximumDataRate,
		UplinkDwellTime:      command.UplinkDwellTime,
		DownlinkDwellTime:    command.DownlinkDwellTime,
		MaximumEIRP:          command.MaximumEIRP,
		Margin:               command.Margin,
		GatewayCount:         command.GatewayCount,
		BatteryLevel:         command.BatteryLevel,
		PingSlotPeriodicity:  command.PingSlotPeriodicity,
	}
	if command.ChannelMask != nil {
		if *command.ChannelMask < 0 || *command.ChannelMask > 65535 {
			return types.MACCommand{}, apperrors.Invalid("channel mask must be between 0 and 65535")
		}
		value := uint16(*command.ChannelMask)
		converted.ChannelMask = &value
	}
	if command.DelaySeconds != nil {
		value := time.Duration(*command.DelaySeconds) * time.Second
		converted.Delay = &value
	}
	if command.DeviceTimeMilliseconds != nil {
		value := time.UnixMilli(*command.DeviceTimeMilliseconds).UTC()
		converted.DeviceTime = &value
	}
	return converted, nil
}

func wrapEngineCommandError(operation string, err error) error {
	switch {
	case errors.Is(err, engine.ErrInvalidTransition), errors.Is(err, engine.ErrEngineNotRunning), errors.Is(err, engine.ErrEngineStopped):
		return apperrors.Conflict("%s: %v", operation, err)
	case errors.Is(err, engine.ErrDeviceNotFound), errors.Is(err, engine.ErrGatewayNotFound):
		return apperrors.NotFound("%s: %v", operation, err)
	default:
		return apperrors.Invalid("%s: %v", operation, err)
	}
}

// The hardware callbacks serialize with Start so a CRUD operation is either
// included in the initial registry or applied atomically to the active one.
func (s *SimulationService) RegisterDevice(device contracts.Device) error {
	release := s.AcquireHardwareMutation()
	defer release()
	return s.RegisterDeviceLocked(device)
}

func (s *SimulationService) UpdateDevice(device contracts.Device) error {
	release := s.AcquireHardwareMutation()
	defer release()
	return s.UpdateDeviceLocked(device)
}

func (s *SimulationService) RemoveDevice(deviceID string) error {
	release := s.AcquireHardwareMutation()
	defer release()
	return s.RemoveDeviceLocked(deviceID)
}

func (s *SimulationService) RegisterGateway(gateway contracts.Gateway) error {
	release := s.AcquireHardwareMutation()
	defer release()
	return s.RegisterGatewayLocked(gateway)
}

func (s *SimulationService) UpdateGateway(gateway contracts.Gateway) error {
	release := s.AcquireHardwareMutation()
	defer release()
	return s.UpdateGatewayLocked(gateway)
}

func (s *SimulationService) RemoveGateway(gatewayID string) error {
	release := s.AcquireHardwareMutation()
	defer release()
	return s.RemoveGatewayLocked(gatewayID)
}

func (s *SimulationService) AcquireHardwareMutation() func() {
	s.operationMu.Lock()
	return s.operationMu.Unlock
}

func (s *SimulationService) RegisterDeviceLocked(device contracts.Device) error {
	return s.withHardwareRuntimeLocked(func(runtime *engine.Engine) error { return runtime.RegisterDevice(device) })
}

func (s *SimulationService) ValidateDeviceUpdateLocked(device contracts.Device) error {
	return s.withHardwareRuntimeLocked(func(runtime *engine.Engine) error { return runtime.ValidateDeviceUpdate(device) })
}

func (s *SimulationService) UpdateDeviceLocked(device contracts.Device) error {
	return s.withHardwareRuntimeLocked(func(runtime *engine.Engine) error { return runtime.UpdateDevice(device) })
}

func (s *SimulationService) RemoveDeviceLocked(deviceID string) error {
	return s.withHardwareRuntimeLocked(func(runtime *engine.Engine) error { return runtime.RemoveDevice(deviceID) })
}

func (s *SimulationService) RegisterGatewayLocked(gateway contracts.Gateway) error {
	return s.withHardwareRuntimeLocked(func(runtime *engine.Engine) error { return runtime.RegisterGateway(gateway) })
}

func (s *SimulationService) ValidateGatewayUpdateLocked(gateway contracts.Gateway) error {
	return s.withHardwareRuntimeLocked(func(runtime *engine.Engine) error { return runtime.ValidateGatewayUpdate(gateway) })
}

func (s *SimulationService) UpdateGatewayLocked(gateway contracts.Gateway) error {
	return s.withHardwareRuntimeLocked(func(runtime *engine.Engine) error { return runtime.UpdateGateway(gateway) })
}

func (s *SimulationService) RemoveGatewayLocked(gatewayID string) error {
	return s.withHardwareRuntimeLocked(func(runtime *engine.Engine) error { return runtime.RemoveGateway(gatewayID) })
}

func (s *SimulationService) withHardwareRuntimeLocked(action func(*engine.Engine) error) error {
	s.mu.RLock()
	runtime := s.engine
	s.mu.RUnlock()
	if runtime == nil {
		return nil
	}
	status := runtime.Snapshot().State.Status
	if status == contracts.SimulationStatusStopped || status == contracts.SimulationStatusFailed {
		return nil
	}
	if status != contracts.SimulationStatusRunning && status != contracts.SimulationStatusPaused {
		return apperrors.Conflict("hardware cannot change while simulation is %s", status)
	}
	if err := action(runtime); err != nil {
		return wrapEngineCommandError("synchronize simulation hardware", err)
	}
	return nil
}

func formatRollbackError(operation string, operationErr, rollbackErr error) error {
	if rollbackErr == nil {
		return operationErr
	}
	return fmt.Errorf("%s: %w; rollback failed: %v", operation, operationErr, rollbackErr)
}

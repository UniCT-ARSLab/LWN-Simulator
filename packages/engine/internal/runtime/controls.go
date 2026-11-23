package runtime

import (
	"fmt"
	"reflect"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

// SetSpeed changes the simulation clock multiplier without rebuilding the
// runtime. The scheduler observes the new value on its next wait cycle.
func (e *Engine) SetSpeed(speed float64) error {
	if err := ValidateSimulationConfig(contracts.SimulationConfig{Speed: speed}); err != nil {
		return fmt.Errorf("validate simulation speed: %w", err)
	}

	e.mu.Lock()
	if e.state.Status != contracts.SimulationStatusRunning && e.state.Status != contracts.SimulationStatusPaused {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot change speed from %q", ErrInvalidTransition, status)
	}
	if speed != 1 && requiresRealtimeTransport(e.registry.Gateways(), e.gatewayAdapterFactory) {
		e.mu.Unlock()
		return fmt.Errorf("external gateway transports require simulation speed 1x")
	}
	e.config.Speed = speed
	e.state.Speed = speed
	clock := e.clock
	e.mu.Unlock()

	if speedClock, ok := clock.(interface{ SetSpeed(float64) error }); ok {
		if err := speedClock.SetSpeed(speed); err != nil {
			return fmt.Errorf("update simulation clock speed: %w", err)
		}
	}

	e.signalWake()
	return nil
}

// QueueUplink injects a manual device transmission at the current simulation
// time. An OTAA device that is not joined sends a join request instead.
func (e *Engine) QueueUplink(deviceID string) (string, error) {
	e.mu.Lock()
	device, session, ok := e.deviceSessionLocked(deviceID)
	if !ok {
		e.mu.Unlock()
		return "", fmt.Errorf("%w: %s", ErrDeviceNotFound, deviceID)
	}
	if !device.Active {
		e.mu.Unlock()
		return "", fmt.Errorf("device %s is inactive", deviceID)
	}
	if !runtimeControlStatus(e.state.Status) {
		e.mu.Unlock()
		return "", fmt.Errorf("%w: cannot queue uplink from %q", ErrInvalidTransition, e.state.Status)
	}
	if session.PendingUplink != nil || session.PendingJoinRequest != nil {
		e.mu.Unlock()
		return "", fmt.Errorf("device %s already has a protocol transaction in progress", deviceID)
	}
	event := initialDeviceEvent(device, session, e.eventTimestampLocked())
	if err := e.scheduler.Schedule(event); err != nil {
		e.mu.Unlock()
		return "", err
	}
	logged := e.newEventLocked(contracts.DeviceUplinkScheduled, "manual device uplink scheduled", device.ID, "", event.ID)
	e.mu.Unlock()
	e.signalWake()
	e.publish(logged)
	return event.ID, nil
}

// RegisterDevice adds a device to an active runtime without rebuilding the
// simulation. Its first protocol transaction is scheduled immediately when
// the device is active.
func (e *Engine) RegisterDevice(device contracts.Device) error {
	e.mu.Lock()
	if !runtimeControlStatus(e.state.Status) {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot register device from %q", ErrInvalidTransition, status)
	}
	if _, exists := e.registry.Device(device.ID); exists {
		e.mu.Unlock()
		return fmt.Errorf("device %s is already registered", device.ID)
	}
	devices := append(e.registry.Devices(), device)
	registry, err := NewRegistry(devices, e.registry.Gateways())
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("validate registered device: %w", err)
	}
	e.registry = registry
	e.sessions[device.ID] = newDeviceSession(device)
	e.state.DeviceCount = len(devices)
	e.metrics.ActiveDevices = int64(len(registry.ActiveDevices()))

	var scheduledID string
	if device.Active {
		event := initialDeviceEvent(device, e.sessions[device.ID], e.eventTimestampLocked())
		if err := e.scheduler.Schedule(event); err != nil {
			e.mu.Unlock()
			return fmt.Errorf("schedule registered device: %w", err)
		}
		scheduledID = event.ID
	}
	logged := e.newEventLocked(contracts.DeviceRegistered, "device added to runtime registry", device.ID, "", scheduledID)
	e.mu.Unlock()
	e.signalWake()
	e.publish(logged)
	return nil
}

// RemoveDevice removes a device and all of its pending protocol and radio
// work from an active runtime.
func (e *Engine) RemoveDevice(deviceID string) error {
	e.mu.Lock()
	if !runtimeControlStatus(e.state.Status) {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot remove device from %q", ErrInvalidTransition, status)
	}
	if _, exists := e.registry.Device(deviceID); !exists {
		e.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrDeviceNotFound, deviceID)
	}
	devices := e.registry.Devices()
	filtered := make([]contracts.Device, 0, len(devices)-1)
	for _, device := range devices {
		if device.ID != deviceID {
			filtered = append(filtered, device)
		}
	}
	registry, err := NewRegistry(filtered, e.registry.Gateways())
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("validate runtime after device removal: %w", err)
	}
	e.cancelDeviceRuntimeLocked(deviceID)
	delete(e.sessions, deviceID)
	e.registry = registry
	e.state.DeviceCount = len(filtered)
	e.metrics.ActiveDevices = int64(len(registry.ActiveDevices()))
	logged := e.newEventLocked(contracts.DeviceRemoved, "device removed from runtime registry", deviceID, "", "")
	e.mu.Unlock()
	e.signalWake()
	e.publish(logged)
	return nil
}

// RegisterGateway adds a virtual or real gateway to an active runtime and
// starts its heartbeat, beacon and transport work as appropriate.
func (e *Engine) RegisterGateway(gateway contracts.Gateway) error {
	e.mu.Lock()
	if !runtimeControlStatus(e.state.Status) {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot register gateway from %q", ErrInvalidTransition, status)
	}
	if _, exists := e.registry.Gateway(gateway.ID); exists {
		e.mu.Unlock()
		return fmt.Errorf("gateway %s is already registered", gateway.ID)
	}
	gateways := append(e.registry.Gateways(), gateway)
	if e.state.Speed != 1 && requiresRealtimeTransport(gateways, e.gatewayAdapterFactory) {
		e.mu.Unlock()
		return fmt.Errorf("set simulation speed to 1x before adding an external gateway")
	}
	registry, err := NewRegistry(e.registry.Devices(), gateways)
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("validate registered gateway: %w", err)
	}
	e.registry = registry
	state := contracts.Connected
	if gateway.Type == contracts.Real || !gateway.Active {
		state = contracts.Disconnected
	}
	e.gatewayRuntime[gateway.ID] = &types.GatewayRuntime{State: state}
	e.state.GatewayCount = len(gateways)
	e.metrics.ActiveGateways = int64(len(registry.ActiveGateways()))
	if gateway.Active {
		e.scheduleGatewayRuntimeLocked(gateway)
	}
	runContext := e.runContext
	logged := e.newEventLocked(contracts.GatewayRegistered, "gateway added to runtime registry", "", gateway.ID, "")
	e.mu.Unlock()
	if gateway.Active && gateway.Type == contracts.Real && runContext != nil {
		e.startGatewayAdapter(runContext, gateway)
	}
	e.signalWake()
	e.publish(logged)
	return nil
}

// RemoveGateway stops its adapter and removes all scheduled gateway work.
func (e *Engine) RemoveGateway(gatewayID string) error {
	e.mu.Lock()
	if !runtimeControlStatus(e.state.Status) {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot remove gateway from %q", ErrInvalidTransition, status)
	}
	if _, exists := e.registry.Gateway(gatewayID); !exists {
		e.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrGatewayNotFound, gatewayID)
	}
	gateways := e.registry.Gateways()
	filtered := make([]contracts.Gateway, 0, len(gateways)-1)
	for _, gateway := range gateways {
		if gateway.ID != gatewayID {
			filtered = append(filtered, gateway)
		}
	}
	registry, err := NewRegistry(e.registry.Devices(), filtered)
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("validate runtime after gateway removal: %w", err)
	}
	e.cancelGatewayRuntimeLocked(gatewayID)
	adapter := e.gatewayAdapters[gatewayID]
	delete(e.gatewayAdapters, gatewayID)
	delete(e.gatewayRuntime, gatewayID)
	e.registry = registry
	e.state.GatewayCount = len(filtered)
	e.metrics.ActiveGateways = int64(len(registry.ActiveGateways()))
	logged := e.newEventLocked(contracts.GatewayRemoved, "gateway removed from runtime registry", "", gatewayID, "")
	e.mu.Unlock()
	if adapter != nil {
		_ = adapter.Close()
	}
	e.signalWake()
	e.publish(logged)
	return nil
}

// UpdateDevice replaces a registered device while preserving its protocol
// session when identity and activation credentials have not changed.
func (e *Engine) UpdateDevice(updated contracts.Device) error {
	e.mu.Lock()
	current, exists := e.registry.Device(updated.ID)
	if !exists {
		e.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrDeviceNotFound, updated.ID)
	}
	if err := validateRunningDeviceUpdate(current, updated, e.state.Status); err != nil {
		e.mu.Unlock()
		return err
	}
	devices := e.registry.Devices()
	for index := range devices {
		if devices[index].ID == updated.ID {
			devices[index] = updated
			break
		}
	}
	registry, err := NewRegistry(devices, e.registry.Gateways())
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("validate updated device: %w", err)
	}
	identityChanged := deviceIdentityChanged(current, updated)
	runtimeChanged := identityChanged || current.Class != updated.Class
	if pending := e.sessions[updated.ID].PendingUplink; pending != nil && pending.PayloadConfig == nil {
		payloadConfig := current.PayloadConfig
		pending.PayloadConfig = &payloadConfig
	}
	e.registry = registry
	e.metrics.ActiveDevices = int64(len(registry.ActiveDevices()))
	if !updated.Active || runtimeChanged {
		e.cancelDeviceRuntimeLocked(updated.ID)
	}
	if identityChanged {
		e.sessions[updated.ID] = newDeviceSession(updated)
	}
	var scheduled *types.ScheduledEvent
	if updated.Active && (!current.Active || runtimeChanged) && runtimeControlStatus(e.state.Status) {
		event := initialDeviceEvent(updated, e.sessions[updated.ID], e.eventTimestampLocked())
		if err := e.scheduler.Schedule(event); err != nil {
			e.mu.Unlock()
			return fmt.Errorf("schedule updated device: %w", err)
		}
		scheduled = &event
	}
	logged := e.newEventLocked(contracts.DeviceRegistered, "device runtime configuration updated", updated.ID, "", "")
	if scheduled != nil {
		logged.ScheduledEventID = &scheduled.ID
	}
	e.mu.Unlock()
	e.signalWake()
	e.publish(logged)
	return nil
}

// UpdateGateway replaces a gateway configuration. Virtual gateway schedules
// are rebuilt immediately; real adapters are restarted when their endpoint or
// active state changes.
func (e *Engine) UpdateGateway(updated contracts.Gateway) error {
	e.mu.Lock()
	current, exists := e.registry.Gateway(updated.ID)
	if !exists {
		e.mu.Unlock()
		return fmt.Errorf("%w: %s", ErrGatewayNotFound, updated.ID)
	}
	if err := validateRunningGatewayUpdate(current, updated, e.state.Status); err != nil {
		e.mu.Unlock()
		return err
	}
	gateways := e.registry.Gateways()
	for index := range gateways {
		if gateways[index].ID == updated.ID {
			gateways[index] = updated
			break
		}
	}
	if e.state.Speed != 1 && requiresRealtimeTransport(gateways, e.gatewayAdapterFactory) {
		e.mu.Unlock()
		return fmt.Errorf("set simulation speed to 1x before configuring an external gateway")
	}
	registry, err := NewRegistry(e.registry.Devices(), gateways)
	if err != nil {
		e.mu.Unlock()
		return fmt.Errorf("validate updated gateway: %w", err)
	}
	restart := current.Active != updated.Active || current.Type != updated.Type ||
		!reflect.DeepEqual(current.GatewayIPv4, updated.GatewayIPv4) ||
		!reflect.DeepEqual(current.GatewayPort, updated.GatewayPort) ||
		!reflect.DeepEqual(current.KeepAlive, updated.KeepAlive)
	e.registry = registry
	e.metrics.ActiveGateways = int64(len(registry.ActiveGateways()))
	var adapter types.GatewayAdapter
	if restart {
		e.cancelGatewayRuntimeLocked(updated.ID)
		adapter = e.gatewayAdapters[updated.ID]
		delete(e.gatewayAdapters, updated.ID)
	}
	if runtimeState := e.gatewayRuntime[updated.ID]; restart && runtimeState != nil {
		if !updated.Active || updated.Type == contracts.Real {
			runtimeState.State = contracts.Disconnected
		} else {
			runtimeState.State = contracts.Connected
		}
	}
	runContext := e.runContext
	if restart && updated.Active && runtimeControlStatus(e.state.Status) {
		e.scheduleGatewayRuntimeLocked(updated)
	}
	logged := e.newEventLocked(contracts.GatewayRegistered, "gateway runtime configuration updated", "", updated.ID, "")
	e.mu.Unlock()
	if adapter != nil {
		_ = adapter.Close()
	}
	if restart && updated.Active && runContext != nil && requiresRealtimeTransport([]contracts.Gateway{updated}, e.gatewayAdapterFactory) {
		e.startGatewayAdapter(runContext, updated)
	}
	e.signalWake()
	e.publish(logged)
	return nil
}

func runtimeControlStatus(status contracts.SimulationStatus) bool {
	return status == contracts.SimulationStatusRunning || status == contracts.SimulationStatusPaused
}

func initialDeviceEvent(device contracts.Device, session *types.DeviceSession, at time.Duration) types.ScheduledEvent {
	if deviceRadioProtocol(device) == contracts.RawLoRa {
		return types.ScheduledEvent{ID: uuid.NewString(), At: at, Type: contracts.DeviceUplinkTransmitted, Message: "device raw LoRa frame scheduled", DeviceID: device.ID, Kind: types.ScheduledEventDeviceUplink, Attempt: 1}
	}
	kind := types.ScheduledEventDeviceUplink
	eventType := contracts.DeviceUplinkTransmitted
	message := "device uplink scheduled"
	if session == nil || !session.Joined {
		kind = types.ScheduledEventJoinRequest
		eventType = contracts.DeviceJoinRequestTransmitted
		message = "device join request scheduled"
	}
	return types.ScheduledEvent{ID: uuid.NewString(), At: at, Type: eventType, Message: message, DeviceID: device.ID, Kind: kind, Attempt: 1}
}

func (e *Engine) cancelDeviceRuntimeLocked(deviceID string) {
	for _, scheduled := range e.scheduler.Events() {
		if scheduled.DeviceID == deviceID {
			e.scheduler.Cancel(scheduled.ID)
		}
	}
	for packetID, transmission := range e.radio {
		if transmission.DeviceID == deviceID {
			delete(e.radio, packetID)
		}
	}
	if session := e.sessions[deviceID]; session != nil {
		session.PendingUplink = nil
		session.PendingJoinRequest = nil
		session.PendingClassADownlinks = nil
		session.PendingClassBDownlinks = nil
		session.PendingClassCDownlinks = nil
	}
}

func (e *Engine) cancelGatewayRuntimeLocked(gatewayID string) {
	for _, scheduled := range e.scheduler.Events() {
		if scheduled.GatewayID == gatewayID {
			e.scheduler.Cancel(scheduled.ID)
		}
	}
	for _, session := range e.sessions {
		if session == nil {
			continue
		}
		if session.PendingUplink != nil {
			session.PendingUplink.GatewayIDs = removeGatewayID(session.PendingUplink.GatewayIDs, gatewayID)
		}
		if session.PendingJoinRequest != nil {
			session.PendingJoinRequest.GatewayIDs = removeGatewayID(session.PendingJoinRequest.GatewayIDs, gatewayID)
		}
		if session.ClassBGatewayID != gatewayID {
			continue
		}
		session.ClassBSynchronized = false
		session.ClassBGatewayID = ""
		session.ClassBNextPingEventID = ""
		session.ClassBBeaconTimeoutID = ""
	}
	for _, transmission := range e.radio {
		if transmission != nil {
			transmission.GatewayIDs = removeGatewayID(transmission.GatewayIDs, gatewayID)
		}
	}
}

func removeGatewayID(gatewayIDs []string, removed string) []string {
	filtered := gatewayIDs[:0]
	for _, gatewayID := range gatewayIDs {
		if gatewayID != removed {
			filtered = append(filtered, gatewayID)
		}
	}
	return filtered
}

func (e *Engine) scheduleGatewayRuntimeLocked(gateway contracts.Gateway) {
	e.scheduleClassBBeaconLocked(gateway.ID, e.eventTimestampLocked()+classBRealGatewayInitialDelay)
	if gateway.Type != contracts.Virtual || gateway.KeepAlive == nil {
		return
	}
	_ = e.scheduler.Schedule(types.ScheduledEvent{
		ID: uuid.NewString(), At: e.eventTimestampLocked() + time.Duration(*gateway.KeepAlive)*time.Second,
		Type: contracts.GatewayHeartbeat, Message: "virtual gateway heartbeat scheduled",
		GatewayID: gateway.ID, Kind: types.ScheduledEventGatewayHeartbeat,
	})
}

func deviceIdentityChanged(left, right contracts.Device) bool {
	return left.DevEUI != right.DevEUI || left.Activation != right.Activation ||
		!reflect.DeepEqual(left.OOTAConfig, right.OOTAConfig) || !reflect.DeepEqual(left.ABPConfig, right.ABPConfig)
}

// ValidateDeviceUpdate checks a hardware edit before it is persisted. Callers
// serialize validation and update with other hardware mutations.
func (e *Engine) ValidateDeviceUpdate(updated contracts.Device) error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	current, exists := e.registry.Device(updated.ID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrDeviceNotFound, updated.ID)
	}
	return validateRunningDeviceUpdate(current, updated, e.state.Status)
}

func validateRunningDeviceUpdate(current, updated contracts.Device, status contracts.SimulationStatus) error {
	if !runtimeControlStatus(status) {
		return nil
	}
	allowed := current
	allowed.Active = updated.Active
	allowed.LocationConfig.Latitude = updated.LocationConfig.Latitude
	allowed.LocationConfig.Longitude = updated.LocationConfig.Longitude
	allowed.LocationConfig.Altitude = updated.LocationConfig.Altitude
	allowed.PayloadConfig.Payload = updated.PayloadConfig.Payload
	allowed.PayloadConfig.Base64Encoded = updated.PayloadConfig.Base64Encoded
	if !reflect.DeepEqual(allowed, updated) {
		return fmt.Errorf("only active, position, payload and Base64 encoding can change while a simulation is running or paused")
	}
	return nil
}

// ValidateGatewayUpdate checks a hardware edit before it is persisted.
func (e *Engine) ValidateGatewayUpdate(updated contracts.Gateway) error {
	e.mu.RLock()
	defer e.mu.RUnlock()
	current, exists := e.registry.Gateway(updated.ID)
	if !exists {
		return fmt.Errorf("%w: %s", ErrGatewayNotFound, updated.ID)
	}
	return validateRunningGatewayUpdate(current, updated, e.state.Status)
}

func validateRunningGatewayUpdate(current, updated contracts.Gateway, status contracts.SimulationStatus) error {
	if !runtimeControlStatus(status) {
		return nil
	}
	allowed := current
	allowed.Active = updated.Active
	allowed.Latitude = updated.Latitude
	allowed.Longitude = updated.Longitude
	allowed.Altitude = updated.Altitude
	if !reflect.DeepEqual(allowed, updated) {
		return fmt.Errorf("only active and position can change while a simulation is running or paused")
	}
	return nil
}

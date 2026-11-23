package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

const runtimeEventLogLimit = 10_000

type Engine struct {
	mu sync.RWMutex

	config                contracts.SimulationConfig
	registry              *Registry
	clock                 types.Clock
	sessions              map[string]*types.DeviceSession
	radio                 map[string]*types.RadioTransmission
	rng                   *rand.Rand
	gatewayAdapterFactory types.GatewayAdapterFactory
	gatewayAdapters       map[string]types.GatewayAdapter
	gatewayRuntime        map[string]*types.GatewayRuntime
	gatewayPackets        []types.GatewayPacket

	scheduler *Scheduler
	events    chan contracts.SimulationEvent
	eventLog  []contracts.SimulationEvent
	sink      types.EventSink

	state   contracts.SimulationState
	metrics contracts.SimulationMetrics

	runCancel    context.CancelFunc
	runContext   context.Context
	done         chan struct{}
	wake         chan struct{}
	gatewayWG    sync.WaitGroup
	processingAt time.Duration
	processing   bool
	restored     bool
}

func New(
	config contracts.SimulationConfig,
	devices []contracts.Device,
	gateways []contracts.Gateway,
	options types.Options,
) (*Engine, error) {
	if err := ValidateSimulationConfig(config); err != nil {
		return nil, fmt.Errorf("validate simulation config: %w", err)
	}

	if options.Checkpoint != nil {
		if len(options.Checkpoint.Devices) > 0 {
			devices = append([]contracts.Device(nil), options.Checkpoint.Devices...)
		}
		if len(options.Checkpoint.Gateways) > 0 {
			gateways = append([]contracts.Gateway(nil), options.Checkpoint.Gateways...)
		}
	}
	registry, err := NewRegistry(devices, gateways)
	if err != nil {
		return nil, fmt.Errorf("validate runtime registry: %w", err)
	}

	if requiresRealtimeTransport(registry.Gateways(), options.GatewayAdapterFactory) {
		config.Speed = 1
	}

	clock := options.Clock
	if clock == nil {
		clock = types.NewRealClock(config.Speed)
	}
	if requiresRealtimeTransport(registry.Gateways(), options.GatewayAdapterFactory) {
		if speedClock, ok := clock.(interface{ SetSpeed(float64) error }); ok {
			if err := speedClock.SetSpeed(1); err != nil {
				return nil, fmt.Errorf("set external transport clock speed: %w", err)
			}
		}
	}

	bufferSize := options.EventBuffer
	if bufferSize <= 0 {
		bufferSize = 256
	}

	snapshot := registry.Snapshot()
	seed := int64(1)
	if config.Seed != nil {
		seed = *config.Seed
	}
	sessions := make(map[string]*types.DeviceSession, len(devices))
	gatewayRuntime := make(map[string]*types.GatewayRuntime, len(gateways))
	externalVirtualTransport := false
	if factory, ok := options.GatewayAdapterFactory.(types.VirtualGatewayAdapterFactory); ok {
		externalVirtualTransport = factory.SupportsVirtualGateways()
	}
	for _, device := range registry.Devices() {
		session := newDeviceSession(device)
		if externalVirtualTransport && device.OOTAConfig != nil {
			identifier := uuid.New()
			session.DevNonce = binary.BigEndian.Uint16(identifier[:2])
		}
		sessions[device.ID] = session
	}
	for _, gateway := range registry.Gateways() {
		state := contracts.Connected
		if gateway.Type == contracts.Real {
			state = contracts.Disconnected
		}
		gatewayRuntime[gateway.ID] = &types.GatewayRuntime{State: state}
	}
	runtime := &Engine{
		config:                config,
		registry:              registry,
		clock:                 clock,
		sessions:              sessions,
		radio:                 make(map[string]*types.RadioTransmission),
		rng:                   rand.New(rand.NewSource(seed)),
		gatewayAdapterFactory: options.GatewayAdapterFactory,
		gatewayAdapters:       make(map[string]types.GatewayAdapter),
		gatewayRuntime:        gatewayRuntime,
		scheduler:             NewScheduler(),
		events:                make(chan contracts.SimulationEvent, bufferSize),
		eventLog:              make([]contracts.SimulationEvent, 0, bufferSize),
		sink:                  options.EventSink,
		state: contracts.SimulationState{
			Status:              contracts.SimulationStatusIdle,
			Speed:               config.Speed,
			Seed:                config.Seed,
			ElapsedMilliseconds: 0,
			EventSequence:       0,
			DeviceCount:         len(snapshot.Devices),
			GatewayCount:        len(snapshot.Gateways),
		},
		metrics: contracts.SimulationMetrics{
			ActiveDevices:  int64(len(registry.ActiveDevices())),
			ActiveGateways: int64(len(registry.ActiveGateways())),
		},
		wake: make(chan struct{}, 1),
	}
	if options.Checkpoint != nil {
		if err := runtime.restoreCheckpoint(options.Checkpoint); err != nil {
			return nil, fmt.Errorf("restore simulation checkpoint: %w", err)
		}
	}
	return runtime, nil
}

func (e *Engine) restoreCheckpoint(checkpoint *types.EngineCheckpoint) error {
	if checkpoint == nil {
		return nil
	}
	if checkpoint.Config.Speed > 0 {
		e.config = checkpoint.Config
		if requiresRealtimeTransport(e.registry.Gateways(), e.gatewayAdapterFactory) {
			e.config.Speed = 1
		}
		e.state.Speed = e.config.Speed
		e.state.Seed = checkpoint.Config.Seed
	}
	if checkpoint.State.EventSequence > 0 || checkpoint.State.ElapsedMilliseconds > 0 {
		e.state.EventSequence = checkpoint.State.EventSequence
		e.state.ElapsedMilliseconds = checkpoint.State.ElapsedMilliseconds
	}
	e.metrics = checkpoint.Metrics
	e.state.DeviceCount = len(e.registry.Devices())
	e.state.GatewayCount = len(e.registry.Gateways())
	e.metrics.ActiveDevices = int64(len(e.registry.ActiveDevices()))
	e.metrics.ActiveGateways = int64(len(e.registry.ActiveGateways()))
	e.eventLog = recentSimulationEvents(checkpoint.EventLog, runtimeEventLogLimit)
	for deviceID, session := range checkpoint.Sessions {
		if _, ok := e.sessions[deviceID]; ok {
			copySession := session
			if session.PendingUplink != nil {
				pending := *session.PendingUplink
				pending.GatewayIDs = append([]string(nil), pending.GatewayIDs...)
				pending.PayloadConfig = cloneRuntimePointer(pending.PayloadConfig)
				pending.WirePayload = append([]byte(nil), pending.WirePayload...)
				copySession.PendingUplink = &pending
			}
			if session.PendingJoinRequest != nil {
				pending := *session.PendingJoinRequest
				pending.GatewayIDs = append([]string(nil), pending.GatewayIDs...)
				copySession.PendingJoinRequest = &pending
			}
			copySession.PendingClassBDownlinks = cloneClassBDownlinks(session.PendingClassBDownlinks)
			copySession.PendingClassADownlinks = cloneDownlinks(session.PendingClassADownlinks)
			copySession.PendingClassCDownlinks = cloneDownlinks(session.PendingClassCDownlinks)
			copySession.AdditionalChannels = append([]types.RadioChannel(nil), session.AdditionalChannels...)
			copySession.PendingMACResponses = cloneMACResponses(session.PendingMACResponses)
			copySession.NetworkChannels = cloneSessionMap(session.NetworkChannels)
			copySession.DisabledChannels = cloneSessionMap(session.DisabledChannels)
			copySession.NwkSKey = append([]byte(nil), session.NwkSKey...)
			copySession.AppSKey = append([]byte(nil), session.AppSKey...)
			e.sessions[deviceID] = &copySession
		}
	}
	for packetID, transmission := range checkpoint.Radio {
		copyTransmission := transmission
		copyTransmission.GatewayIDs = append([]string(nil), transmission.GatewayIDs...)
		copyTransmission.Payload = append([]byte(nil), transmission.Payload...)
		e.radio[packetID] = &copyTransmission
	}
	for _, packet := range checkpoint.GatewayPackets {
		copyPacket := packet
		copyPacket.Payload = append([]byte(nil), packet.Payload...)
		e.gatewayPackets = append(e.gatewayPackets, copyPacket)
	}
	for _, scheduled := range checkpoint.Scheduled {
		if err := e.scheduler.Schedule(scheduled); err != nil {
			return err
		}
	}
	if realClock, ok := e.clock.(interface{ SetElapsed(time.Duration) }); ok {
		realClock.SetElapsed(time.Duration(checkpoint.State.ElapsedMilliseconds) * time.Millisecond)
	}
	e.restored = true
	return nil
}

func (e *Engine) Start(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("start context cannot be nil")
	}

	e.mu.Lock()
	if e.state.Status != contracts.SimulationStatusIdle {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot start from %q", ErrInvalidTransition, status)
	}
	if lifecycleClock, ok := e.clock.(types.LifecycleClock); ok {
		lifecycleClock.Start()
	}

	runCtx, cancel := context.WithCancel(ctx)
	e.runCancel = cancel
	e.runContext = runCtx
	e.done = make(chan struct{})
	e.state.Status = contracts.SimulationStatusRunning
	events := []contracts.SimulationEvent{
		e.newEventLocked(
			contracts.SimulationStarted,
			"simulation started",
			"",
			"",
			"",
		),
	}
	for _, device := range e.registry.Devices() {
		events = append(events, e.newEventLocked(
			contracts.DeviceRegistered,
			"device registered in runtime registry",
			device.ID,
			"",
			"",
		))
	}
	for _, gateway := range e.registry.Gateways() {
		events = append(events, e.newEventLocked(
			contracts.GatewayRegistered,
			"gateway registered in runtime registry",
			"",
			gateway.ID,
			"",
		))
	}
	if !e.restored {
		activeDevices := e.registry.ActiveDevices()
		for _, gateway := range e.registry.ActiveVirtualGateways() {
			beacon := e.scheduleClassBBeaconLocked(gateway.ID, 0)
			events = append(events, beacon)
		}
		for _, gateway := range e.registry.Gateways() {
			if !gateway.Active || gateway.Type != contracts.Real {
				continue
			}
			beacon := e.scheduleClassBBeaconLocked(gateway.ID, classBRealGatewayInitialDelay)
			events = append(events, beacon)
		}
		for _, gateway := range e.registry.ActiveVirtualGateways() {
			if gateway.KeepAlive == nil {
				continue
			}
			heartbeat := types.ScheduledEvent{
				ID: uuid.NewString(), At: time.Duration(*gateway.KeepAlive) * time.Second,
				Type: contracts.GatewayHeartbeat, Message: "virtual gateway heartbeat scheduled",
				GatewayID: gateway.ID, Kind: types.ScheduledEventGatewayHeartbeat,
			}
			if err := e.scheduler.Schedule(heartbeat); err != nil {
				e.state.Status = contracts.SimulationStatusFailed
				e.mu.Unlock()
				cancel()
				return fmt.Errorf("schedule heartbeat for gateway %s: %w", gateway.ID, err)
			}
		}
		for _, device := range activeDevices {
			session := e.sessions[device.ID]
			scheduled := initialDeviceEvent(device, session, e.initialTransmissionDelay(device, len(activeDevices)))
			scheduledLogType := contracts.DeviceUplinkScheduled
			if scheduled.Kind == types.ScheduledEventJoinRequest {
				scheduledLogType = contracts.DeviceJoinRequestScheduled
			}
			if err := e.scheduler.Schedule(scheduled); err != nil {
				e.state.Status = contracts.SimulationStatusFailed
				e.mu.Unlock()
				cancel()
				return fmt.Errorf("schedule initial uplink for device %s: %w", device.ID, err)
			}
			events = append(events, e.newEventLocked(
				scheduledLogType,
				scheduled.Message,
				device.ID,
				"",
				scheduled.ID,
			))
		}
	}
	e.mu.Unlock()

	for _, event := range events {
		e.publish(event)
	}
	e.startGatewayAdapters(runCtx)
	go e.run(runCtx)
	return nil
}

// initialTransmissionDelay gives every device a deterministic phase within
// its reporting period. Real fleets are not powered on on the same radio
// tick; scheduling every first frame at t=0 creates an artificial collision
// storm that gets worse as the topology grows. The seeded engine RNG keeps
// repeated simulations reproducible while distributing the initial load.
func (e *Engine) initialTransmissionDelay(device contracts.Device, activeDeviceCount int) time.Duration {
	if activeDeviceCount <= 1 {
		return 0
	}
	interval := durationSecondsFloat(device.PayloadConfig.UplinkInterval)
	if interval <= 0 {
		return 0
	}
	return time.Duration(e.rng.Int63n(int64(interval)))
}

func (e *Engine) Pause() error {
	e.mu.Lock()
	if e.state.Status != contracts.SimulationStatusRunning {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot pause from %q", ErrInvalidTransition, status)
	}
	e.state.Status = contracts.SimulationStatusPaused
	if lifecycleClock, ok := e.clock.(types.LifecycleClock); ok {
		lifecycleClock.Pause()
	}
	event := e.newEventLocked(
		contracts.SimulationPaused,
		"simulation paused",
		"",
		"",
		"",
	)
	e.mu.Unlock()

	e.signalWake()
	e.publish(event)
	return nil
}

func (e *Engine) Resume() error {
	e.mu.Lock()
	if e.state.Status != contracts.SimulationStatusPaused {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot resume from %q", ErrInvalidTransition, status)
	}
	e.state.Status = contracts.SimulationStatusRunning
	if lifecycleClock, ok := e.clock.(types.LifecycleClock); ok {
		lifecycleClock.Resume()
	}
	event := e.newEventLocked(
		contracts.SimulationResumed,
		"simulation resumed",
		"",
		"",
		"",
	)
	e.mu.Unlock()

	e.signalWake()
	e.publish(event)
	return nil
}

func (e *Engine) Stop(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("stop context cannot be nil")
	}

	e.mu.Lock()
	if e.state.Status == contracts.SimulationStatusStopped {
		e.mu.Unlock()
		return nil
	}
	if e.state.Status != contracts.SimulationStatusRunning && e.state.Status != contracts.SimulationStatusPaused {
		status := e.state.Status
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot stop from %q", ErrInvalidTransition, status)
	}
	e.state.Status = contracts.SimulationStatusStopping
	cancel := e.runCancel
	done := e.done
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.signalWake()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Engine) Schedule(event types.ScheduledEvent) error {
	e.mu.Lock()
	status := e.state.Status
	if status != contracts.SimulationStatusRunning && status != contracts.SimulationStatusPaused {
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot schedule from %q", ErrInvalidTransition, status)
	}
	if err := e.scheduler.Schedule(event); err != nil {
		e.mu.Unlock()
		return err
	}
	logEvent := e.newEventLocked(
		contracts.SchedulerEventScheduled,
		"scheduled simulation event",
		event.DeviceID,
		event.GatewayID,
		event.ID,
	)
	e.mu.Unlock()

	e.signalWake()
	e.publish(logEvent)
	return nil
}

func (e *Engine) Cancel(eventID string) error {
	e.mu.Lock()
	status := e.state.Status
	if status != contracts.SimulationStatusRunning && status != contracts.SimulationStatusPaused {
		e.mu.Unlock()
		return fmt.Errorf("%w: cannot cancel from %q", ErrInvalidTransition, status)
	}
	if _, ok := e.scheduler.Cancel(eventID); !ok {
		e.mu.Unlock()
		return ErrEventNotFound
	}
	logEvent := e.newEventLocked(
		contracts.SchedulerEventCancelled,
		"cancelled scheduled simulation event",
		"",
		"",
		eventID,
	)
	e.mu.Unlock()

	e.signalWake()
	e.publish(logEvent)
	return nil
}

// QueueClassBDownlink queues an application downlink for the device's next
// available Class B ping slot. The payload remains queued while the device is
// acquiring or recovering beacon synchronization.
func (e *Engine) QueueClassBDownlink(downlink types.ClassBDownlink) (string, error) {
	return e.QueueDownlink(downlink)
}

// QueueMACCommand delivers a semantic LoRaWAN MAC command through the same
// class-aware downlink path used by application payloads.
func (e *Engine) QueueMACCommand(deviceID string, command types.MACCommand) (string, error) {
	return e.QueueDownlink(types.Downlink{DeviceID: deviceID, DataRate: -1, MACCommands: []types.MACCommand{command}})
}

// QueueDownlink queues an application downlink according to the device class.
// Class A uses the next receive window, Class B uses the next ping slot and
// Class C is delivered as soon as a gateway path is available.
func (e *Engine) QueueDownlink(downlink types.Downlink) (string, error) {
	if downlink.DeviceID == "" {
		return "", fmt.Errorf("downlink device id is required")
	}
	if len(downlink.Payload) == 0 && len(downlink.MACCommands) == 0 {
		return "", fmt.Errorf("downlink must contain an application payload or at least one MAC command")
	}

	e.mu.Lock()

	device, session, ok := e.deviceSessionLocked(downlink.DeviceID)
	if !ok {
		e.mu.Unlock()
		return "", fmt.Errorf("%w: %s", ErrDeviceNotFound, downlink.DeviceID)
	}
	if device.Class != contracts.ClassA && device.Class != contracts.ClassB && device.Class != contracts.ClassC {
		e.mu.Unlock()
		return "", fmt.Errorf("device %s has an unsupported class", downlink.DeviceID)
	}
	if deviceRadioProtocol(device) == contracts.RawLoRa && len(downlink.MACCommands) > 0 {
		e.mu.Unlock()
		return "", fmt.Errorf("raw LoRa downlinks cannot contain LoRaWAN MAC commands")
	}
	if !device.Active {
		e.mu.Unlock()
		return "", fmt.Errorf("device %s is inactive", downlink.DeviceID)
	}
	if e.state.Status != contracts.SimulationStatusRunning && e.state.Status != contracts.SimulationStatusPaused {
		e.mu.Unlock()
		return "", fmt.Errorf("%w: cannot queue Class B downlink from %q", ErrInvalidTransition, e.state.Status)
	}
	if deviceRadioProtocol(device) == contracts.RawLoRa {
		downlink.FPort = 0
		downlink.Confirmed = false
		downlink.ACK = false
		downlink.FPending = false
	} else {
		if downlink.FPort == 0 && len(downlink.Payload) > 0 {
			downlink.FPort = device.FrameConfig.FPort
		}
		if len(downlink.Payload) > 0 && (downlink.FPort < 1 || downlink.FPort > 223) {
			e.mu.Unlock()
			return "", fmt.Errorf("downlink FPort must be between 1 and 223")
		}
	}
	for index, command := range downlink.MACCommands {
		if err := validateMACCommand(device.LocationConfig.Region, command); err != nil {
			e.mu.Unlock()
			return "", fmt.Errorf("validate MAC command %d: %w", index, err)
		}
	}
	if _, err := lorawanHandler.EncodeMACCommands(downlink.MACCommands); err != nil {
		e.mu.Unlock()
		return "", fmt.Errorf("encode MAC commands: %w", err)
	}
	plan := regional.MustPlan(device.LocationConfig.Region)
	_, downlinkDataRateSupported := plan.DataRates[downlink.DataRate]
	if downlink.DataRate < -1 || (downlink.DataRate >= 0 && !downlinkDataRateSupported) {
		e.mu.Unlock()
		return "", fmt.Errorf("downlink data rate DR%d is not supported by %s", downlink.DataRate, device.LocationConfig.Region)
	}
	if downlink.ID != "" {
		if _, err := uuid.Parse(downlink.ID); err != nil {
			e.mu.Unlock()
			return "", fmt.Errorf("downlink id must be a valid UUID: %w", err)
		}
	}
	dataRate := downlink.DataRate
	if dataRate < 0 {
		dataRate = currentDeviceDataRate(session, device)
	}
	if len(downlink.Payload) > maximumPayloadSize(device.LocationConfig.Region, dataRate) {
		e.mu.Unlock()
		return "", fmt.Errorf("downlink payload exceeds the regional maximum for data rate %d", dataRate)
	}
	if downlink.ID == "" {
		downlink.ID = uuid.NewString()
	}
	downlink.Payload = append([]byte(nil), downlink.Payload...)
	downlink.MACCommands = cloneMACCommands(downlink.MACCommands)
	eventType := contracts.DeviceDownlinkScheduled
	message := fmt.Sprintf("Class %s downlink queued", device.Class)
	switch device.Class {
	case contracts.ClassA:
		session.PendingClassADownlinks = append(session.PendingClassADownlinks, downlink)
	case contracts.ClassB:
		session.PendingClassBDownlinks = append(session.PendingClassBDownlinks, downlink)
		eventType = contracts.ClassBDownlinkScheduled
		message = "Class B downlink queued for the next ping slot"
	case contracts.ClassC:
		session.PendingClassCDownlinks = append(session.PendingClassCDownlinks, downlink)
	}
	event := e.newDownlinkEventLocked(eventType, message, device, session, downlink, "")

	// The normal ping-slot chain is created by beacon synchronization. If a
	// caller queues after synchronization but the chain was interrupted, make
	// sure the next slot is restored immediately.
	if device.Class == contracts.ClassB && session.ClassBSynchronized && session.ClassBNextPingEventID == "" {
		scheduleEvent := e.scheduleClassBPingSlotLocked(device, session, e.eventTimestampLocked()+classBPingSlotPeriod(session))
		if scheduleEvent != nil {
			deferredScheduleEvent := *scheduleEvent
			e.mu.Unlock()
			e.signalWake()
			e.publish(deferredScheduleEvent)
			e.publish(event)
			return downlink.ID, nil
		}
	}
	if device.Class == contracts.ClassC && session.ClassCNextDownlinkEventID == "" {
		if scheduled := e.scheduleClassCDownlinkLocked(device, session, e.eventTimestampLocked()); scheduled != nil {
			deferredScheduleEvent := *scheduled
			e.mu.Unlock()
			e.signalWake()
			e.publish(deferredScheduleEvent)
			e.publish(event)
			return downlink.ID, nil
		}
	}
	e.mu.Unlock()
	e.publish(event)
	return downlink.ID, nil
}

// StateAndMetrics returns the current counters without building the hardware view.
func (e *Engine) StateAndMetrics() (contracts.SimulationState, contracts.SimulationMetrics) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentStateLocked(), e.metrics
}

func (e *Engine) currentStateLocked() contracts.SimulationState {
	state := e.state
	if elapsed := e.clock.Now().Milliseconds(); elapsed > state.ElapsedMilliseconds {
		state.ElapsedMilliseconds = elapsed
	}
	return state
}

func (e *Engine) Snapshot() contracts.SimulationSnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()

	snapshot := e.registry.Snapshot()
	snapshot.State = e.currentStateLocked()
	snapshot.Metrics = e.metrics
	for index, device := range e.registry.Devices() {
		if session := e.sessions[device.ID]; session != nil {
			snapshot.Devices[index].Joined = session.Joined
			snapshot.Devices[index].FrameCounterUp = session.FrameCounterUp
			snapshot.Devices[index].FrameCounterDown = session.FrameCounterDown
			snapshot.Devices[index].PendingConfirmedUplink = session.PendingUplink != nil
			snapshot.Devices[index].LastRSSI = session.LastRSSI
			snapshot.Devices[index].LastSNR = session.LastSNR
			snapshot.Devices[index].LastAirtimeMilliseconds = session.LastAirtime.Milliseconds()
			snapshot.Devices[index].LastChannelFrequency = session.LastChannel
			snapshot.Devices[index].CurrentDataRate = session.CurrentDataRate
			snapshot.Devices[index].CurrentSpreadingFactor = session.CurrentSpreadingFactor
			snapshot.Devices[index].LastPayloadSize = session.LastPayloadSize
			snapshot.Devices[index].LastFPort = session.LastFPort
			snapshot.Devices[index].ClassBSynchronized = session.ClassBSynchronized
			snapshot.Devices[index].LastBeaconTimestampMilliseconds = session.LastBeaconAt.Milliseconds()
			snapshot.Devices[index].NextPingSlotTimestampMilliseconds = session.NextPingSlotAt.Milliseconds()
			snapshot.Devices[index].ClassBMissedBeacons = session.ClassBMissedBeacons
		}
	}
	for index, gateway := range e.registry.Gateways() {
		if runtime := e.gatewayRuntime[gateway.ID]; runtime != nil {
			snapshot.Gateways[index].GatewayType = gateway.Type
			snapshot.Gateways[index].GatewayEUI = gateway.GatewayEUI
			snapshot.Gateways[index].GatewayState = runtime.State
			snapshot.Gateways[index].LastHeartbeatMilliseconds = runtime.LastHeartbeat.Milliseconds()
			snapshot.Gateways[index].ConnectionAttempts = runtime.ConnectionAttempts
			snapshot.Gateways[index].LastNetworkError = runtime.LastNetworkError
			snapshot.Gateways[index].IngressPackets = runtime.IngressPackets
			snapshot.Gateways[index].EgressPackets = runtime.EgressPackets
		}
	}
	return snapshot
}

// Checkpoint returns a consistent, serializable copy of the engine runtime.
// It is safe to call from an event sink while the simulation is running.
func (e *Engine) Checkpoint() types.EngineCheckpoint {
	e.mu.RLock()
	defer e.mu.RUnlock()

	checkpoint := types.EngineCheckpoint{
		Config:         e.config,
		Devices:        e.registry.Devices(),
		Gateways:       e.registry.Gateways(),
		State:          e.currentStateLocked(),
		Metrics:        e.metrics,
		Sessions:       make(map[string]types.DeviceSession, len(e.sessions)),
		Radio:          make(map[string]types.RadioTransmission, len(e.radio)),
		Scheduled:      e.scheduler.Events(),
		GatewayPackets: cloneGatewayPackets(e.gatewayPackets),
		EventLog:       recentSimulationEvents(e.eventLog, runtimeEventLogLimit),
	}
	for deviceID, session := range e.sessions {
		if session == nil {
			continue
		}
		copySession := *session
		if session.PendingUplink != nil {
			pending := *session.PendingUplink
			pending.GatewayIDs = append([]string(nil), pending.GatewayIDs...)
			pending.PayloadConfig = cloneRuntimePointer(pending.PayloadConfig)
			pending.WirePayload = append([]byte(nil), pending.WirePayload...)
			copySession.PendingUplink = &pending
		}
		if session.PendingJoinRequest != nil {
			pending := *session.PendingJoinRequest
			pending.GatewayIDs = append([]string(nil), pending.GatewayIDs...)
			copySession.PendingJoinRequest = &pending
		}
		copySession.PendingClassBDownlinks = cloneClassBDownlinks(session.PendingClassBDownlinks)
		copySession.PendingClassADownlinks = cloneDownlinks(session.PendingClassADownlinks)
		copySession.PendingClassCDownlinks = cloneDownlinks(session.PendingClassCDownlinks)
		copySession.AdditionalChannels = append([]types.RadioChannel(nil), session.AdditionalChannels...)
		copySession.PendingMACResponses = cloneMACResponses(session.PendingMACResponses)
		copySession.NetworkChannels = cloneSessionMap(session.NetworkChannels)
		copySession.DisabledChannels = cloneSessionMap(session.DisabledChannels)
		copySession.NwkSKey = append([]byte(nil), session.NwkSKey...)
		copySession.AppSKey = append([]byte(nil), session.AppSKey...)
		checkpoint.Sessions[deviceID] = copySession
	}
	for packetID, transmission := range e.radio {
		if transmission == nil {
			continue
		}
		copyTransmission := *transmission
		copyTransmission.GatewayIDs = append([]string(nil), transmission.GatewayIDs...)
		copyTransmission.Payload = append([]byte(nil), transmission.Payload...)
		checkpoint.Radio[packetID] = copyTransmission
	}
	return checkpoint
}

func (e *Engine) EventLog() []contracts.SimulationEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()

	log := make([]contracts.SimulationEvent, len(e.eventLog))
	copy(log, e.eventLog)
	return log
}

func (e *Engine) Events() <-chan contracts.SimulationEvent {
	return e.events
}

func (e *Engine) run(ctx context.Context) {
	defer func() {
		e.stopGatewayAdapters()
		e.gatewayWG.Wait()
		e.mu.Lock()
		if e.state.Status != contracts.SimulationStatusFailed {
			e.state.Status = contracts.SimulationStatusStopped
		}
		event := e.newEventLocked(
			contracts.SimulationStopped,
			"simulation stopped",
			"",
			"",
			"",
		)

		done := e.done
		e.runCancel = nil
		e.runContext = nil
		e.done = nil
		e.mu.Unlock()

		e.publish(event)
		close(done)
		close(e.events)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		e.mu.RLock()
		status := e.state.Status
		next, hasNext := e.scheduler.Peek()
		speed := e.state.Speed
		e.mu.RUnlock()

		if status == contracts.SimulationStatusPaused || !hasNext {
			select {
			case <-ctx.Done():
				return
			case <-e.wake:
			}
			continue
		}

		if err := e.clock.WaitUntil(ctx, next.At, speed, e.wake); err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}

		e.mu.Lock()
		// Pause may have won the lock while WaitUntil was returning. Re-check
		// the lifecycle state before consuming a due event so no scheduled work
		// is processed after the pause transition has been committed.
		if e.state.Status != contracts.SimulationStatusRunning {
			e.mu.Unlock()
			continue
		}
		due, ok := e.scheduler.PopDue(e.clock.Now())
		if !ok {
			e.mu.Unlock()
			continue
		}
		if elapsed := e.clock.Now().Milliseconds(); elapsed > e.state.ElapsedMilliseconds {
			e.state.ElapsedMilliseconds = elapsed
		}
		e.processingAt = due.At
		e.processing = true
		events := e.processScheduledEventLocked(due)
		e.processing = false
		gatewayPackets := e.drainGatewayPacketsLocked()
		e.mu.Unlock()
		for _, event := range events {
			e.publish(event)
		}
		e.dispatchGatewayPackets(gatewayPackets)
	}
}

func locationCoordinates(device contracts.Device) (float64, float64) {
	return float64(pointerValue(device.LocationConfig.Latitude)), float64(pointerValue(device.LocationConfig.Longitude))
}

func gatewayCoordinates(gateway contracts.Gateway) (float64, float64) {
	return float64(pointerValue(gateway.Latitude)), float64(pointerValue(gateway.Longitude))
}

func pointerValue[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}
	return *value
}

func int32Pointer(value int) *int32 {
	converted := int32(value)
	return &converted
}

func newDeviceSession(device contracts.Device) *types.DeviceSession {
	plan := regional.MustPlan(device.LocationConfig.Region)
	dataRate := defaultUplinkDataRate(device.LocationConfig.Region)
	if configured := device.AdvancedConfig.UplinkDataRate; configured != nil {
		dataRate = regional.ClampDataRate(device.LocationConfig.Region, *configured)
	}
	session := &types.DeviceSession{
		Joined:                 device.Activation == contracts.ABP,
		FrameCounterUp:         initialFrameCounter(device.FrameConfig.FCntUp),
		FrameCounterDown:       initialFrameCounter(device.FrameConfig.FCntDown),
		CurrentDataRate:        dataRate,
		CurrentSpreadingFactor: spreadingFactorForDataRate(device.LocationConfig.Region, dataRate),
		LastFPort:              device.FrameConfig.FPort,
		CurrentTxPower:         int(radioTransmitPowerDBm),
		UnconfirmedRepetitions: 1,
		MaximumDutyCycle:       1,
		RX1DataRateOffset:      intValue(device.RX1Config.DataRateOffset),
		RX2DataRate:            intValue(device.RX2Config.DataRate),
		RX2Frequency:           int64(device.RX2Config.ChannelFrequency),
		ReceiveDelay:           durationMilliseconds(device.RX1Config.Delay),
		MaximumEIRP:            int(radioTransmitPowerDBm),
		PingSlotDataRate:       plan.PingSlotDataRate,
		PingSlotFrequency:      plan.PingSlotFrequency,
		BeaconFrequency:        plan.BeaconFrequency,
	}
	if device.ABPConfig != nil {
		session.DeviceAddress = device.ABPConfig.DevAddr
		session.SecurityFingerprint = securityFingerprint(device.DevEUI, device.ABPConfig.NwkSKey, device.ABPConfig.AppSKey)
		if nwkSKey, err := lorawan.DecodeHex(device.ABPConfig.NwkSKey); err == nil {
			session.NwkSKey = nwkSKey
		}
		if appSKey, err := lorawan.DecodeHex(device.ABPConfig.AppSKey); err == nil {
			session.AppSKey = appSKey
		}
	}
	if device.OOTAConfig != nil {
		session.JoinEUI = device.OOTAConfig.JoinEUI
		session.SecurityFingerprint = securityFingerprint(device.DevEUI, device.OOTAConfig.JoinEUI, device.OOTAConfig.AppKey)
	}
	return session
}

func deriveOTAAKeys(device contracts.Device, session *types.DeviceSession) {
	if device.OOTAConfig == nil || session == nil {
		return
	}
	appKey, err := lorawan.DecodeHex(device.OOTAConfig.AppKey)
	if err != nil {
		return
	}
	nwkSKey, appSKey, err := lorawan.DeriveSessionKeys(appKey, 0, 0, session.DevNonce)
	if err != nil {
		return
	}
	session.NwkSKey = nwkSKey
	session.AppSKey = appSKey
}

func cloneClassBDownlinks(downlinks []types.ClassBDownlink) []types.ClassBDownlink {
	if len(downlinks) == 0 {
		return nil
	}
	cloned := make([]types.ClassBDownlink, len(downlinks))
	for index, downlink := range downlinks {
		cloned[index] = downlink
		cloned[index].Payload = append([]byte(nil), downlink.Payload...)
		cloned[index].MACCommands = cloneMACCommands(downlink.MACCommands)
	}
	return cloned
}

func cloneDownlinks(downlinks []types.Downlink) []types.Downlink {
	if len(downlinks) == 0 {
		return nil
	}
	cloned := make([]types.Downlink, len(downlinks))
	for index, downlink := range downlinks {
		cloned[index] = downlink
		cloned[index].Payload = append([]byte(nil), downlink.Payload...)
	}
	return cloned
}

func cloneGatewayPackets(packets []types.GatewayPacket) []types.GatewayPacket {
	if len(packets) == 0 {
		return nil
	}
	cloned := make([]types.GatewayPacket, len(packets))
	for index, packet := range packets {
		cloned[index] = packet
		cloned[index].Payload = append([]byte(nil), packet.Payload...)
		cloned[index].ConcentratorTimestamp = cloneRuntimePointer(packet.ConcentratorTimestamp)
	}
	return cloned
}

func cloneMACCommands(commands []types.MACCommand) []types.MACCommand {
	if len(commands) == 0 {
		return nil
	}
	cloned := make([]types.MACCommand, len(commands))
	for index, command := range commands {
		cloned[index] = command
		cloned[index].DataRate = cloneRuntimePointer(command.DataRate)
		cloned[index].TxPower = cloneRuntimePointer(command.TxPower)
		cloned[index].NbTrans = cloneRuntimePointer(command.NbTrans)
		cloned[index].ChannelMask = cloneRuntimePointer(command.ChannelMask)
		cloned[index].ChannelMaskControl = cloneRuntimePointer(command.ChannelMaskControl)
		cloned[index].MaxDutyCycleExponent = cloneRuntimePointer(command.MaxDutyCycleExponent)
		cloned[index].RX1DataRateOffset = cloneRuntimePointer(command.RX1DataRateOffset)
		cloned[index].Frequency = cloneRuntimePointer(command.Frequency)
		cloned[index].ChannelIndex = cloneRuntimePointer(command.ChannelIndex)
		cloned[index].MinimumDataRate = cloneRuntimePointer(command.MinimumDataRate)
		cloned[index].MaximumDataRate = cloneRuntimePointer(command.MaximumDataRate)
		cloned[index].Delay = cloneRuntimePointer(command.Delay)
		cloned[index].UplinkDwellTime = cloneRuntimePointer(command.UplinkDwellTime)
		cloned[index].DownlinkDwellTime = cloneRuntimePointer(command.DownlinkDwellTime)
		cloned[index].MaximumEIRP = cloneRuntimePointer(command.MaximumEIRP)
		cloned[index].Margin = cloneRuntimePointer(command.Margin)
		cloned[index].GatewayCount = cloneRuntimePointer(command.GatewayCount)
		cloned[index].BatteryLevel = cloneRuntimePointer(command.BatteryLevel)
		cloned[index].PingSlotPeriodicity = cloneRuntimePointer(command.PingSlotPeriodicity)
		cloned[index].DeviceTime = cloneRuntimePointer(command.DeviceTime)
	}
	return cloned
}

func cloneRuntimePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func securityFingerprint(values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func joinedDeviceAddress(session *types.DeviceSession) string {
	if session == nil || session.SecurityFingerprint == "" {
		return ""
	}
	return session.SecurityFingerprint[:8]
}

func initialFrameCounter(value **int) int64 {
	if value == nil || *value == nil {
		return 0
	}
	return int64(**value)
}

func (e *Engine) newEventLocked(
	eventType contracts.SimulationEventType,
	message string,
	deviceID string,
	gatewayID string,
	scheduledEventID string,
) contracts.SimulationEvent {
	e.state.EventSequence++
	event := contracts.SimulationEvent{
		ID:                    uuid.NewString(),
		Sequence:              e.state.EventSequence,
		Type:                  eventType,
		TimestampMilliseconds: e.eventTimestampLocked().Milliseconds(),
		Message:               message,
	}
	if deviceID != "" {
		event.DeviceID = &deviceID
	}
	if gatewayID != "" {
		event.GatewayID = &gatewayID
	}
	if scheduledEventID != "" {
		event.ScheduledEventID = &scheduledEventID
	}
	e.eventLog = append(e.eventLog, event)
	if len(e.eventLog) >= runtimeEventLogLimit*2 {
		e.eventLog = recentSimulationEvents(e.eventLog, runtimeEventLogLimit)
	}
	return event
}

func recentSimulationEvents(events []contracts.SimulationEvent, limit int) []contracts.SimulationEvent {
	if limit <= 0 || len(events) == 0 {
		return []contracts.SimulationEvent{}
	}
	start := 0
	if len(events) > limit {
		start = len(events) - limit
	}
	return append([]contracts.SimulationEvent(nil), events[start:]...)
}

func (e *Engine) eventTimestampLocked() time.Duration {
	if e.processing {
		return e.processingAt
	}
	return e.clock.Now()
}

func (e *Engine) newPacketEventLocked(
	eventType contracts.SimulationEventType,
	message string,
	deviceID string,
	gatewayID string,
	scheduledEventID string,
	packetID string,
) contracts.SimulationEvent {
	event := e.newEventLocked(eventType, message, deviceID, gatewayID, scheduledEventID)
	event.PacketID = &packetID
	e.eventLog[len(e.eventLog)-1] = event
	return event
}

func (e *Engine) publish(event contracts.SimulationEvent) {
	if e.sink != nil {
		e.sink(event)
	}

	select {
	case e.events <- event:
	default:
		// EventLog is authoritative. The channel is a non-blocking live view so
		// a slow consumer cannot stop the simulation loop.
	}
}

func (e *Engine) signalWake() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

func cloneMACResponses(responses [][]byte) [][]byte {
	if responses == nil {
		return nil
	}
	result := make([][]byte, len(responses))
	for i, r := range responses {
		result[i] = append([]byte(nil), r...)
	}
	return result
}
func cloneSessionMap[V any](source map[int]V) map[int]V {
	if source == nil {
		return nil
	}
	result := make(map[int]V, len(source))
	for k, v := range source {
		result[k] = v
	}
	return result
}

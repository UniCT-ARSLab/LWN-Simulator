package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
	"lwn-simulator-backend/internal/apperrors"
)

var ErrSimulationNotStarted = errors.New("simulation has not been started")

type SimulationDeviceSource interface {
	GetDevices(contracts.GetDevicesRequest) (contracts.GetDevicesResponse, error)
}

type SimulationGatewaySource interface {
	GetGateways(contracts.GetGatewaysRequest) (contracts.GetGatewaysResponse, error)
}

type SimulationUpdate struct {
	Snapshot        contracts.SimulationSnapshot
	Event           contracts.SimulationEvent
	IncludeSnapshot bool
}

type simulationPersistenceCommand struct {
	runID   string
	event   *contracts.SimulationEvent
	barrier chan error
}

const (
	simulationPersistenceInterval = 500 * time.Millisecond
	simulationCheckpointInterval  = time.Second
	simulationSnapshotInterval    = 500 * time.Millisecond
	simulationPersistenceQueue    = 65_536
	simulationPersistenceBatch    = 8_192
)

type SimulationSubscription struct {
	Updates <-chan SimulationUpdate
	Close   func()
}

type SimulationService struct {
	devices         SimulationDeviceSource
	gateways        SimulationGatewaySource
	options         types.Options
	checkpointStore SimulationCheckpointStore
	logStore        SimulationLogStore

	operationMu                    sync.Mutex
	mu                             sync.RWMutex
	engine                         *engine.Engine
	starting                       bool
	nextID                         uint64
	activeRunID                    string
	activeRunStartedAtMilliseconds int64
	subscribers                    map[uint64]chan SimulationUpdate
	realtimePublisher              RealtimePublisher

	persistenceQueue chan simulationPersistenceCommand
	persistenceDone  chan struct{}
	persistenceOnce  sync.Once
	lastSnapshotAt   time.Time
}

func (s *SimulationService) SetRealtimePublisher(publisher RealtimePublisher) {
	s.mu.Lock()
	s.realtimePublisher = publisher
	s.mu.Unlock()
}

func NewSimulationService(
	devices SimulationDeviceSource,
	gateways SimulationGatewaySource,
	options types.Options,
	stores ...interface{},
) *SimulationService {
	var checkpointStore SimulationCheckpointStore
	if len(stores) > 0 {
		checkpointStore, _ = stores[0].(SimulationCheckpointStore)
	}
	var logStore SimulationLogStore
	if len(stores) > 1 {
		logStore, _ = stores[1].(SimulationLogStore)
	}
	service := &SimulationService{
		devices:          devices,
		gateways:         gateways,
		options:          options,
		checkpointStore:  checkpointStore,
		logStore:         logStore,
		subscribers:      make(map[uint64]chan SimulationUpdate),
		persistenceQueue: make(chan simulationPersistenceCommand, simulationPersistenceQueue),
	}
	if checkpointStore != nil || logStore != nil {
		service.persistenceDone = make(chan struct{})
		go service.runPersistenceWorker()
	}
	return service
}

// Close releases the persistence worker for a profile runtime that is no
// longer reachable. It is safe to call more than once.
func (s *SimulationService) Close() {
	s.persistenceOnce.Do(func() {
		if s.persistenceDone != nil {
			close(s.persistenceDone)
		}
	})
}

// RestorePersisted resumes a simulation that was active when the backend
// stopped. A deliberately stopped or failed simulation is not restarted.
func (s *SimulationService) RestorePersisted(ctx context.Context) error {
	if s.checkpointStore == nil {
		return nil
	}
	checkpoint, err := s.checkpointStore.Load()
	if err != nil || checkpoint == nil {
		if err != nil {
			return err
		}
		return s.reconcileOrphanedLogs()
	}
	if checkpoint.State.Status == contracts.SimulationStatusStopped || checkpoint.State.Status == contracts.SimulationStatusFailed {
		if err := s.checkpointStore.Clear(); err != nil {
			return err
		}
		return s.reconcileOrphanedLogs()
	}
	s.mu.Lock()
	s.activeRunID = checkpoint.RunID
	s.mu.Unlock()
	_, err = s.Start(ctx, checkpoint.Config)
	return err
}

func (s *SimulationService) reconcileOrphanedLogs() error {
	if s.logStore == nil {
		return nil
	}
	runs, err := s.logStore.List()
	if err != nil {
		return err
	}
	for _, run := range runs.Runs {
		if run.Summary.Status != contracts.SimulationRunStatusRunning && run.Summary.Status != contracts.SimulationRunStatusPaused {
			continue
		}
		summary := run.Summary
		summary.Status = contracts.SimulationRunStatusFailed
		summary.EndedAtMilliseconds = int64Pointer(time.Now().UnixMilli())
		if err := s.logStore.Finalize(run.Summary.Id, summary); err != nil {
			return fmt.Errorf("finalize orphaned simulation log %s: %w", run.Summary.Id, err)
		}
	}
	return nil
}

func (s *SimulationService) Start(
	ctx context.Context,
	config contracts.SimulationConfig,
) (contracts.SimulationSnapshot, error) {
	if ctx == nil {
		return contracts.SimulationSnapshot{}, fmt.Errorf("start context cannot be nil")
	}
	s.operationMu.Lock()
	defer s.operationMu.Unlock()

	s.mu.Lock()
	if s.starting {
		s.mu.Unlock()
		return contracts.SimulationSnapshot{}, apperrors.Conflict("simulation is already starting")
	}
	if s.engine != nil {
		status := s.engine.Snapshot().State.Status
		if status != contracts.SimulationStatusStopped && status != contracts.SimulationStatusFailed {
			s.mu.Unlock()
			return contracts.SimulationSnapshot{}, apperrors.Conflict("a simulation is already active")
		}
	}
	s.starting = true
	s.mu.Unlock()

	devices, err := s.devices.GetDevices(contracts.GetDevicesRequest{})
	if err != nil {
		s.finishStarting()
		return contracts.SimulationSnapshot{}, fmt.Errorf("load devices: %w", err)
	}
	if !hasActiveSimulationDevice(devices.Devices) {
		s.finishStarting()
		return contracts.SimulationSnapshot{}, apperrors.Invalid("at least one active device is required to start a simulation")
	}
	gateways, err := s.gateways.GetGateways(contracts.GetGatewaysRequest{})
	if err != nil {
		s.finishStarting()
		return contracts.SimulationSnapshot{}, fmt.Errorf("load gateways: %w", err)
	}

	options := s.options
	options.EventSink = s.handleEvent
	if config.GatewayBridge != nil {
		if configurator, ok := options.GatewayAdapterFactory.(types.GatewayBridgeConfigurator); ok {
			if err := configurator.ConfigureGatewayBridge(*config.GatewayBridge); err != nil {
				s.finishStarting()
				return contracts.SimulationSnapshot{}, fmt.Errorf("configure gateway bridge: %w", err)
			}
		} else if options.GatewayAdapterFactory != nil {
			s.finishStarting()
			return contracts.SimulationSnapshot{}, fmt.Errorf("gateway adapter factory does not support gateway bridge configuration")
		}
	}
	var persistedCheckpoint *types.EngineCheckpoint
	if s.checkpointStore != nil {
		checkpoint, checkpointErr := s.checkpointStore.Load()
		if checkpointErr != nil {
			s.finishStarting()
			return contracts.SimulationSnapshot{}, checkpointErr
		}
		if checkpoint != nil && checkpoint.State.Status != contracts.SimulationStatusStopped && checkpoint.State.Status != contracts.SimulationStatusFailed {
			options.Checkpoint = checkpoint
			persistedCheckpoint = checkpoint
		}
	}
	if persistedCheckpoint == nil {
		s.mu.Lock()
		s.activeRunID = ""
		s.mu.Unlock()
	}
	runtime, err := engine.New(config, devices.Devices, gateways.Gateways, options)
	if err != nil {
		s.finishStarting()
		return contracts.SimulationSnapshot{}, fmt.Errorf("create simulation engine: %w", err)
	}

	s.mu.Lock()
	s.engine = runtime
	if s.activeRunID == "" {
		if persistedCheckpoint != nil {
			s.activeRunID = persistedCheckpoint.RunID
		}
		if s.activeRunID == "" {
			s.activeRunID = uuid.NewString()
		}
	}
	runID := s.activeRunID
	runStartedAtMilliseconds := time.Now().UnixMilli()
	if persistedCheckpoint != nil && s.logStore != nil {
		if persistedRun, persistedRunErr := s.logStore.GetRun(runID); persistedRunErr == nil && persistedRun.Summary.StartedAtMilliseconds > 0 {
			runStartedAtMilliseconds = persistedRun.Summary.StartedAtMilliseconds
		}
	}
	s.activeRunStartedAtMilliseconds = runStartedAtMilliseconds
	s.starting = false
	s.mu.Unlock()
	if s.logStore != nil && persistedCheckpoint == nil {
		objects := make([]contracts.SimulationLogObject, 0, len(devices.Devices)+len(gateways.Gateways))
		for _, device := range devices.Devices {
			objects = append(objects, contracts.SimulationLogObject{Id: device.ID, Name: device.Name, Kind: contracts.SimulationLogObjectKindDevice, Identifier: device.DevEUI})
		}
		for _, gateway := range gateways.Gateways {
			objects = append(objects, contracts.SimulationLogObject{Id: gateway.ID, Name: gateway.Name, Kind: contracts.SimulationLogObjectKindGateway, Identifier: gateway.GatewayEUI})
		}
		seed := config.Seed
		state, _ := runtime.StateAndMetrics()
		run := contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: runID, StartedAtMilliseconds: runStartedAtMilliseconds, Status: contracts.SimulationRunStatusRunning, Speed: state.Speed, Seed: seed, DeviceCount: int32(len(devices.Devices)), GatewayCount: int32(len(gateways.Gateways))}, Objects: objects}
		if err := s.logStore.Create(run); err != nil {
			s.resetRuntimeAfterStartFailure(runtime)
			return contracts.SimulationSnapshot{}, fmt.Errorf("create simulation log: %w", err)
		}
		s.publishRunSummary(contracts.RealtimeSimulationRunCreatedMessage, run.Summary)
	}

	// The service owns the simulation lifecycle. In particular, an HTTP request
	// context must not stop the engine as soon as the start response is sent.
	if err := runtime.Start(context.WithoutCancel(ctx)); err != nil {
		failure := fmt.Errorf("start simulation engine: %w", err)
		if persistenceErr := s.flushPersistence(context.Background()); persistenceErr != nil {
			failure = errors.Join(failure, fmt.Errorf("flush failed simulation events: %w", persistenceErr))
		}
		if s.logStore != nil {
			state, metrics := runtime.StateAndMetrics()
			summary := s.runSummaryFor(runID, state, metrics, contracts.SimulationRunStatusFailed)
			if finalizeErr := s.logStore.Finalize(runID, summary); finalizeErr != nil {
				failure = errors.Join(failure, fmt.Errorf("finalize failed simulation log: %w", finalizeErr))
			}
		}
		if s.checkpointStore != nil {
			if clearErr := s.checkpointStore.Clear(); clearErr != nil {
				failure = errors.Join(failure, fmt.Errorf("clear failed simulation checkpoint: %w", clearErr))
			}
		}
		s.resetRuntimeAfterStartFailure(runtime)
		return contracts.SimulationSnapshot{}, failure
	}

	snapshot := runtime.Snapshot()
	s.publishSnapshot(snapshot)
	return snapshot, nil
}

func hasActiveSimulationDevice(devices []contracts.Device) bool {
	for _, device := range devices {
		if device.Active {
			return true
		}
	}
	return false
}

func (s *SimulationService) Pause() (contracts.SimulationSnapshot, error) {
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	if err := runtime.Pause(); err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	snapshot := runtime.Snapshot()
	s.publishSnapshot(snapshot)
	return snapshot, nil
}

func (s *SimulationService) Resume() (contracts.SimulationSnapshot, error) {
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	if err := runtime.Resume(); err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	snapshot := runtime.Snapshot()
	s.publishSnapshot(snapshot)
	return snapshot, nil
}

func (s *SimulationService) Stop(ctx context.Context) (contracts.SimulationSnapshot, error) {
	s.operationMu.Lock()
	defer s.operationMu.Unlock()
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	if err := runtime.Stop(ctx); err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	if err := s.flushPersistence(ctx); err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	if s.checkpointStore != nil {
		if err := s.checkpointStore.Clear(); err != nil {
			return contracts.SimulationSnapshot{}, err
		}
	}
	snapshot := runtime.Snapshot()
	if s.logStore != nil {
		runID := s.currentRunID()
		summary := s.runSummaryFor(runID, snapshot.State, snapshot.Metrics, contracts.SimulationRunStatusStopped)
		if err := s.logStore.Finalize(runID, summary); err != nil {
			return contracts.SimulationSnapshot{}, err
		}
		s.publishRunSummary(contracts.RealtimeSimulationRunUpdatedMessage, summary)
	}
	s.publishSnapshot(snapshot)
	return snapshot, nil
}

func (s *SimulationService) Snapshot() (contracts.SimulationSnapshot, error) {
	runtime, err := s.currentEngine()
	if err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	return runtime.Snapshot(), nil
}

func (s *SimulationService) Subscribe() (SimulationSubscription, error) {
	if _, err := s.currentEngine(); err != nil {
		return SimulationSubscription{}, err
	}

	channel := make(chan SimulationUpdate, 128)
	s.mu.Lock()
	s.nextID++
	id := s.nextID
	s.subscribers[id] = channel
	s.mu.Unlock()

	return SimulationSubscription{
		Updates: channel,
		Close: func() {
			s.mu.Lock()
			if _, ok := s.subscribers[id]; ok {
				delete(s.subscribers, id)
			}
			s.mu.Unlock()
		},
	}, nil
}

func (s *SimulationService) currentEngine() (*engine.Engine, error) {
	s.mu.RLock()
	runtime := s.engine
	s.mu.RUnlock()
	if runtime == nil {
		return nil, apperrors.Conflict("%s", ErrSimulationNotStarted.Error())
	}
	return runtime, nil
}

func (s *SimulationService) finishStarting() {
	s.mu.Lock()
	s.starting = false
	s.mu.Unlock()
}

func (s *SimulationService) resetRuntimeAfterStartFailure(runtime *engine.Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.engine == runtime {
		s.engine = nil
	}
	s.starting = false
	s.activeRunID = ""
	s.activeRunStartedAtMilliseconds = 0
}

func (s *SimulationService) currentRunID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeRunID
}

func (s *SimulationService) publishEvent(runID string, event contracts.SimulationEvent, update SimulationUpdate) {
	s.mu.RLock()
	publisher := s.realtimePublisher
	s.mu.RUnlock()
	if publisher == nil {
		return
	}
	message := contracts.RealtimeWebSocketMessage{
		Type:                  contracts.RealtimeSimulationEventMessage,
		TimestampMilliseconds: time.Now().UnixMilli(),
		RunID:                 stringPointerOrNil(runID),
		Event:                 &event,
	}
	if update.IncludeSnapshot {
		message.Snapshot = &update.Snapshot
	}
	publisher.Publish(message)
}

func (s *SimulationService) publishSnapshot(snapshot contracts.SimulationSnapshot) {
	s.mu.RLock()
	publisher := s.realtimePublisher
	runID := s.activeRunID
	s.mu.RUnlock()
	if publisher == nil {
		return
	}
	publisher.Publish(contracts.RealtimeWebSocketMessage{
		Type:                  contracts.RealtimeSimulationSnapshotMessage,
		TimestampMilliseconds: time.Now().UnixMilli(),
		RunID:                 stringPointerOrNil(runID),
		Snapshot:              &snapshot,
	})
}

func (s *SimulationService) publishRunSummary(messageType contracts.RealtimeWebSocketMessageType, summary contracts.SimulationRunSummary) {
	s.mu.RLock()
	publisher := s.realtimePublisher
	s.mu.RUnlock()
	if publisher == nil {
		return
	}
	publisher.Publish(contracts.RealtimeWebSocketMessage{
		Type:                  messageType,
		TimestampMilliseconds: time.Now().UnixMilli(),
		RunID:                 stringPointerOrNil(summary.Id),
		RunSummary:            &summary,
	})
}

func stringPointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (s *SimulationService) handleEvent(event contracts.SimulationEvent) {
	s.mu.Lock()
	runtime := s.engine
	runID := s.activeRunID
	updates := make([]chan SimulationUpdate, 0, len(s.subscribers))
	for _, channel := range s.subscribers {
		updates = append(updates, channel)
	}
	includeSnapshot := (len(updates) > 0 || s.realtimePublisher != nil) && (s.lastSnapshotAt.IsZero() || time.Since(s.lastSnapshotAt) >= simulationSnapshotInterval || isLifecycleEvent(event.Type))
	if includeSnapshot {
		s.lastSnapshotAt = time.Now()
	}
	s.mu.Unlock()
	if runtime == nil {
		return
	}

	if s.checkpointStore != nil || s.logStore != nil {
		copy := event
		s.persistenceQueue <- simulationPersistenceCommand{runID: runID, event: &copy}
	}

	update := SimulationUpdate{Event: event, IncludeSnapshot: includeSnapshot}
	if includeSnapshot {
		update.Snapshot = runtime.Snapshot()
	}
	s.publishEvent(runID, event, update)
	for _, channel := range updates {
		select {
		case channel <- update:
		default:
			// A slow websocket client must not pause the simulation loop.
		}
	}
}

func (s *SimulationService) runPersistenceWorker() {
	ticker := time.NewTicker(simulationPersistenceInterval)
	defer ticker.Stop()
	pending := make([]contracts.SimulationEvent, 0, simulationPersistenceBatch)
	runID := ""
	var terminalStatus *contracts.SimulationRunStatus
	lastCheckpointAt := time.Time{}
	lastCheckpointRunID := ""
	lastCheckpointStatus := contracts.SimulationStatus("")

	flush := func(forceCheckpoint bool) error {
		if runID == "" && len(pending) == 0 && terminalStatus == nil {
			return nil
		}
		s.mu.RLock()
		runtime := s.engine
		activeRunID := s.activeRunID
		s.mu.RUnlock()
		if runtime == nil || activeRunID == "" || (runID != "" && runID != activeRunID) {
			pending = pending[:0]
			runID = ""
			terminalStatus = nil
			return nil
		}

		state, metrics := runtime.StateAndMetrics()
		var persistenceErr error
		if s.logStore != nil && len(pending) > 0 {
			summary := s.runSummaryFor(activeRunID, state, metrics, contracts.SimulationRunStatus(state.Status))
			summary.EndedAtMilliseconds = nil
			if err := s.logStore.AppendBatch(activeRunID, pending, &summary); err != nil {
				persistenceErr = errors.Join(persistenceErr, fmt.Errorf("append simulation log batch: %w", err))
			} else {
				s.publishRunSummary(contracts.RealtimeSimulationRunUpdatedMessage, summary)
				pending = pending[:0]
			}
		} else if s.logStore == nil {
			pending = pending[:0]
		}
		if s.checkpointStore != nil && (forceCheckpoint || lastCheckpointRunID != activeRunID || time.Since(lastCheckpointAt) >= simulationCheckpointInterval) {
			terminal := state.Status == contracts.SimulationStatusStopped || state.Status == contracts.SimulationStatusFailed
			// A terminal checkpoint is a marker used to prevent automatic restart.
			checkpoint := types.EngineCheckpoint{RunID: activeRunID, State: state}
			if !terminal || lastCheckpointRunID != activeRunID || state.Status != lastCheckpointStatus {
				if !terminal {
					checkpoint = runtime.Checkpoint()
					checkpoint.RunID = activeRunID
				}
				if err := s.checkpointStore.Save(checkpoint); err != nil {
					persistenceErr = errors.Join(persistenceErr, fmt.Errorf("save simulation checkpoint: %w", err))
				} else {
					lastCheckpointAt = time.Now()
					lastCheckpointStatus = checkpoint.State.Status
					lastCheckpointRunID = activeRunID
				}
			}
		}
		if terminalStatus != nil && len(pending) == 0 {
			status := *terminalStatus
			if s.logStore != nil {
				summary := s.runSummaryFor(activeRunID, state, metrics, status)
				if err := s.logStore.Finalize(activeRunID, summary); err != nil {
					persistenceErr = errors.Join(persistenceErr, fmt.Errorf("finalize simulation log: %w", err))
				} else {
					s.publishRunSummary(contracts.RealtimeSimulationRunUpdatedMessage, summary)
				}
			}
			if s.checkpointStore != nil && persistenceErr == nil {
				if err := s.checkpointStore.Clear(); err != nil {
					persistenceErr = errors.Join(persistenceErr, fmt.Errorf("clear simulation checkpoint: %w", err))
				}
			}
			if persistenceErr == nil {
				terminalStatus = nil
			}
		}
		if len(pending) == 0 && terminalStatus == nil {
			runID = ""
		}
		return persistenceErr
	}

	for {
		select {
		case <-s.persistenceDone:
			return
		case command := <-s.persistenceQueue:
			if command.event != nil {
				if runID == "" {
					runID = command.runID
				}
				pending = append(pending, *command.event)
				if command.event.Type == contracts.SimulationFailed {
					status := contracts.SimulationRunStatusFailed
					terminalStatus = &status
				}
			}
			if command.barrier != nil {
				command.barrier <- flush(true)
				continue
			}
			if terminalStatus != nil {
				if err := flush(true); err != nil {
					log.Printf("simulation persistence failed: %v", err)
				}
			}
			if len(pending) >= simulationPersistenceBatch {
				if err := flush(false); err != nil {
					log.Printf("simulation persistence failed: %v", err)
				}
			}
		case <-ticker.C:
			if err := flush(false); err != nil {
				log.Printf("simulation persistence failed: %v", err)
			}
		}
	}
}

func (s *SimulationService) flushPersistence(ctx context.Context) error {
	if s.checkpointStore == nil && s.logStore == nil {
		return nil
	}
	barrier := make(chan error, 1)
	command := simulationPersistenceCommand{barrier: barrier}
	select {
	case s.persistenceQueue <- command:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-barrier:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func isLifecycleEvent(eventType contracts.SimulationEventType) bool {
	switch eventType {
	case contracts.SimulationStarted, contracts.SimulationPaused, contracts.SimulationResumed, contracts.SimulationStopped, contracts.SimulationFailed:
		return true
	default:
		return false
	}
}

func (s *SimulationService) ListLogs() (contracts.SimulationRunsResponse, error) {
	if s.logStore == nil {
		return contracts.SimulationRunsResponse{Runs: []contracts.SimulationRun{}, Total: 0}, nil
	}
	return s.logStore.List()
}

func (s *SimulationService) GetLog(id string) (contracts.SimulationRun, []contracts.SimulationEvent, error) {
	if s.logStore == nil {
		return contracts.SimulationRun{}, nil, os.ErrNotExist
	}
	return s.logStore.Get(id)
}

func (s *SimulationService) GetLogRun(id string) (contracts.SimulationRun, error) {
	if s.logStore == nil {
		return contracts.SimulationRun{}, os.ErrNotExist
	}
	return s.logStore.GetRun(id)
}

func (s *SimulationService) GetLogEvents(id string, query SimulationLogEventQuery) (contracts.SimulationEventsResponse, error) {
	if s.logStore == nil {
		return contracts.SimulationEventsResponse{}, os.ErrNotExist
	}
	return s.logStore.Events(id, query)
}

// FlushPersistence makes the current simulation history durable before a
// snapshot such as a profile export is read.
func (s *SimulationService) FlushPersistence(ctx context.Context) error {
	return s.flushPersistence(ctx)
}

func (s *SimulationService) ImportLogs(logs []contracts.ProfileLog) error {
	if s.logStore == nil {
		return errors.New("simulation log store is not configured")
	}
	for _, log := range logs {
		if err := s.logStore.Import(log.Run, log.Events); err != nil {
			return fmt.Errorf("import simulation log %q: %w", log.Run.Summary.Id, err)
		}
	}
	return nil
}

func (s *SimulationService) runSummaryFor(runID string, state contracts.SimulationState, metrics contracts.SimulationMetrics, status contracts.SimulationRunStatus) contracts.SimulationRunSummary {
	s.mu.RLock()
	startedAt := s.activeRunStartedAtMilliseconds
	s.mu.RUnlock()
	if startedAt <= 0 {
		startedAt = time.Now().UnixMilli() - state.ElapsedMilliseconds
	}
	return contracts.SimulationRunSummary{Id: runID, StartedAtMilliseconds: startedAt, EndedAtMilliseconds: int64Pointer(time.Now().UnixMilli()), DurationMilliseconds: state.ElapsedMilliseconds, Status: status, Speed: state.Speed, DeviceCount: int32(state.DeviceCount), GatewayCount: int32(state.GatewayCount), EventCount: int32(state.EventSequence), PacketSuccessRate: metrics.PacketSuccessRate}
}

func int64Pointer(value int64) *int64 { return &value }

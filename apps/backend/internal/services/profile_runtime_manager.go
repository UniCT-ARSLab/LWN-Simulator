package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	engineValidation "github.com/Giuseppe-Compagnone/lwn-engine/validation"
	"github.com/google/uuid"
	"lwn-simulator-backend/internal/apperrors"
	"lwn-simulator-backend/internal/database"
	"lwn-simulator-backend/internal/repositories"
)

const (
	profileArchiveFormat  = "lwn-simulator-profile"
	profileArchiveVersion = int32(1)
)

// ProfileRuntime contains the services backed by one profile directory.
// Runtime instances are created lazily and remain cached for the lifetime of
// the backend so websocket subscriptions and simulation state stay stable.
type ProfileRuntime struct {
	Profile       contracts.Profile
	Device        *DeviceService
	Gateway       *GatewayService
	Configuration *ProfileConfigurationService
	Simulation    *ProfileSimulationService
}

type ProfileRuntimeManager struct {
	dataDir       string
	profiles      *ProfileService
	options       types.Options
	realtime      RealtimePublisher
	mu            sync.Mutex
	runtimes      map[string]*ProfileRuntime
	activeProfile string
}

func NewProfileRuntimeManager(
	dataDir string,
	profiles *ProfileService,
	options types.Options,
	realtime RealtimePublisher,
) *ProfileRuntimeManager {
	manager := &ProfileRuntimeManager{
		dataDir:  dataDir,
		profiles: profiles,
		options:  options,
		realtime: realtime,
		runtimes: make(map[string]*ProfileRuntime),
	}
	if profiles != nil {
		profiles.SetDeleteGuard(manager.canDeleteProfile)
		profiles.SetDeleteObserver(manager.forgetProfile)
	}
	return manager
}

// Runtime returns the isolated runtime for an existing profile.
func (m *ProfileRuntimeManager) Runtime(profileID string) (*ProfileRuntime, error) {
	if m.profiles == nil {
		return nil, errors.New("profile service is not configured")
	}
	profile, err := m.profiles.Get(profileID)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if runtime, ok := m.runtimes[profile.ID]; ok {
		return runtime, nil
	}

	storage, err := database.NewProfileStorage(m.dataDir, profile.ID)
	if err != nil {
		return nil, err
	}
	if err := storage.EnsureDataFiles(); err != nil {
		return nil, fmt.Errorf("initialize profile %q storage: %w", profile.ID, err)
	}

	deviceService := NewDeviceService(repositories.NewDeviceRepository(storage.Directory()))
	gatewayService := NewGatewayService(repositories.NewGatewayRepository(storage.Directory()))
	innerSimulation := NewSimulationService(
		deviceService,
		gatewayService,
		m.options,
		NewFileSimulationCheckpointStore(filepath.Join(storage.Directory(), "simulation-checkpoint.json")),
		NewFileSimulationLogStore(filepath.Join(storage.Directory(), "simulation-logs.json")),
	)
	profilePublisher := profileRealtimePublisher{profileID: profile.ID, next: m.realtime}
	deviceService.SetRealtimePublisher(profilePublisher)
	gatewayService.SetRealtimePublisher(profilePublisher)
	innerSimulation.SetRealtimePublisher(profilePublisher)
	configurationService := NewProfileConfigurationService(
		filepath.Join(storage.Directory(), "gateway-bridge.json"),
		profile.ID,
	)
	configurationService.SetRealtimePublisher(profilePublisher)
	deviceService.SetRuntimeSynchronizer(innerSimulation)
	gatewayService.SetRuntimeSynchronizer(innerSimulation)

	runtime := &ProfileRuntime{
		Profile:       profile,
		Device:        deviceService,
		Gateway:       gatewayService,
		Configuration: configurationService,
		Simulation: &ProfileSimulationService{
			profileID: profile.ID,
			inner:     innerSimulation,
			manager:   m,
		},
	}
	m.runtimes[profile.ID] = runtime
	return runtime, nil
}

// RestorePersisted restores at most one active simulation across all
// profiles. A checkpoint in another profile is rejected instead of starting
// a second simulation, preserving the backend-wide simulation invariant.
func (m *ProfileRuntimeManager) RestorePersisted(ctx context.Context) error {
	profiles, err := m.profiles.List()
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		runtime, err := m.Runtime(profile.ID)
		if err != nil {
			return err
		}
		checkpoint, err := runtime.Simulation.inner.checkpointStore.Load()
		if err != nil {
			return fmt.Errorf("load profile %q checkpoint: %w", profile.ID, err)
		}
		var restoreErr error
		if checkpoint != nil && checkpoint.State.Status != contracts.SimulationStatusStopped && checkpoint.State.Status != contracts.SimulationStatusFailed {
			restoreErr = runtime.Simulation.RestorePersisted(ctx)
		} else {
			// No active checkpoint means there is no simulation to claim. The
			// inner service still reconciles orphaned logs when needed.
			restoreErr = runtime.Simulation.inner.RestorePersisted(ctx)
		}
		if restoreErr != nil {
			return fmt.Errorf("restore profile %q simulation: %w", profile.ID, restoreErr)
		}
	}
	return nil
}

func (m *ProfileRuntimeManager) ExportArchive(profileID string) (contracts.ProfileArchive, error) {
	runtime, err := m.Runtime(profileID)
	if err != nil {
		return contracts.ProfileArchive{}, err
	}
	if err := runtime.Simulation.FlushPersistence(context.Background()); err != nil {
		return contracts.ProfileArchive{}, fmt.Errorf("flush simulation history: %w", err)
	}
	devices, err := runtime.Device.GetDevices(contracts.GetDevicesRequest{})
	if err != nil {
		return contracts.ProfileArchive{}, fmt.Errorf("export devices: %w", err)
	}
	gateways, err := runtime.Gateway.GetGateways(contracts.GetGatewaysRequest{})
	if err != nil {
		return contracts.ProfileArchive{}, fmt.Errorf("export gateways: %w", err)
	}
	gatewayBridge, err := runtime.Configuration.GetGatewayBridge()
	if err != nil {
		return contracts.ProfileArchive{}, fmt.Errorf("export gateway bridge configuration: %w", err)
	}
	runs, err := runtime.Simulation.ListLogs()
	if err != nil {
		return contracts.ProfileArchive{}, fmt.Errorf("export simulation logs: %w", err)
	}
	logs := make([]contracts.ProfileLog, 0, len(runs.Runs))
	for _, run := range runs.Runs {
		storedRun, events, err := runtime.Simulation.GetLog(run.Summary.Id)
		if err != nil {
			return contracts.ProfileArchive{}, fmt.Errorf("export simulation log %q: %w", run.Summary.Id, err)
		}
		logs = append(logs, contracts.ProfileLog{Run: storedRun, Events: events})
	}
	return contracts.ProfileArchive{
		Format:        profileArchiveFormat,
		Version:       profileArchiveVersion,
		Profile:       runtime.Profile,
		Devices:       devices.Devices,
		Gateways:      gateways.Gateways,
		GatewayBridge: &gatewayBridge,
		Logs:          logs,
	}, nil
}

func (m *ProfileRuntimeManager) ImportArchive(archive contracts.ProfileArchive) (contracts.Profile, error) {
	if err := validateProfileArchive(archive); err != nil {
		return contracts.Profile{}, err
	}
	profile, err := m.profiles.Create(archive.Profile.Name)
	if errors.Is(err, apperrors.ErrConflict) {
		for suffix := 2; suffix <= 1000 && errors.Is(err, apperrors.ErrConflict); suffix++ {
			profile, err = m.profiles.Create(fmt.Sprintf("%s (imported %d)", archive.Profile.Name, suffix))
		}
	}
	if err != nil {
		return contracts.Profile{}, err
	}
	runtime, err := m.Runtime(profile.ID)
	if err == nil {
		err = runtime.Device.ImportDevices(archive.Devices)
	}
	if err == nil {
		err = runtime.Gateway.ImportGateways(archive.Gateways)
	}
	if err == nil && archive.GatewayBridge != nil {
		_, err = runtime.Configuration.UpdateGatewayBridge(*archive.GatewayBridge)
	}
	if err == nil {
		for index := range archive.Logs {
			archive.Logs[index].Run.Summary.Status = contracts.SimulationRunStatusStopped
			if archive.Logs[index].Run.Summary.EndedAtMilliseconds == nil {
				endedAt := time.Now().UnixMilli()
				archive.Logs[index].Run.Summary.EndedAtMilliseconds = &endedAt
			}
		}
		err = runtime.Simulation.inner.ImportLogs(archive.Logs)
	}
	if err != nil {
		if deleteErr := m.profiles.Delete(profile.ID); deleteErr != nil {
			return contracts.Profile{}, fmt.Errorf("import profile: %w; rollback failed: %v", err, deleteErr)
		}
		return contracts.Profile{}, fmt.Errorf("import profile: %w", err)
	}
	return profile, nil
}

func validateProfileArchive(archive contracts.ProfileArchive) error {
	if archive.Format != profileArchiveFormat || archive.Version != profileArchiveVersion {
		return apperrors.Invalid("unsupported profile archive format or version")
	}
	if _, err := uuid.Parse(archive.Profile.ID); err != nil {
		return apperrors.Invalid("profile archive contains an invalid profile id")
	}
	if _, err := normalizeProfileName(archive.Profile.Name); err != nil {
		return err
	}
	if err := engineValidation.ValidateHardware(archive.Devices, archive.Gateways); err != nil {
		return apperrors.Invalid("profile archive contains invalid hardware: %s", err)
	}
	if archive.GatewayBridge != nil {
		if err := engineValidation.ValidateSimulationConfig(contracts.SimulationConfig{
			Speed:         1,
			GatewayBridge: archive.GatewayBridge,
		}); err != nil {
			return apperrors.Invalid("profile archive contains invalid gateway bridge configuration: %s", err)
		}
	}
	return nil
}

func (m *ProfileRuntimeManager) acquireSimulation(profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeProfile == "" || m.activeProfile == profileID {
		m.activeProfile = profileID
		return nil
	}
	if runtime, ok := m.runtimes[m.activeProfile]; ok {
		if snapshot, err := runtime.Simulation.inner.Snapshot(); err == nil {
			status := snapshot.State.Status
			if status != contracts.SimulationStatusStopped && status != contracts.SimulationStatusFailed {
				return apperrors.Conflict("a simulation is already active for profile %q", m.activeProfile)
			}
		}
	}
	m.activeProfile = profileID
	return nil
}

func (m *ProfileRuntimeManager) releaseSimulation(profileID string) {
	m.mu.Lock()
	if m.activeProfile == profileID {
		m.activeProfile = ""
	}
	m.mu.Unlock()
}

// ActiveSimulation reports the single simulation that can run on this
// backend, regardless of which profile owns it.
func (m *ProfileRuntimeManager) ActiveSimulation() (contracts.SimulationActivity, error) {
	m.mu.Lock()
	profileID := m.activeProfile
	runtime := m.runtimes[profileID]
	m.mu.Unlock()

	activity := contracts.SimulationActivity{
		Active: false,
		Status: contracts.SimulationStatusIdle,
	}
	if profileID == "" || runtime == nil {
		return activity, nil
	}

	snapshot, err := runtime.Simulation.inner.Snapshot()
	if err != nil {
		return contracts.SimulationActivity{}, err
	}
	if snapshot.State.Status == contracts.SimulationStatusStopped || snapshot.State.Status == contracts.SimulationStatusFailed {
		m.releaseSimulation(profileID)
		return activity, nil
	}

	activity.Active = true
	activity.ProfileID = &profileID
	activity.Status = snapshot.State.Status
	return activity, nil
}

func (m *ProfileRuntimeManager) canDeleteProfile(profileID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeProfile != profileID {
		return nil
	}
	runtime, ok := m.runtimes[profileID]
	if !ok {
		m.activeProfile = ""
		return nil
	}
	snapshot, err := runtime.Simulation.inner.Snapshot()
	if err == nil && snapshot.State.Status != contracts.SimulationStatusStopped && snapshot.State.Status != contracts.SimulationStatusFailed {
		return apperrors.Conflict("cannot delete profile %q while its simulation is active", profileID)
	}
	m.activeProfile = ""
	return nil
}

func (m *ProfileRuntimeManager) forgetProfile(profileID string) {
	m.mu.Lock()
	runtime := m.runtimes[profileID]
	delete(m.runtimes, profileID)
	if m.activeProfile == profileID {
		m.activeProfile = ""
	}
	m.mu.Unlock()
	if runtime != nil {
		runtime.Simulation.inner.Close()
	}
}

// ProfileSimulationService coordinates one profile's simulation with the
// backend-wide single-simulation guard while exposing the existing service
// contract to handlers.
type ProfileSimulationService struct {
	profileID string
	inner     *SimulationService
	manager   *ProfileRuntimeManager
}

func (s *ProfileSimulationService) Start(ctx context.Context, config contracts.SimulationConfig) (contracts.SimulationSnapshot, error) {
	if err := s.manager.acquireSimulation(s.profileID); err != nil {
		return contracts.SimulationSnapshot{}, err
	}
	snapshot, err := s.inner.Start(ctx, config)
	if err != nil {
		s.manager.releaseSimulation(s.profileID)
	}
	return snapshot, err
}

func (s *ProfileSimulationService) SetSpeed(speed float64) (contracts.SimulationSnapshot, error) {
	return s.inner.SetSpeed(speed)
}

func (s *ProfileSimulationService) Pause() (contracts.SimulationSnapshot, error) {
	return s.inner.Pause()
}

func (s *ProfileSimulationService) Resume() (contracts.SimulationSnapshot, error) {
	return s.inner.Resume()
}

func (s *ProfileSimulationService) Stop(ctx context.Context) (contracts.SimulationSnapshot, error) {
	snapshot, err := s.inner.Stop(ctx)
	if err == nil {
		s.manager.releaseSimulation(s.profileID)
	}
	return snapshot, err
}

func (s *ProfileSimulationService) Snapshot() (contracts.SimulationSnapshot, error) {
	snapshot, err := s.inner.Snapshot()
	if err == nil && (snapshot.State.Status == contracts.SimulationStatusStopped || snapshot.State.Status == contracts.SimulationStatusFailed) {
		s.manager.releaseSimulation(s.profileID)
	}
	return snapshot, err
}

func (s *ProfileSimulationService) QueueUplink(req contracts.SimulationUplinkRequest) (contracts.SimulationActionResponse, error) {
	return s.inner.QueueUplink(req)
}

func (s *ProfileSimulationService) QueueDownlink(req contracts.SimulationDownlinkRequest) (contracts.SimulationActionResponse, error) {
	return s.inner.QueueDownlink(req)
}

func (s *ProfileSimulationService) QueueMACCommand(req contracts.SimulationMACCommandRequest) (contracts.SimulationActionResponse, error) {
	return s.inner.QueueMACCommand(req)
}

func (s *ProfileSimulationService) Events(afterSequence int64, limit int) (contracts.SimulationEventsResponse, error) {
	return s.inner.Events(afterSequence, limit)
}

func (s *ProfileSimulationService) EventsFiltered(afterSequence int64, limit int, filters SimulationEventFilters) (contracts.SimulationEventsResponse, error) {
	return s.inner.EventsFiltered(afterSequence, limit, filters)
}

func (s *ProfileSimulationService) Subscribe() (SimulationSubscription, error) {
	return s.inner.Subscribe()
}

func (s *ProfileSimulationService) ListLogs() (contracts.SimulationRunsResponse, error) {
	return s.inner.ListLogs()
}

func (s *ProfileSimulationService) ExportLogs(ctx context.Context, id string, destination io.Writer) error {
	return s.inner.ExportLogs(ctx, id, destination)
}

func (s *ProfileSimulationService) ExportTelemetry(ctx context.Context, id string, destination io.Writer) error {
	return s.inner.ExportTelemetry(ctx, id, destination)
}

func (s *ProfileSimulationService) GetLog(id string) (contracts.SimulationRun, []contracts.SimulationEvent, error) {
	return s.inner.GetLog(id)
}

func (s *ProfileSimulationService) GetLogRun(id string) (contracts.SimulationRun, error) {
	return s.inner.GetLogRun(id)
}

func (s *ProfileSimulationService) GetLogEvents(id string, query SimulationLogEventQuery) (contracts.SimulationEventsResponse, error) {
	return s.inner.GetLogEvents(id, query)
}

func (s *ProfileSimulationService) ImportLogs(logs []contracts.ProfileLog) error {
	return s.inner.ImportLogs(logs)
}

func (s *ProfileSimulationService) FlushPersistence(ctx context.Context) error {
	return s.inner.FlushPersistence(ctx)
}

func (s *ProfileSimulationService) RestorePersisted(ctx context.Context) error {
	if err := s.manager.acquireSimulation(s.profileID); err != nil {
		return err
	}
	if err := s.inner.RestorePersisted(ctx); err != nil {
		s.manager.releaseSimulation(s.profileID)
		return err
	}
	if _, err := s.inner.Snapshot(); err != nil {
		s.manager.releaseSimulation(s.profileID)
	}
	return nil
}

type profileRealtimePublisher struct {
	profileID string
	next      RealtimePublisher
}

func (publisher profileRealtimePublisher) Publish(message contracts.RealtimeWebSocketMessage) {
	if publisher.next == nil {
		return
	}
	id := publisher.profileID
	message.ProfileID = &id
	publisher.next.Publish(message)
}

var _ SimulationDeviceSource = (*DeviceService)(nil)
var _ SimulationGatewaySource = (*GatewayService)(nil)

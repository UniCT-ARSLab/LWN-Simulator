package services

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/gateway"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"lwn-simulator-backend/internal/apperrors"
)

type memoryCheckpointStore struct {
	checkpoint *types.EngineCheckpoint
	loadErr    error
	saveErr    error
	clearErr   error
	cleared    bool
}

func (store *memoryCheckpointStore) Load() (*types.EngineCheckpoint, error) {
	return store.checkpoint, store.loadErr
}

func (store *memoryCheckpointStore) Save(checkpoint types.EngineCheckpoint) error {
	copy := checkpoint
	store.checkpoint = &copy
	return store.saveErr
}

func (store *memoryCheckpointStore) Clear() error {
	store.cleared = true
	store.checkpoint = nil
	return store.clearErr
}

type simulationDeviceSource struct{}

func (simulationDeviceSource) GetDevices(contracts.GetDevicesRequest) (contracts.GetDevicesResponse, error) {
	device := validSimulationDevice("a1c6e32b-4f0d-4b50-9fc5-000000000080")
	device.DevEUI = "70B3D57ED0000080"
	return contracts.GetDevicesResponse{Devices: []contracts.Device{
		device,
	}}, nil
}

type emptySimulationDeviceSource struct{}

func (emptySimulationDeviceSource) GetDevices(contracts.GetDevicesRequest) (contracts.GetDevicesResponse, error) {
	return contracts.GetDevicesResponse{Devices: []contracts.Device{}}, nil
}

type simulationGatewaySource struct{}

func (simulationGatewaySource) GetGateways(contracts.GetGatewaysRequest) (contracts.GetGatewaysResponse, error) {
	return contracts.GetGatewaysResponse{Gateways: []contracts.Gateway{}}, nil
}

func TestSimulationServiceLifecycleAndSingleSession(t *testing.T) {
	service := NewSimulationService(
		simulationDeviceSource{},
		simulationGatewaySource{},
		types.Options{EventBuffer: 32},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	snapshot, err := service.Start(ctx, contracts.SimulationConfig{Speed: 1})
	if err != nil {
		t.Fatalf("start simulation: %v", err)
	}
	if snapshot.State.Status != contracts.SimulationStatusRunning {
		t.Fatalf("unexpected start status: %s", snapshot.State.Status)
	}

	if _, err := service.Start(ctx, contracts.SimulationConfig{Speed: 1}); err == nil {
		t.Fatal("starting a second simulation should fail")
	}

	subscription, err := service.Subscribe()
	if err != nil {
		t.Fatalf("subscribe to simulation: %v", err)
	}
	defer subscription.Close()

	if _, err := service.Pause(); err != nil {
		t.Fatalf("pause simulation: %v", err)
	}
	if update := receiveSimulationUpdateOfType(t, subscription.Updates, contracts.SimulationPaused); update.Event.Type != contracts.SimulationPaused {
		t.Fatalf("unexpected pause event: %s", update.Event.Type)
	}

	if _, err := service.Resume(); err != nil {
		t.Fatalf("resume simulation: %v", err)
	}
	if update := receiveSimulationUpdateOfType(t, subscription.Updates, contracts.SimulationResumed); update.Event.Type != contracts.SimulationResumed {
		t.Fatalf("unexpected resume event: %s", update.Event.Type)
	}

	if _, err := service.Stop(context.Background()); err != nil {
		t.Fatalf("stop simulation: %v", err)
	}
	if snapshot, err := service.Snapshot(); err != nil || snapshot.State.Status != contracts.SimulationStatusStopped {
		t.Fatalf("unexpected stopped snapshot: %+v, %v", snapshot.State.Status, err)
	}
}

type configuredSimulationGateways []contracts.Gateway

func (gateways configuredSimulationGateways) GetGateways(contracts.GetGatewaysRequest) (contracts.GetGatewaysResponse, error) {
	return contracts.GetGatewaysResponse{Gateways: gateways}, nil
}

func TestSimulationServiceRejectsStartWithoutActiveHardware(t *testing.T) {
	inactive := validSimulationGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c81")
	inactive.Active = false
	for _, gateways := range []configuredSimulationGateways{nil, {inactive}} {
		service := NewSimulationService(emptySimulationDeviceSource{}, gateways, types.Options{})
		_, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1})
		if !errors.Is(err, apperrors.ErrInvalid) || !strings.Contains(err.Error(), "active device or gateway") {
			t.Fatalf("start without active hardware returned %v, want invalid request", err)
		}
	}
}

func TestSimulationServiceStartsWithOneGatewayAndNoDevices(t *testing.T) {
	gateway := validSimulationGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c81")
	service := NewSimulationService(emptySimulationDeviceSource{}, configuredSimulationGateways{gateway}, types.Options{})
	started, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := service.Stop(context.Background()); err != nil {
			t.Errorf("stop gateway-only simulation: %v", err)
		}
	})
	if started.State.Status != contracts.SimulationStatusRunning || len(started.Devices) != 0 ||
		len(started.Gateways) != 1 || started.Metrics.ActiveGateways != 1 {
		t.Fatalf("unexpected gateway-only topology: %+v", started)
	}
	time.Sleep(20 * time.Millisecond)
	current, err := service.Snapshot()
	if err != nil || current.State.ElapsedMilliseconds <= started.State.ElapsedMilliseconds || current.Metrics.TotalUplinks != 0 {
		t.Fatalf("unexpected idle gateway snapshot: %+v, %v", current, err)
	}
}

func TestSimulationServiceOutlivesStartRequestContext(t *testing.T) {
	service := NewSimulationService(
		simulationDeviceSource{},
		simulationGatewaySource{},
		types.Options{EventBuffer: 32},
	)

	requestContext, cancelRequest := context.WithCancel(context.Background())
	if _, err := service.Start(requestContext, contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatalf("start simulation: %v", err)
	}
	cancelRequest()

	time.Sleep(20 * time.Millisecond)
	snapshot, err := service.Snapshot()
	if err != nil {
		t.Fatalf("snapshot after request cancellation: %v", err)
	}
	if snapshot.State.Status != contracts.SimulationStatusRunning {
		t.Fatalf("simulation stopped with request context: %s", snapshot.State.Status)
	}

	if _, err := service.Stop(context.Background()); err != nil {
		t.Fatalf("stop simulation: %v", err)
	}
}

func TestSimulationServiceBroadcastsUpdatesToMultipleSubscribers(t *testing.T) {
	service := NewSimulationService(
		simulationDeviceSource{},
		simulationGatewaySource{},
		types.Options{EventBuffer: 32},
	)

	if _, err := service.Start(context.Background(), contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatalf("start simulation: %v", err)
	}
	defer service.Stop(context.Background())
	publisher := &recordingRealtimePublisher{messages: make(chan contracts.RealtimeWebSocketMessage, 8)}
	service.SetRealtimePublisher(publisher)

	first, err := service.Subscribe()
	if err != nil {
		t.Fatalf("subscribe first frontend: %v", err)
	}
	defer first.Close()
	second, err := service.Subscribe()
	if err != nil {
		t.Fatalf("subscribe second frontend: %v", err)
	}
	defer second.Close()

	if _, err := service.Pause(); err != nil {
		t.Fatalf("pause simulation: %v", err)
	}
	firstUpdate := receiveSimulationUpdateOfType(t, first.Updates, contracts.SimulationPaused)
	secondUpdate := receiveSimulationUpdateOfType(t, second.Updates, contracts.SimulationPaused)
	if firstUpdate.Event.Type != contracts.SimulationPaused || secondUpdate.Event.Type != contracts.SimulationPaused {
		t.Fatalf("frontends received different lifecycle updates: %s, %s", firstUpdate.Event.Type, secondUpdate.Event.Type)
	}
	if message := receiveRealtimeSimulationEventOfType(t, publisher.messages, contracts.SimulationPaused); message.Event == nil || message.Event.Type != contracts.SimulationPaused {
		t.Fatalf("shared websocket did not receive pause event: %+v", message)
	}
}

func TestSimulationServicePersistsAndFinalizesSimulationLog(t *testing.T) {
	logStore := NewFileSimulationLogStore(t.TempDir() + "/simulation-logs.json")
	service := NewSimulationService(
		simulationDeviceSource{},
		simulationGatewaySource{},
		types.Options{EventBuffer: 32},
		nil,
		logStore,
	)
	if _, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1, Seed: int64Pointer(42)}); err != nil {
		t.Fatalf("start simulation: %v", err)
	}
	if _, err := service.Stop(t.Context()); err != nil {
		t.Fatalf("stop simulation: %v", err)
	}
	logs, err := service.ListLogs()
	if err != nil || logs.Total != 1 || len(logs.Runs) != 1 {
		t.Fatalf("unexpected simulation logs: %+v, %v", logs, err)
	}
	if logs.Runs[0].Summary.Status != contracts.SimulationRunStatusStopped || logs.Runs[0].Summary.EventCount == 0 {
		t.Fatalf("simulation log was not finalized: %+v", logs.Runs[0].Summary)
	}
	_, events, err := service.GetLog(logs.Runs[0].Summary.Id)
	if err != nil || len(events) == 0 {
		t.Fatalf("simulation events were not persisted: %v", err)
	}
}

func receiveSimulationUpdateOfType(
	t *testing.T,
	updates <-chan SimulationUpdate,
	expected contracts.SimulationEventType,
) SimulationUpdate {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case update := <-updates:
			if update.Event.Type == expected {
				return update
			}
		case <-deadline:
			t.Fatalf("timed out waiting for simulation event %s", expected)
			return SimulationUpdate{}
		}
	}
}

func receiveRealtimeSimulationEventOfType(
	t *testing.T,
	messages <-chan contracts.RealtimeWebSocketMessage,
	expected contracts.SimulationEventType,
) contracts.RealtimeWebSocketMessage {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case message := <-messages:
			if message.Type == contracts.RealtimeSimulationEventMessage && message.Event != nil && message.Event.Type == expected {
				return message
			}
		case <-deadline:
			t.Fatalf("timed out waiting for realtime simulation event %s", expected)
			return contracts.RealtimeWebSocketMessage{}
		}
	}
}

func TestSimulationServiceRestoresAndClearsPersistedLifecycle(t *testing.T) {
	t.Run("restores active checkpoint", func(t *testing.T) {
		store := &memoryCheckpointStore{checkpoint: &types.EngineCheckpoint{
			Config: contracts.SimulationConfig{Speed: 2},
			State:  contracts.SimulationState{Status: contracts.SimulationStatusRunning, Speed: 2, ElapsedMilliseconds: 2500},
		}}
		service := NewSimulationService(simulationDeviceSource{}, simulationGatewaySource{}, types.Options{}, store)
		if err := service.RestorePersisted(t.Context()); err != nil {
			t.Fatalf("restore simulation: %v", err)
		}
		snapshot, err := service.Snapshot()
		if err != nil || snapshot.State.Status != contracts.SimulationStatusRunning || snapshot.State.ElapsedMilliseconds < 2500 {
			t.Fatalf("unexpected restored snapshot: %+v, %v", snapshot.State, err)
		}
		if _, err := service.Stop(context.Background()); err != nil {
			t.Fatalf("stop restored simulation: %v", err)
		}
	})

	t.Run("clears terminal checkpoint", func(t *testing.T) {
		store := &memoryCheckpointStore{checkpoint: &types.EngineCheckpoint{
			Config: contracts.SimulationConfig{Speed: 1},
			State:  contracts.SimulationState{Status: contracts.SimulationStatusStopped, Speed: 1},
		}}
		service := NewSimulationService(simulationDeviceSource{}, simulationGatewaySource{}, types.Options{}, store)
		if err := service.RestorePersisted(t.Context()); err != nil || !store.cleared {
			t.Fatalf("terminal checkpoint was not cleared: cleared=%v err=%v", store.cleared, err)
		}
	})

	t.Run("propagates checkpoint load error", func(t *testing.T) {
		service := NewSimulationService(simulationDeviceSource{}, simulationGatewaySource{}, types.Options{}, &memoryCheckpointStore{loadErr: errors.New("read failed")})
		if err := service.RestorePersisted(t.Context()); err == nil {
			t.Fatal("checkpoint load error should be returned")
		}
	})
}

func TestSimulationServicePersistsEffectiveExternalTransportSpeed(t *testing.T) {
	factory := gateway.NewUDPFactory(gateway.UDPOptions{LocalAddress: "127.0.0.1:0"})
	defer factory.Close()
	logStore := NewFileSimulationLogStore(t.TempDir() + "/simulation-logs.json")
	service := NewSimulationService(simulationDeviceSource{}, simulationGatewaySource{}, types.Options{GatewayAdapterFactory: factory}, nil, logStore)
	defer service.Close()
	snapshot, err := service.Start(t.Context(), contracts.SimulationConfig{
		Speed:         10,
		GatewayBridge: &contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: 1700},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer service.Stop(t.Context())
	if snapshot.State.Speed != 1 {
		t.Fatal("bridge simulation did not start at real-time speed")
	}
	if _, err := service.SetSpeed(10); err == nil {
		t.Fatal("API accepted accelerated external transport")
	}
	logs, err := service.ListLogs()
	if err != nil || len(logs.Runs) != 1 || logs.Runs[0].Summary.Speed != 1 {
		t.Fatalf("logs did not record effective speed: %+v %v", logs, err)
	}
}

func TestConsecutiveSimulationsIsolateBridgeAndPreserveSeed(t *testing.T) {
	factory := gateway.NewUDPFactory(gateway.UDPOptions{LocalAddress: "127.0.0.1:0"})
	defer factory.Close()
	store := NewFileSimulationLogStore(t.TempDir() + "/logs.json")
	service := NewSimulationService(simulationDeviceSource{}, simulationGatewaySource{}, types.Options{GatewayAdapterFactory: factory}, nil, store)
	defer service.Close()
	for _, bridge := range []*contracts.GatewayBridgeConfig{{Enabled: true, Address: "127.0.0.1", Port: 1700}, nil} {
		seed := int64(0)
		if _, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1, Seed: &seed, GatewayBridge: bridge}); err != nil {
			t.Fatal(err)
		}
		if factory.SupportsVirtualGateways() != (bridge != nil) {
			t.Error("gateway transport reflects the previous run")
		}
		if _, err := service.Stop(t.Context()); err != nil {
			t.Fatal(err)
		}
		run, err := store.GetRun(service.currentRunID())
		if err != nil || run.Summary.Seed == nil || *run.Summary.Seed != seed {
			t.Fatalf("archived seed = %v, err = %v", run.Summary.Seed, err)
		}
	}
}

func TestClosedProfilePersistenceReturnsImmediately(t *testing.T) {
	service := NewSimulationService(simulationDeviceSource{}, simulationGatewaySource{}, types.Options{}, nil, NewFileSimulationLogStore(t.TempDir()+"/logs.json"))
	service.Close()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := service.FlushPersistence(ctx); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("flush on deleted profile returned %v", err)
	}
}

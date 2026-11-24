package services

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestFileSimulationCheckpointStoreRoundTripAndAtomicClear(t *testing.T) {
	store := NewFileSimulationCheckpointStore(filepath.Join(t.TempDir(), "simulation-checkpoint.json"))
	seed := int64(42)
	want := types.EngineCheckpoint{
		Config:   contracts.SimulationConfig{Speed: 2, Seed: &seed},
		State:    contracts.SimulationState{Status: contracts.SimulationStatusRunning, Speed: 2, ElapsedMilliseconds: 100},
		Sessions: map[string]types.DeviceSession{"device": {Joined: true, FrameCounterUp: 3}},
	}
	if err := store.Save(want); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if got == nil || got.Config.Speed != want.Config.Speed || got.State.ElapsedMilliseconds != want.State.ElapsedMilliseconds || got.Sessions["device"].FrameCounterUp != 3 {
		t.Fatalf("unexpected checkpoint after round trip: %+v", got)
	}
	if err := store.Clear(); err != nil {
		t.Fatalf("clear checkpoint: %v", err)
	}
	got, err = store.Load()
	if err != nil || got != nil {
		t.Fatalf("expected empty checkpoint after clear, got %v, %v", got, err)
	}
}

func TestCompactCheckpointCanReadLegacyAndRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint.json")
	legacy := []byte(`{"RunID":"run","Config":{"speed":2},"State":{"status":"paused","speed":2},"Sessions":{"device":{"Joined":true,"DeviceAddress":"26011BDA","LastRSSI":0,"PendingUplink":null,"FrameCounterUp":4,"NetworkChannels":{},"PendingDownlinkACK":true,"AppSKey":"AQID"}},"Scheduled":[{"ID":"event","At":0,"DataRate":0,"Confirmed":false}],"Radio":null}`)
	if err := os.WriteFile(path, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewFileSimulationCheckpointStore(path)
	checkpoint, err := store.Load()
	if err != nil || checkpoint == nil {
		t.Fatalf("load legacy checkpoint: %v", err)
	}
	if err := store.Save(*checkpoint); err != nil {
		t.Fatal(err)
	}
	compact, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(compact, []byte(`"LastRSSI"`)) || bytes.Contains(compact, []byte(`"PendingUplink"`)) || bytes.Contains(compact, []byte(`"Confirmed"`)) {
		t.Fatalf("checkpoint still writes redundant values: %s", compact)
	}
	got, err := store.Load()
	if err != nil || !reflect.DeepEqual(checkpoint, got) {
		t.Fatalf("compact checkpoint changed persisted state: %+v, %v", got, err)
	}
}

type recordedSimulationCheckpoints struct {
	*FileSimulationCheckpointStore
	saved []types.EngineCheckpoint
}

func (store *recordedSimulationCheckpoints) Save(checkpoint types.EngineCheckpoint) error {
	store.saved = append(store.saved, checkpoint)
	return store.FileSimulationCheckpointStore.Save(checkpoint)
}

func TestSimulationStopPersistsOnlyATerminalCheckpointMarker(t *testing.T) {
	store := &recordedSimulationCheckpoints{
		FileSimulationCheckpointStore: NewFileSimulationCheckpointStore(filepath.Join(t.TempDir(), "checkpoint.json")),
	}
	device := validSimulationDevice("a1c6e32b-4f0d-4b50-9fc5-000000000081")
	service := NewSimulationService(
		populatedSimulationDeviceSource{devices: []contracts.Device{device}},
		populatedSimulationGatewaySource{},
		types.Options{Clock: types.NewManualClock(0)},
		store,
	)
	defer service.Close()
	if _, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatal(err)
	}
	if err := service.FlushPersistence(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Stop(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(store.saved) < 2 || len(store.saved[0].Sessions) != 1 {
		t.Fatalf("missing recoverable active checkpoint: %+v", store.saved)
	}
	marker := store.saved[len(store.saved)-1]
	if marker.RunID == "" || marker.State.Status != contracts.SimulationStatusStopped || len(marker.Sessions) != 0 || len(marker.Devices) != 0 {
		t.Fatalf("unexpected terminal checkpoint: %+v", marker)
	}
	if got, err := store.Load(); got != nil || err != nil {
		t.Fatalf("checkpoint still exists after completed stop: %+v, %v", got, err)
	}
}

package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	engineTypes "github.com/Giuseppe-Compagnone/lwn-engine/types"
	"lwn-simulator-backend/internal/apperrors"
	"lwn-simulator-backend/internal/database"
	"lwn-simulator-backend/internal/repositories"
)

func TestProfileRuntimeManagerIsolatesProfilesAndAllowsOneSimulation(t *testing.T) {
	dataDir := t.TempDir()
	profiles := NewProfileService(repositories.NewProfileRepository(dataDir), dataDir)
	defaultProfile, err := profiles.EnsureDefaultProfile()
	if err != nil {
		t.Fatal(err)
	}
	secondProfile, err := profiles.Create("Second network")
	if err != nil {
		t.Fatal(err)
	}
	manager := NewProfileRuntimeManager(dataDir, profiles, engineTypes.Options{}, nil)
	first, err := manager.Runtime(defaultProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.Runtime(secondProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if first.Device == second.Device || first.Gateway == second.Gateway || first.Configuration == second.Configuration || first.Simulation == second.Simulation {
		t.Fatal("profiles share runtime services")
	}
	if _, err := first.Configuration.UpdateGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: true,
		Address: "127.0.0.1",
		Port:    1701,
	}); err != nil {
		t.Fatal(err)
	}
	secondBridge, err := second.Configuration.GetGatewayBridge()
	if err != nil {
		t.Fatal(err)
	}
	if secondBridge != defaultGatewayBridgeConfig {
		t.Fatalf("profiles share gateway bridge configuration: %+v", secondBridge)
	}
	firstStorage, err := database.NewProfileStorage(dataDir, defaultProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	secondStorage, err := database.NewProfileStorage(dataDir, secondProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if firstStorage.Directory() == secondStorage.Directory() {
		t.Fatal("profiles share storage")
	}
	if err := first.Device.ImportDevices([]contracts.Device{
		validSimulationDevice("a1c6e32b-4f0d-4b50-9fc5-000000000101"),
	}); err != nil {
		t.Fatalf("seed first profile device: %v", err)
	}
	if err := second.Device.ImportDevices([]contracts.Device{
		validSimulationDevice("a1c6e32b-4f0d-4b50-9fc5-000000000102"),
	}); err != nil {
		t.Fatalf("seed second profile device: %v", err)
	}

	ctx := context.Background()
	if _, err := first.Simulation.Start(ctx, contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Simulation.Start(ctx, contracts.SimulationConfig{Speed: 1}); !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("duplicate start returned %v", err)
	}
	if err := profiles.Delete(defaultProfile.ID); !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("deleting the running profile returned %v", err)
	}
	activity, err := manager.ActiveSimulation()
	if err != nil || !activity.Active || activity.ProfileID == nil || *activity.ProfileID != defaultProfile.ID {
		t.Fatalf("active simulation activity = %+v, err = %v", activity, err)
	}
	if _, err := second.Simulation.Start(ctx, contracts.SimulationConfig{Speed: 1}); err == nil {
		t.Fatal("second profile started while another simulation was active")
	} else if !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("starting a simulation on another profile returned %v, want a conflict", err)
	}
	if _, err := first.Simulation.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	activity, err = manager.ActiveSimulation()
	if err != nil || activity.Active {
		t.Fatalf("simulation remained active after stop: %+v, err = %v", activity, err)
	}
	if _, err := second.Simulation.Start(ctx, contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatalf("second profile could not start after the first stopped: %v", err)
	}
	if _, err := second.Simulation.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err := profiles.Delete(secondProfile.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Simulation.Start(ctx, contracts.SimulationConfig{Speed: 1}); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("start using a deleted profile returned %v", err)
	}
}

func TestProfileRuntimeManagerExportsAndImportsAProfileAsANewProfile(t *testing.T) {
	dataDir := t.TempDir()
	profiles := NewProfileService(repositories.NewProfileRepository(dataDir), dataDir)
	defaultProfile, err := profiles.EnsureDefaultProfile()
	if err != nil {
		t.Fatal(err)
	}
	manager := NewProfileRuntimeManager(dataDir, profiles, engineTypes.Options{}, nil)
	if _, err := manager.Runtime(defaultProfile.ID); err != nil {
		t.Fatal(err)
	}
	runtime, err := manager.Runtime(defaultProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	run := contracts.SimulationRun{
		Summary: contracts.SimulationRunSummary{
			Id:                    "550e8400-e29b-41d4-a716-446655440100",
			StartedAtMilliseconds: 100,
			EndedAtMilliseconds:   int64Pointer(200),
			Status:                contracts.SimulationRunStatusFailed,
			EventCount:            1,
			PacketSuccessRate:     100,
		},
		Objects: []contracts.SimulationLogObject{},
	}
	if err := runtime.Simulation.inner.logStore.Import(run, []contracts.SimulationEvent{{
		ID:       "550e8400-e29b-41d4-a716-446655440101",
		Sequence: 1,
		Type:     contracts.SimulationStarted,
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Configuration.UpdateGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: true,
		Address: "127.0.0.1",
		Port:    1701,
	}); err != nil {
		t.Fatal(err)
	}

	renamed, err := profiles.Rename(defaultProfile.ID, strings.Repeat("界", maxProfileNameLength))
	if err != nil {
		t.Fatal(err)
	}
	archive, err := manager.ExportArchive(defaultProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if archive.Profile != renamed {
		t.Fatalf("exported stale profile: %+v", archive.Profile)
	}
	if len(archive.Logs) != 1 || len(archive.Logs[0].Events) != 1 {
		t.Fatalf("simulation history was not exported: %+v", archive.Logs)
	}
	if archive.GatewayBridge == nil || archive.GatewayBridge.Address != "127.0.0.1" {
		t.Fatalf("gateway bridge configuration was not exported: %+v", archive.GatewayBridge)
	}
	imported, err := manager.ImportArchive(archive)
	if err != nil {
		t.Fatal(err)
	}
	if imported.ID == defaultProfile.ID || imported.Name == defaultProfile.Name {
		t.Fatalf("imported profile was not recreated: %+v", imported)
	}
	importedArchive, err := manager.ExportArchive(imported.ID)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Name != strings.Repeat("界", maxProfileNameLength-len(" (imported 2)"))+" (imported 2)" {
		t.Fatalf("unexpected import name: %q", imported.Name)
	}
	if len(importedArchive.Logs) != 1 || (importedArchive.Logs[0].Run.Summary.Status != run.Summary.Status || importedArchive.Logs[0].Run.Summary.EventCount != 1) {
		t.Fatalf("history changed after round trip: %+v", importedArchive.Logs)
	}
	if importedArchive.Profile.ID != imported.ID {
		t.Fatalf("archive kept the source profile id: %+v", importedArchive.Profile)
	}
	if importedArchive.GatewayBridge == nil || importedArchive.GatewayBridge.Port != 1701 {
		t.Fatalf("gateway bridge configuration was not imported: %+v", importedArchive.GatewayBridge)
	}
}

func TestProfileRuntimeManagerRejectsInvalidArchiveBeforeCreatingProfile(t *testing.T) {
	dataDir := t.TempDir()
	profiles := NewProfileService(repositories.NewProfileRepository(dataDir), dataDir)
	defaultProfile, err := profiles.EnsureDefaultProfile()
	if err != nil {
		t.Fatal(err)
	}
	manager := NewProfileRuntimeManager(dataDir, profiles, engineTypes.Options{}, nil)

	_, err = manager.ImportArchive(contracts.ProfileArchive{
		Format:  profileArchiveFormat,
		Version: profileArchiveVersion,
		Profile: contracts.Profile{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "Invalid import"},
		Devices: []contracts.Device{{ID: "not-a-uuid"}},
	})
	if err == nil {
		t.Fatal("invalid archive was accepted")
	}

	for _, ids := range [][]string{{"not-a-uuid"}, {defaultProfile.ID, defaultProfile.ID}} {
		archive := contracts.ProfileArchive{Format: profileArchiveFormat, Version: profileArchiveVersion, Profile: defaultProfile}
		for _, id := range ids {
			archive.Logs = append(archive.Logs, contracts.ProfileLog{Run: contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: id}}})
		}
		if _, err := manager.ImportArchive(archive); !errors.Is(err, apperrors.ErrInvalid) {
			t.Fatalf("archive with ids %v returned %v", ids, err)
		}
	}
	stored, err := profiles.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].ID != defaultProfile.ID {
		t.Fatalf("invalid import created a profile: %+v", stored)
	}
}

func TestProfileArchiveIgnoresRemovedGatewayMACAddress(t *testing.T) {
	dataDir := t.TempDir()
	profiles := NewProfileService(repositories.NewProfileRepository(dataDir), dataDir)
	if _, err := profiles.EnsureDefaultProfile(); err != nil {
		t.Fatal(err)
	}
	manager := NewProfileRuntimeManager(dataDir, profiles, engineTypes.Options{}, nil)

	var archive contracts.ProfileArchive
	if err := json.Unmarshal([]byte(`{
		"format":"lwn-simulator-profile","version":1,
		"profile":{"id":"550e8400-e29b-41d4-a716-446655440001","name":"Legacy profile"},
		"devices":[],"logs":[],
		"gateways":[{
			"id":"550e8400-e29b-41d4-a716-446655440002","active":true,
			"name":"Legacy gateway","type":"virtual","gatewayEUI":"A840410001000181",
			"macAddress":"obsolete value","keepAlive":30,"latitude":37.5,"longitude":15.1,"altitude":100
		}]
	}`), &archive); err != nil {
		t.Fatal(err)
	}
	profile, err := manager.ImportArchive(archive)
	if err != nil {
		t.Fatalf("old profile failed to import: %v", err)
	}
	exported, err := manager.ExportArchive(profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Devices == nil {
		t.Fatal("missing devices must export as an empty array")
	}
	if len(exported.Gateways) != 1 || exported.Gateways[0].GatewayEUI != "A840410001000181" {
		t.Fatalf("the imported gateway identity was not preserved: %+v", exported.Gateways)
	}
	data, err := json.Marshal(exported)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"macAddress"`) {
		t.Fatalf("the exported profile included the removed field: %s", data)
	}
}

func TestConcurrentStartsChooseOneProfile(t *testing.T) {
	profiles, dir := newTestProfileService(t)
	if _, err := profiles.EnsureDefaultProfile(); err != nil {
		t.Fatal(err)
	}
	manager := NewProfileRuntimeManager(dir, profiles, engineTypes.Options{}, nil)
	start := make(chan struct{})
	results := make(chan error, 2)
	var simulations []*ProfileSimulationService
	for _, name := range []string{"first", "second"} {
		profile, err := profiles.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		runtime, err := manager.Runtime(profile.ID)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(runtime.Simulation.inner.Close)
		if err := runtime.Device.ImportDevices([]contracts.Device{validSimulationDevice("a1c6e32b-4f0d-4b50-9fc5-000000000101")}); err != nil {
			t.Fatal(err)
		}
		simulations = append(simulations, runtime.Simulation)
	}
	for _, simulation := range simulations {
		go func() {
			<-start
			_, err := simulation.Start(t.Context(), contracts.SimulationConfig{Speed: 1})
			results <- err
		}()
	}
	close(start)
	winners := 0
	for range simulations {
		if err := <-results; err == nil {
			winners++
		} else if !errors.Is(err, apperrors.ErrConflict) {
			t.Errorf("unexpected start error: %v", err)
		}
	}
	for _, simulation := range simulations {
		if _, err := simulation.Snapshot(); err == nil {
			if _, err := simulation.Stop(t.Context()); err != nil {
				t.Fatal(err)
			}
		}
	}
	if winners != 1 {
		t.Fatalf("%d simulations started concurrently", winners)
	}
}

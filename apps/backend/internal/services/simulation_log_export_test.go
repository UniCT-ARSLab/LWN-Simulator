package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestSimulationLogExportIncludesCompleteHistoryAndEmptyRuns(t *testing.T) {
	store := NewFileSimulationLogStore(filepath.Join(t.TempDir(), "logs.json"))
	run := contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: "run-1", Status: contracts.SimulationRunStatusStopped}, Objects: []contracts.SimulationLogObject{{Id: "device-1", Name: "Sensor", Kind: contracts.SimulationLogObjectKindDevice}}}
	events := make([]contracts.SimulationEvent, 5007)
	decoded := map[string]interface{}{"temperature": 21.5}
	for index := range events {
		events[index] = contracts.SimulationEvent{Sequence: int64(index + 1), Type: contracts.DeviceUplinkTransmitted, Message: "payload", DecodedPayload: &decoded}
	}
	if err := store.Import(run, events); err != nil {
		t.Fatal(err)
	}
	empty := contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: "run-2"}}
	if err := store.Import(empty, nil); err != nil {
		t.Fatal(err)
	}
	service := NewSimulationService(nil, nil, types.Options{}, nil, store)
	defer service.Close()

	var single bytes.Buffer
	if err := service.ExportLogs(t.Context(), "run-1", &single); err != nil {
		t.Fatal(err)
	}
	var exported contracts.ProfileLog
	if err := json.Unmarshal(single.Bytes(), &exported); err != nil {
		t.Fatal(err)
	}
	if len(exported.Events) != 5007 || exported.Events[5006].Sequence != 5007 || exported.Run.Summary.EventCount != 5007 || exported.Run.Objects[0].Name != "Sensor" || (*exported.Events[5006].DecodedPayload)["temperature"] != 21.5 {
		t.Fatalf("export lost history or metadata: events=%d summary=%+v", len(exported.Events), exported.Run.Summary)
	}
	var all bytes.Buffer
	if err := service.ExportLogs(t.Context(), "", &all); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(all.Bytes()), int64(all.Len()))
	if err != nil || len(archive.File) != 2 {
		t.Fatalf("invalid archive: %v", err)
	}
	counts := map[string]int{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		var log contracts.ProfileLog
		err = json.NewDecoder(reader).Decode(&log)
		reader.Close()
		if err != nil || log.Events == nil {
			t.Fatalf("invalid JSON entry %s: %v", file.Name, err)
		}
		counts[log.Run.Summary.Id] = len(log.Events)
	}
	if counts["run-1"] != 5007 || counts["run-2"] != 0 {
		t.Fatalf("unexpected archive contents: %v", counts)
	}
}

func TestSimulationLogExportFlushesRunningSimulation(t *testing.T) {
	store := NewFileSimulationLogStore(filepath.Join(t.TempDir(), "logs.json"))
	service := NewSimulationService(simulationDeviceSource{}, simulationGatewaySource{}, types.Options{}, nil, store)
	defer service.Close()
	if _, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatal(err)
	}
	defer service.Stop(t.Context())
	var output bytes.Buffer
	if err := service.ExportLogs(t.Context(), service.activeRunID, &output); err != nil {
		t.Fatal(err)
	}
	var exported contracts.ProfileLog
	if err := json.Unmarshal(output.Bytes(), &exported); err != nil || len(exported.Events) == 0 || exported.Events[0].Type != contracts.SimulationStarted {
		t.Fatalf("running simulation events were not flushed: %+v %v", exported, err)
	}
}

func TestSimulationLogExportHandlesMissingEmptyAndCancelledRequests(t *testing.T) {
	service := NewSimulationService(nil, nil, types.Options{})
	defer service.Close()
	if err := service.ExportLogs(t.Context(), "missing", io.Discard); !os.IsNotExist(err) {
		t.Fatalf("missing run was accepted: %v", err)
	}
	var output bytes.Buffer
	if err := service.ExportLogs(t.Context(), "", &output); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil || len(archive.File) != 0 {
		t.Fatalf("expected an empty valid ZIP: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := service.ExportLogs(ctx, "", io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatalf("export ignored cancellation: %v", err)
	}
	if err := service.ExportLogs(t.Context(), "", failingLogExportWriter{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("export hid a write failure: %v", err)
	}
}

type failingLogExportWriter struct{}

func (failingLogExportWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

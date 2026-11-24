package services

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestSimulationTelemetryExportsArchivedAndLegacyRuns(t *testing.T) {
	store := NewFileSimulationLogStore(filepath.Join(t.TempDir(), "logs.json"))
	service := NewSimulationService(nil, nil, types.Options{}, nil, store)
	defer service.Close()
	deviceID, gatewayID := "device-1", "gateway-1"
	started, successful, dropped := contracts.UplinkStarted, contracts.UplinkSucceeded, contracts.UplinkDropped
	retry := int32(2)
	airtime := int64(50)
	rssi, snr := -87.5, 5.25
	collision := contracts.Collision
	gatewayType := contracts.Virtual
	raw := &contracts.RadioFrame{Protocol: contracts.RawLoRa, Frequency: 868100000, SpreadingFactor: 7}
	finalSample := &contracts.SimulationTelemetry{
		ElapsedMilliseconds: 22_000,
		Sequence:            10,
		ConnectedGateways:   1,
		Metrics: contracts.SimulationMetrics{
			TotalUplinks: 3, SuccessfulUplinks: 1, DroppedUplinks: 1, Retransmissions: 1,
			ActiveDevices: 1, Collisions: 1, RadioPacketLosses: 1,
		},
	}
	run := contracts.SimulationRun{
		Summary: contracts.SimulationRunSummary{Id: "recent", Status: contracts.SimulationRunStatusStopped, Speed: 1, DurationMilliseconds: 22_000},
		Objects: []contracts.SimulationLogObject{
			{Id: deviceID, Kind: contracts.SimulationLogObjectKindDevice, Name: "=unsafe", Identifier: "device-eui"},
			{Id: gatewayID, Kind: contracts.SimulationLogObjectKindGateway, Name: "Gateway", Identifier: "gateway-eui", GatewayType: &gatewayType},
		},
	}
	events := []contracts.SimulationEvent{
		{Sequence: 1, Type: contracts.SimulationStarted, Telemetry: &contracts.SimulationTelemetry{Sequence: 1}},
		{Sequence: 2, Type: contracts.DeviceUplinkTransmitted, DeviceID: &deviceID, UplinkOutcome: &started},
		{Sequence: 3, Type: contracts.RadioPacketLost, DeviceID: &deviceID, RadioFrame: raw, RadioLossReason: &collision},
		{Sequence: 4, Type: contracts.DeviceUplinkTransmitted, DeviceID: &deviceID, RetryAttempt: &retry, TimestampMilliseconds: 10_200},
		{Sequence: 5, Type: contracts.MetricsUpdated, DeviceID: &deviceID, UplinkOutcome: &successful, TimestampMilliseconds: 10_250},
		{Sequence: 6, Type: contracts.RadioTransmissionCompleted, RadioFrame: raw, AirtimeMilliseconds: &airtime, TimestampMilliseconds: 10_250},
		{Sequence: 7, Type: contracts.GatewayPacketReceived, GatewayID: &gatewayID, RadioFrame: raw, RSSI: &rssi, SNR: &snr, TimestampMilliseconds: 10_250},
		{Sequence: 8, Type: contracts.DeviceUplinkTransmitted, DeviceID: &deviceID, UplinkOutcome: &started, TimestampMilliseconds: 10_270},
		{Sequence: 9, Type: contracts.PacketDropped, DeviceID: &deviceID, UplinkOutcome: &dropped, TimestampMilliseconds: 10_300},
		{Sequence: 10, Type: contracts.DeviceUplinkTransmitted, DeviceID: &deviceID, UplinkOutcome: &started, TimestampMilliseconds: 22_000, Telemetry: finalSample},
	}
	if err := store.Import(run, events); err != nil {
		t.Fatal(err)
	}
	if err := store.Finalize(run.Summary.Id, run.Summary); err != nil {
		t.Fatal(err)
	}
	legacy := contracts.SimulationRun{Summary: contracts.SimulationRunSummary{
		Id: "legacy", Status: contracts.SimulationRunStatusStopped, PacketSuccessRate: 0.5,
	}}
	if err := store.Import(legacy, []contracts.SimulationEvent{{
		Sequence: 1, Type: contracts.RadioPacketLost, DeviceID: &deviceID, RadioFrame: raw, RadioLossReason: &collision,
	}}); err != nil {
		t.Fatal(err)
	}

	var summary bytes.Buffer
	if err := service.ExportTelemetry(t.Context(), "", &summary); err != nil {
		t.Fatal(err)
	}
	rows := readTelemetryRows(t, &summary)
	if len(rows) != 2 {
		t.Fatalf("summary should include both runs: %+v", rows)
	}
	for _, row := range rows {
		switch row["run_id"] {
		case "recent":
			if row["uplinks_started"] != "3" || row["uplinks_without_outcome"] != "1" || row["metrics_through_sequence"] != "10" {
				t.Fatalf("expected latest persisted sample: %+v", row)
			}
		case "legacy":
			if row["packet_success_rate_pct"] != "50" || row["metrics_source"] != "legacy-summary" || row["uplinks_started"] != "" {
				t.Fatalf("legacy log acquired incorrect metrics: %+v", row)
			}
		}
	}

	archive := exportTelemetryArchive(t, service, "recent")
	if len(archive.File) != 6 {
		t.Fatalf("expected five CSV files and README, got %d", len(archive.File))
	}
	device := readTelemetryEntry(t, archive, "devices.csv")[0]
	if device["name"] != "'=unsafe" || device["uplinks_started"] != "3" || device["uplinks_succeeded"] != "1" ||
		device["uplinks_dropped"] != "1" || device["retransmissions_observed"] != "1" {
		t.Fatalf("unexpected device report: %+v", device)
	}
	timeline := readTelemetryEntry(t, archive, "timeline.csv")
	if len(timeline) != 3 || timeline[0]["radio_losses_observed"] != "1" || timeline[1]["interval_start_ms"] != "10000" ||
		timeline[1]["uplinks_started"] != "1" || timeline[1]["packet_success_rate_pct"] != "50" ||
		timeline[2]["metrics_sample_at_ms"] != "22000" {
		t.Fatalf("unexpected timeline: %+v", timeline)
	}
	radio := readTelemetryEntry(t, archive, "radio.csv")[0]
	if radio["protocol"] != "raw-lora" || radio["collision_losses"] != "1" || radio["airtime_p95_ms"] != "50" ||
		radio["rssi_p50_dbm"] != "-87.5" || radio["snr_samples"] != "1" {
		t.Fatalf("unexpected radio report: %+v", radio)
	}
	gateway := readTelemetryEntry(t, archive, "gateways.csv")[0]
	if gateway["gateway_type"] != "virtual" || gateway["radio_receptions_observed"] != "1" {
		t.Fatalf("unexpected gateway report: %+v", gateway)
	}

	legacyArchive := exportTelemetryArchive(t, service, "legacy")
	legacyTimeline := readTelemetryEntry(t, legacyArchive, "timeline.csv")
	if len(legacyTimeline) != 1 || legacyTimeline[0]["uplinks_started"] != "" || legacyTimeline[0]["radio_losses_observed"] != "1" {
		t.Fatalf("legacy timeline should preserve observable radio losses: %+v", legacyTimeline)
	}
	if err := service.ExportTelemetry(t.Context(), "unknown", io.Discard); !os.IsNotExist(err) {
		t.Fatalf("missing run: %v", err)
	}
}

func TestSimulationTelemetryCapturesActiveAndFinalEngineMetrics(t *testing.T) {
	store := NewFileSimulationLogStore(filepath.Join(t.TempDir(), "logs.json"))
	clock := types.NewManualClock(0)
	device := validSimulationDevice("a1c6e32b-4f0d-4b50-9fc5-000000000081")
	gateway := validSimulationGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c81")
	service := NewSimulationService(
		populatedSimulationDeviceSource{devices: []contracts.Device{device}},
		populatedSimulationGatewaySource{gateways: []contracts.Gateway{gateway}},
		types.Options{Clock: clock}, nil, store,
	)
	defer service.Close()
	if _, err := service.Start(t.Context(), contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatal(err)
	}
	active := exportTelemetryArchive(t, service, service.currentRunID())
	summary := readTelemetryEntry(t, active, "summary.csv")[0]
	if summary["status"] != "running" || summary["metrics_source"] != "sampled" ||
		summary["uplinks_started"] != "0" || summary["packet_success_rate_pct"] != "" {
		t.Fatalf("initial sample should represent an active run with no traffic: %+v", summary)
	}
	if err := clock.Advance(25 * time.Second); err != nil {
		t.Fatal(err)
	}
	waitForSimulationFrame(t, service, device.ID)
	finished, err := service.Stop(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	archive := exportTelemetryArchive(t, service, service.currentRunID())
	summary = readTelemetryEntry(t, archive, "summary.csv")[0]
	if summary["status"] != "stopped" || summary["uplinks_succeeded"] != telemetryInt(finished.Metrics.SuccessfulUplinks) ||
		summary["uplinks_started"] != telemetryInt(finished.Metrics.TotalUplinks) || summary["metrics_sample_at_ms"] != "25000" {
		t.Fatalf("report doesn't match terminal metrics %+v: %+v", finished.Metrics, summary)
	}
	deviceRow := readTelemetryEntry(t, archive, "devices.csv")[0]
	if deviceRow["uplinks_started"] != summary["uplinks_started"] || deviceRow["uplinks_succeeded"] != summary["uplinks_succeeded"] {
		t.Fatalf("device report and terminal metrics disagree: %+v, %+v", deviceRow, summary)
	}
}

func exportTelemetryArchive(t *testing.T, service *SimulationService, runID string) *zip.Reader {
	t.Helper()
	var output bytes.Buffer
	if err := service.ExportTelemetry(t.Context(), runID, &output); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	return archive
}

func readTelemetryEntry(t *testing.T, archive *zip.Reader, name string) []map[string]string {
	t.Helper()
	file, err := archive.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	return readTelemetryRows(t, file)
}

func readTelemetryRows(t *testing.T, input io.Reader) []map[string]string {
	t.Helper()
	data, err := csv.NewReader(input).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("CSV header missing")
	}
	rows := make([]map[string]string, 0, len(data)-1)
	for _, values := range data[1:] {
		row := make(map[string]string, len(data[0]))
		for index, column := range data[0] {
			row[column] = values[index]
		}
		rows = append(rows, row)
	}
	return rows
}

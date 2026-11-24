package services

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestFileSimulationLogStorePersistsRunsAndFiltersByEventAtReadTime(t *testing.T) {
	store := NewFileSimulationLogStore(filepath.Join(t.TempDir(), "logs.json"))
	run := contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: "550e8400-e29b-41d4-a716-446655440000", StartedAtMilliseconds: 10, Status: contracts.SimulationRunStatusRunning, Speed: 1}, Objects: []contracts.SimulationLogObject{{Id: "550e8400-e29b-41d4-a716-446655440001", Name: "Weather station", Kind: contracts.SimulationLogObjectKindDevice, Identifier: "70B3D57ED0001001"}}}
	if err := store.Create(run); err != nil {
		t.Fatal(err)
	}
	deviceID := "550e8400-e29b-41d4-a716-446655440001"
	decoded := map[string]interface{}{"temperature": 21.5, "humidity": 48.0}
	if err := store.Append(run.Summary.Id, contracts.SimulationEvent{ID: "550e8400-e29b-41d4-a716-446655440002", Sequence: 1, Type: contracts.DeviceUplinkTransmitted, DeviceID: &deviceID, Message: "uplink", DecodedPayload: &decoded}); err != nil {
		t.Fatal(err)
	}
	loaded, events, err := store.Get(run.Summary.Id)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Summary.EventCount != 1 || len(events) != 1 || events[0].DeviceID == nil || *events[0].DeviceID != deviceID || events[0].DecodedPayload == nil || (*events[0].DecodedPayload)["temperature"] != 21.5 {
		t.Fatalf("unexpected persisted run: %+v %+v", loaded, events)
	}
	updatedSummary := loaded.Summary
	updatedSummary.PacketSuccessRate = 0.75
	updatedSummary.DurationMilliseconds = 1500
	if err := store.UpdateSummary(run.Summary.Id, updatedSummary); err != nil {
		t.Fatal(err)
	}
	updated, _, err := store.Get(run.Summary.Id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Summary.PacketSuccessRate != 0.75 || updated.Summary.EventCount != 1 {
		t.Fatalf("live summary was not persisted: %+v", updated.Summary)
	}
	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Runs) != 1 {
		t.Fatalf("unexpected list response: %+v", list)
	}
}

func TestFileSimulationLogStorePersistsEventBatchesInSidecar(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.json")
	store := NewFileSimulationLogStore(path)
	runID := "550e8400-e29b-41d4-a716-446655440010"
	run := contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: runID, StartedAtMilliseconds: 10, Status: contracts.SimulationRunStatusRunning, Speed: 1}}
	if err := store.Create(run); err != nil {
		t.Fatal(err)
	}
	events := []contracts.SimulationEvent{
		{ID: "550e8400-e29b-41d4-a716-446655440011", Sequence: 1, Type: contracts.DeviceRegistered, Message: "registered"},
		{ID: "550e8400-e29b-41d4-a716-446655440012", Sequence: 2, Type: contracts.DeviceUplinkTransmitted, Message: "uplink"},
	}
	summary := run.Summary
	summary.PacketSuccessRate = 1
	if err := store.AppendBatch(runID, events, &summary); err != nil {
		t.Fatal(err)
	}

	// Re-open the store to prove that events are durable and not served only
	// by the in-memory cache.
	reopened := NewFileSimulationLogStore(path)
	if err := reopened.AppendBatch(runID, events, nil); err != nil {
		t.Fatal(err)
	}
	loaded, persisted, err := reopened.Get(runID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Summary.EventCount != 2 || loaded.Summary.PacketSuccessRate != 1 || len(persisted) != 2 {
		t.Fatalf("unexpected persisted batch: run=%+v events=%+v", loaded.Summary, persisted)
	}

	page, err := reopened.Events(runID, SimulationLogEventQuery{AfterSequence: 1, Limit: 1, Types: map[contracts.SimulationEventType]struct{}{contracts.DeviceUplinkTransmitted: {}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 1 || page.Events[0].Sequence != 2 || page.LastSequence != 2 {
		t.Fatalf("unexpected event page: %+v", page)
	}

	indexPath, err := reopened.eventIndexPath(runID)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(indexPath); err != nil || info.Size() == 0 {
		t.Fatalf("event index was not persisted: info=%v err=%v", info, err)
	}
}

func TestFileSimulationLogStoreAppendBatchIsIdempotentAcrossRetries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.json")
	runID := "550e8400-e29b-41d4-a716-446655440020"
	run := contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: runID, StartedAtMilliseconds: 10, Status: contracts.SimulationRunStatusRunning}}
	store := NewFileSimulationLogStore(path)
	if err := store.Create(run); err != nil {
		t.Fatal(err)
	}
	events := []contracts.SimulationEvent{
		{ID: "550e8400-e29b-41d4-a716-446655440021", Sequence: 1, Type: contracts.DeviceRegistered, Message: "registered"},
		{ID: "550e8400-e29b-41d4-a716-446655440022", Sequence: 2, Type: contracts.DeviceUplinkTransmitted, Message: "uplink"},
	}
	if err := store.AppendBatch(runID, events, nil); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendBatch(runID, events, nil); err != nil {
		t.Fatal(err)
	}

	reopened := NewFileSimulationLogStore(path)
	loaded, persisted, err := reopened.Get(runID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Summary.EventCount != 2 || len(persisted) != 2 {
		t.Fatalf("retry duplicated events: summary=%+v events=%d", loaded.Summary, len(persisted))
	}
	page, err := reopened.Events(runID, SimulationLogEventQuery{BeforeSequence: 3, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 1 || page.Events[0].Sequence != 2 || page.LastSequence != 2 {
		t.Fatalf("unexpected latest page: %+v", page)
	}
}

func TestFileSimulationLogStorePaginatesAcrossBatchesAndLegacyEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logs.json")
	store := NewFileSimulationLogStore(path)
	const runID = "run"
	// Older profiles can still contain inline events.
	data := `[{"run":{"summary":{"id":"run","eventCount":2}},"events":[{"sequence":1},{"sequence":2}]}]`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	deviceID := "device"
	for _, batch := range [][]int64{{5, 3, 4}, {6, 8}, {10, 11, 12, 13, 14}, {15, 16}} {
		var events []contracts.SimulationEvent
		for _, seq := range batch {
			event := contracts.SimulationEvent{Sequence: seq}
			if seq%2 == 0 {
				event.DeviceID = &deviceID
			}
			events = append(events, event)
		}
		if err := store.AppendBatch(runID, events, nil); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		query SimulationLogEventQuery
		want  []int64
	}{
		{SimulationLogEventQuery{BeforeSequence: 15, Limit: 3}, []int64{12, 13, 14}},
		{SimulationLogEventQuery{BeforeSequence: 13, Limit: 4}, []int64{8, 10, 11, 12}},
		{SimulationLogEventQuery{BeforeSequence: 5, Limit: 5}, []int64{1, 2, 3, 4}},
		{SimulationLogEventQuery{BeforeSequence: 16, Limit: 4, DeviceID: deviceID}, []int64{8, 10, 12, 14}},
		{SimulationLogEventQuery{AfterSequence: 8, Limit: 3}, []int64{10, 11, 12}},
	}
	checkPages := func(t *testing.T, store *FileSimulationLogStore) {
		t.Helper()
		for _, test := range tests {
			result, err := store.Events(runID, test.query)
			if err != nil {
				t.Fatal(err)
			}
			var got []int64
			for _, event := range result.Events {
				got = append(got, event.Sequence)
			}
			if !slices.Equal(got, test.want) || result.LastSequence != 16 {
				t.Fatalf("query %+v: got %v (last %d), want %v (last 16)", test.query, got, result.LastSequence, test.want)
			}
		}
	}
	checkPages(t, NewFileSimulationLogStore(path))
	indexPath, err := store.eventIndexPath(runID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte("incomplete index\n"), 0600); err != nil {
		t.Fatal(err)
	}
	checkPages(t, NewFileSimulationLogStore(path))
}

func TestSimulationEventReaderHasStableEndWhileWriterAppends(t *testing.T) {
	store := NewFileSimulationLogStore(filepath.Join(t.TempDir(), "logs.json"))
	const runID = "run"
	if err := store.Create(contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: runID}}); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(runID, contracts.SimulationEvent{Sequence: 1}); err != nil {
		t.Fatal(err)
	}
	_, reader, err := store.openRunEvents(runID)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.close()

	if err := store.Append(runID, contracts.SimulationEvent{Sequence: 2}); err != nil {
		t.Fatal(err)
	}
	var sequences []int64
	if err := reader.scan(0, reader.size, func(event contracts.SimulationEvent) bool {
		sequences = append(sequences, event.Sequence)
		return false
	}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(sequences, []int64{1}) {
		t.Fatalf("reader included an event appended after it opened: %v", sequences)
	}
	page, err := store.Events(runID, SimulationLogEventQuery{AfterSequence: 1})
	if err != nil || len(page.Events) != 1 || page.Events[0].Sequence != 2 {
		t.Fatalf("next reader did not see the new event: %+v, %v", page, err)
	}
}

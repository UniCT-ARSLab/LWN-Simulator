package services

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type SimulationLogStore interface {
	List() (contracts.SimulationRunsResponse, error)
	GetRun(string) (contracts.SimulationRun, error)
	Get(string) (contracts.SimulationRun, []contracts.SimulationEvent, error)
	Events(string, SimulationLogEventQuery) (contracts.SimulationEventsResponse, error)
	Create(contracts.SimulationRun) error
	Append(string, contracts.SimulationEvent) error
	AppendBatch(string, []contracts.SimulationEvent, *contracts.SimulationRunSummary) error
	UpdateSummary(string, contracts.SimulationRunSummary) error
	Finalize(string, contracts.SimulationRunSummary) error
	Import(contracts.SimulationRun, []contracts.SimulationEvent) error
}

type SimulationLogEventQuery struct {
	AfterSequence  int64
	BeforeSequence int64
	Limit          int
	DeviceID       string
	GatewayID      string
	Types          map[contracts.SimulationEventType]struct{}
}

type persistedSimulationRun struct {
	Run    contracts.SimulationRun     `json:"run"`
	Events []contracts.SimulationEvent `json:"events"`
}

type simulationEventOffset struct {
	Sequence int64 `json:"sequence"`
	Offset   int64 `json:"offset"`
}

const simulationEventIndexStride = 1024

type simulationEventWriter struct {
	writer io.Writer
	offset int64
}

func (writer *simulationEventWriter) Write(data []byte) (int, error) {
	count, err := writer.writer.Write(data)
	writer.offset += int64(count)
	return count, err
}

type FileSimulationLogStore struct {
	path   string
	mu     sync.Mutex
	loaded bool
	runs   []persistedSimulationRun
	// sidecarSequences is populated lazily. It lets retries after a metadata
	// write failure remain idempotent without scanning the whole event stream
	// for every persistence batch.
	sidecarSequences map[string]int64
}

func NewFileSimulationLogStore(path string) *FileSimulationLogStore {
	return &FileSimulationLogStore{path: path, sidecarSequences: make(map[string]int64)}
}

func (store *FileSimulationLogStore) List() (contracts.SimulationRunsResponse, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	runs, err := store.loadLocked()
	if err != nil {
		return contracts.SimulationRunsResponse{}, err
	}
	result := make([]contracts.SimulationRun, len(runs))
	for index, run := range runs {
		result[index] = run.Run
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].Summary.StartedAtMilliseconds > result[right].Summary.StartedAtMilliseconds
	})
	return contracts.SimulationRunsResponse{Runs: result, Total: int32(len(result))}, nil
}

func (store *FileSimulationLogStore) Get(id string) (contracts.SimulationRun, []contracts.SimulationEvent, error) {
	run, reader, err := store.openRunEvents(id)
	if err != nil {
		return contracts.SimulationRun{}, nil, err
	}
	defer reader.close()
	events := append([]contracts.SimulationEvent(nil), run.Events...)
	err = reader.scan(0, reader.size, func(event contracts.SimulationEvent) bool {
		events = append(events, event)
		return false
	})
	if err != nil {
		return contracts.SimulationRun{}, nil, err
	}
	return run.Run, events, nil
}

func (store *FileSimulationLogStore) GetRun(id string) (contracts.SimulationRun, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	runs, err := store.loadLocked()
	if err != nil {
		return contracts.SimulationRun{}, err
	}
	for _, run := range runs {
		if run.Run.Summary.Id == id {
			return run.Run, nil
		}
	}
	return contracts.SimulationRun{}, os.ErrNotExist
}

func simulationLogEventMatches(event contracts.SimulationEvent, query SimulationLogEventQuery) bool {
	if event.Sequence <= query.AfterSequence {
		return false
	}
	if query.BeforeSequence > 0 && event.Sequence >= query.BeforeSequence {
		return false
	}
	if query.DeviceID != "" && (event.DeviceID == nil || *event.DeviceID != query.DeviceID) {
		return false
	}
	if query.GatewayID != "" && (event.GatewayID == nil || *event.GatewayID != query.GatewayID) {
		return false
	}
	if len(query.Types) > 0 {
		if _, ok := query.Types[event.Type]; !ok {
			return false
		}
	}
	return true
}

func (store *FileSimulationLogStore) Create(run contracts.SimulationRun) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	runs, err := store.loadLocked()
	if err != nil {
		return err
	}
	runs = append(runs, persistedSimulationRun{Run: run, Events: []contracts.SimulationEvent{}})
	return store.saveLocked(runs)
}

func (store *FileSimulationLogStore) Import(run contracts.SimulationRun, events []contracts.SimulationEvent) error {
	// AppendBatch counts the archived events as it persists them.
	if len(events) > 0 {
		run.Summary.EventCount = 0
	}
	if err := store.Create(run); err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}
	return store.AppendBatch(run.Summary.Id, events, &run.Summary)
}

func (store *FileSimulationLogStore) Append(id string, event contracts.SimulationEvent) error {
	return store.AppendBatch(id, []contracts.SimulationEvent{event}, nil)
}

// AppendBatch persists a group of events and its latest summary with one
// atomic file replacement. Keeping the decoded runs in memory avoids reading
// and decoding an ever-growing history for every engine event.
func (store *FileSimulationLogStore) AppendBatch(id string, events []contracts.SimulationEvent, summary *contracts.SimulationRunSummary) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	runs, err := store.loadLocked()
	if err != nil {
		return err
	}
	for index := range runs {
		if runs[index].Run.Summary.Id != id {
			continue
		}
		persistedSequence, err := store.lastEventSequenceLocked(id)
		if err != nil {
			return err
		}
		for _, persistedEvent := range runs[index].Events {
			if persistedEvent.Sequence > persistedSequence {
				persistedSequence = persistedEvent.Sequence
			}
		}
		newEvents := make([]contracts.SimulationEvent, 0, len(events))
		seen := make(map[int64]struct{}, len(events))
		for _, event := range events {
			if event.Sequence <= persistedSequence {
				continue
			}
			if _, exists := seen[event.Sequence]; exists {
				continue
			}
			seen[event.Sequence] = struct{}{}
			newEvents = append(newEvents, event)
		}
		sort.Slice(newEvents, func(i, j int) bool { return newEvents[i].Sequence < newEvents[j].Sequence })
		if err := store.appendEventSidecarLocked(id, newEvents); err != nil {
			return err
		}
		telemetry := runs[index].Run.Summary.Telemetry
		if summary != nil {
			telemetry = latestTelemetry(telemetry, summary.Telemetry)
		}
		if len(newEvents) > 0 {
			for _, event := range newEvents {
				telemetry = latestTelemetry(telemetry, event.Telemetry)
				if event.Sequence > persistedSequence {
					persistedSequence = event.Sequence
				}
			}
			store.sidecarSequences[id] = persistedSequence
		}
		eventCount := runs[index].Run.Summary.EventCount + int32(len(newEvents))
		if eventCount < int32(persistedSequence) {
			eventCount = int32(persistedSequence)
		}
		if summary != nil {
			runs[index].Run.Summary = *summary
		}
		runs[index].Run.Summary.EventCount = eventCount
		runs[index].Run.Summary.Telemetry = telemetry
		// Keep the in-memory view consistent even when the metadata replacement
		// fails after the sidecar has already been synced. A retry can then save
		// the metadata without appending the same events again.
		store.runs = runs
		store.loaded = true
		return store.saveLocked(runs)
	}
	return os.ErrNotExist
}

func (store *FileSimulationLogStore) Finalize(id string, summary contracts.SimulationRunSummary) error {
	return store.UpdateSummary(id, summary)
}

func (store *FileSimulationLogStore) UpdateSummary(id string, summary contracts.SimulationRunSummary) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	runs, err := store.loadLocked()
	if err != nil {
		return err
	}
	for index := range runs {
		if runs[index].Run.Summary.Id == id {
			eventCount := runs[index].Run.Summary.EventCount
			if persistedSequence, sequenceErr := store.lastEventSequenceLocked(id); sequenceErr == nil && int32(persistedSequence) > eventCount {
				eventCount = int32(persistedSequence)
			}
			summary.Telemetry = latestTelemetry(runs[index].Run.Summary.Telemetry, summary.Telemetry)
			runs[index].Run.Summary = summary
			runs[index].Run.Summary.EventCount = eventCount
			return store.saveLocked(runs)
		}
	}
	return os.ErrNotExist
}

func (store *FileSimulationLogStore) lastEventSequenceLocked(id string) (int64, error) {
	if sequence, ok := store.sidecarSequences[id]; ok {
		return sequence, nil
	}
	// Index entries point to the first event of each batch, so their sequence
	// is not necessarily the last sequence in the sidecar. Read the tail once
	// after restart and cache it for subsequent batches.
	sequence, err := store.scanLastEventSequenceLocked(id)
	if err != nil {
		return 0, err
	}
	store.sidecarSequences[id] = sequence
	return sequence, nil
}

func (store *FileSimulationLogStore) scanLastEventSequenceLocked(id string) (int64, error) {
	reader, err := store.openEventReader(id)
	if err != nil {
		return 0, err
	}
	defer reader.close()
	offsets := reader.offsets()
	var sequence int64
	err = reader.scan(offsets[len(offsets)-1].Offset, reader.size, func(event contracts.SimulationEvent) bool {
		sequence = max(sequence, event.Sequence)
		return false
	})
	return sequence, err
}

func (store *FileSimulationLogStore) eventDirectory() string {
	extension := filepath.Ext(store.path)
	return strings.TrimSuffix(store.path, extension) + ".events"
}

func (store *FileSimulationLogStore) eventPath(id string) (string, error) {
	if id == "" || filepath.Base(id) != id {
		return "", fmt.Errorf("invalid simulation run id %q", id)
	}
	return filepath.Join(store.eventDirectory(), id+".jsonl"), nil
}

func (store *FileSimulationLogStore) eventIndexPath(id string) (string, error) {
	if id == "" || filepath.Base(id) != id {
		return "", fmt.Errorf("invalid simulation run id %q", id)
	}
	return filepath.Join(store.eventDirectory(), id+".idx"), nil
}

func (store *FileSimulationLogStore) appendEventSidecarLocked(id string, events []contracts.SimulationEvent) error {
	if len(events) == 0 {
		return nil
	}
	path, err := store.eventPath(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create simulation event directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open simulation event log: %w", err)
	}
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("seek simulation event log: %w", err)
	}
	buffer := bufio.NewWriterSize(file, 64*1024)
	writer := &simulationEventWriter{writer: buffer, offset: offset}
	encoder := json.NewEncoder(writer)
	offsets := make([]simulationEventOffset, 0, 1+(len(events)-1)/simulationEventIndexStride)
	for i, event := range events {
		if i%simulationEventIndexStride == 0 {
			offsets = append(offsets, simulationEventOffset{Sequence: event.Sequence, Offset: writer.offset})
		}
		if err := encoder.Encode(event); err != nil {
			return errors.Join(fmt.Errorf("encode simulation event: %w", err), file.Truncate(offset), file.Close())
		}
	}
	if err := buffer.Flush(); err != nil {
		return errors.Join(fmt.Errorf("write simulation event batch: %w", err), file.Truncate(offset), file.Close())
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync simulation event log: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close simulation event log: %w", err)
	}
	if err := store.appendEventOffsetsLocked(id, offsets); err != nil {
		// The event stream is the source of truth. A missing index only makes a
		// historical query slower and must never cause the durable events to be
		// retried and duplicated.
		log.Printf("append simulation event index: %v", err)
	}
	return nil
}

func (store *FileSimulationLogStore) appendEventOffsetsLocked(id string, entries []simulationEventOffset) error {
	path, err := store.eventIndexPath(id)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open simulation event index: %w", err)
	}
	encoder := json.NewEncoder(file)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			_ = file.Close()
			return fmt.Errorf("encode simulation event index: %w", err)
		}
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync simulation event index: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close simulation event index: %w", err)
	}
	return nil
}

func (store *FileSimulationLogStore) loadLocked() ([]persistedSimulationRun, error) {
	if store.loaded {
		return store.runs, nil
	}
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		store.runs = []persistedSimulationRun{}
		store.loaded = true
		return store.runs, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read simulation logs: %w", err)
	}
	var runs []persistedSimulationRun
	if err := json.Unmarshal(data, &runs); err != nil {
		return nil, fmt.Errorf("decode simulation logs: %w", err)
	}
	store.runs = runs
	store.loaded = true
	return store.runs, nil
}

func (store *FileSimulationLogStore) saveLocked(runs []persistedSimulationRun) error {
	data, err := json.Marshal(runs)
	if err != nil {
		return fmt.Errorf("encode simulation logs: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(store.path), 0700); err != nil {
		return fmt.Errorf("create simulation log directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(store.path), ".simulation-logs-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary simulation logs: %w", err)
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, store.path); err != nil {
		return err
	}
	store.runs = runs
	store.loaded = true
	return nil
}

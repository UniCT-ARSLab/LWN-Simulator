package services

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type simulationEventReader struct {
	events    *os.File
	size      int64
	index     *os.File
	indexSize int64
}

func (store *FileSimulationLogStore) openRunEvents(id string) (persistedSimulationRun, simulationEventReader, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	runs, err := store.loadLocked()
	if err != nil {
		return persistedSimulationRun{}, simulationEventReader{}, err
	}
	for _, run := range runs {
		if run.Run.Summary.Id == id {
			reader, err := store.openEventReader(id)
			return run, reader, err
		}
	}
	return persistedSimulationRun{}, simulationEventReader{}, os.ErrNotExist
}

func (store *FileSimulationLogStore) openEventReader(id string) (simulationEventReader, error) {
	path, err := store.eventPath(id)
	if err != nil {
		return simulationEventReader{}, err
	}
	events, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return simulationEventReader{}, nil
	}
	if err != nil {
		return simulationEventReader{}, fmt.Errorf("open simulation event log: %w", err)
	}
	info, err := events.Stat()
	if err != nil {
		_ = events.Close()
		return simulationEventReader{}, fmt.Errorf("stat simulation event log: %w", err)
	}
	// Capture both lengths while the writer lock is held. Reads can safely
	// continue as later batches append beyond these boundaries.
	reader := simulationEventReader{events: events, size: info.Size()}
	indexPath, _ := store.eventIndexPath(id)
	if index, err := os.Open(indexPath); err == nil {
		if info, err := index.Stat(); err == nil {
			reader.index = index
			reader.indexSize = info.Size()
		} else {
			_ = index.Close()
		}
	}
	return reader, nil
}

func (reader simulationEventReader) close() {
	if reader.events != nil {
		_ = reader.events.Close()
	}
	if reader.index != nil {
		_ = reader.index.Close()
	}
}

func (reader simulationEventReader) offsets() []simulationEventOffset {
	origin := []simulationEventOffset{{}}
	if reader.index == nil {
		return origin
	}
	result := origin
	scanner := bufio.NewScanner(io.NewSectionReader(reader.index, 0, reader.indexSize))
	for scanner.Scan() {
		var entry simulationEventOffset
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			return origin
		}
		if entry.Offset == 0 && len(result) == 1 && result[0].Sequence == 0 {
			continue
		}
		last := result[len(result)-1]
		if entry.Offset <= last.Offset || entry.Offset >= reader.size || entry.Sequence <= last.Sequence {
			return origin
		}
		result = append(result, entry)
	}
	if scanner.Err() != nil {
		return origin
	}
	return result
}

func (reader simulationEventReader) scan(start, end int64, visit func(contracts.SimulationEvent) bool) error {
	if reader.events == nil || start >= end {
		return nil
	}
	scanner := bufio.NewScanner(io.NewSectionReader(reader.events, start, end-start))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var event contracts.SimulationEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("decode simulation event: %w", err)
		}
		if visit(event) {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read simulation event log: %w", err)
	}
	return nil
}

func (store *FileSimulationLogStore) Events(id string, query SimulationLogEventQuery) (contracts.SimulationEventsResponse, error) {
	run, reader, err := store.openRunEvents(id)
	if err != nil {
		return contracts.SimulationEventsResponse{}, err
	}
	defer reader.close()

	response := contracts.SimulationEventsResponse{
		Events:       make([]contracts.SimulationEvent, 0, max(0, query.Limit)),
		LastSequence: int64(run.Run.Summary.EventCount),
	}
	offsets := reader.offsets()
	matches := func(event contracts.SimulationEvent) bool {
		response.LastSequence = max(response.LastSequence, event.Sequence)
		return simulationLogEventMatches(event, query)
	}
	if query.BeforeSequence > 0 && query.Limit > 0 {
		start := sort.Search(len(offsets), func(i int) bool { return offsets[i].Sequence >= query.BeforeSequence })
		// Include the first batch at or past the cursor for older files that
		// recorded the first event's sequence before batch sorting was enforced.
		end := reader.size
		if start+1 < len(offsets) {
			end = offsets[start+1].Offset
		}
		for i := min(start, len(offsets)-1); i >= 0 && len(response.Events) < query.Limit; i-- {
			remaining := query.Limit - len(response.Events)
			page := make([]contracts.SimulationEvent, 0, remaining)
			count := 0
			offset := offsets[i].Offset
			err := reader.scan(offset, end, func(event contracts.SimulationEvent) bool {
				if matches(event) {
					if len(page) < remaining {
						page = append(page, event)
					} else {
						page[count%remaining] = event
					}
					count++
				}
				return false
			})
			if err != nil {
				return contracts.SimulationEventsResponse{}, err
			}
			pivot := count % remaining
			if len(page) < remaining {
				pivot = 0
			}
			slices.Reverse(page[:pivot])
			slices.Reverse(page[pivot:])
			response.Events = append(response.Events, page...)
			end = offset
		}
		for i := len(run.Events) - 1; i >= 0 && len(response.Events) < query.Limit; i-- {
			if event := run.Events[i]; matches(event) {
				response.Events = append(response.Events, event)
			}
		}
		slices.Reverse(response.Events)
		return response, nil
	}

	visit := func(event contracts.SimulationEvent) bool {
		if matches(event) {
			response.Events = append(response.Events, event)
		}
		return query.Limit > 0 && len(response.Events) >= query.Limit
	}
	for _, event := range run.Events {
		if visit(event) {
			return response, nil
		}
	}
	start := sort.Search(len(offsets), func(i int) bool { return offsets[i].Sequence > query.AfterSequence })
	if err := reader.scan(offsets[max(0, start-1)].Offset, reader.size, visit); err != nil {
		return contracts.SimulationEventsResponse{}, err
	}
	return response, nil
}

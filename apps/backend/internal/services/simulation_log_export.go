package services

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

// ExportLogs writes a complete run as JSON, or all runs in this profile as a ZIP
// containing one JSON file per run. Pending events are persisted before reading.
func (s *SimulationService) ExportLogs(ctx context.Context, id string, destination io.Writer) (err error) {
	if err := s.FlushPersistence(ctx); err != nil {
		return err
	}
	if id != "" {
		return s.writeLogExport(ctx, id, destination)
	}
	runs, err := s.ListLogs()
	if err != nil {
		return err
	}
	archive := zip.NewWriter(destination)
	defer func() { err = errors.Join(err, archive.Close()) }()
	for _, run := range runs.Runs {
		entry, err := archive.Create("simulation-" + url.PathEscape(run.Summary.Id) + ".json")
		if err != nil {
			return err
		}
		if err := s.writeLogExport(ctx, run.Summary.Id, entry); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func (s *SimulationService) writeLogExport(ctx context.Context, id string, destination io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	run, events, err := s.GetLog(id)
	if err != nil {
		return err
	}
	if events == nil {
		events = []contracts.SimulationEvent{}
	}
	encoder := json.NewEncoder(destination)
	encoder.SetIndent("", "  ")
	return encoder.Encode(contracts.ProfileLog{Run: run, Events: events})
}

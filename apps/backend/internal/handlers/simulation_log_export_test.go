package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"lwn-simulator-backend/internal/repositories"
	"lwn-simulator-backend/internal/services"
)

func TestSimulationLogExportDownloadsAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := services.NewFileSimulationLogStore(filepath.Join(t.TempDir(), "logs.json"))
	if err := store.Import(contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: "run-1"}}, []contracts.SimulationEvent{{Sequence: 1, Message: "received"}}); err != nil {
		t.Fatal(err)
	}
	service := services.NewSimulationService(nil, nil, types.Options{}, nil, store)
	defer service.Close()
	handler := NewSimulationHandler(service, validator.New())
	router := gin.New()
	router.GET("/logs/export", handler.ExportLogs)
	router.GET("/logs/:id/export", handler.ExportLogs)
	for _, path := range []string{"/logs/run-1/export", "/logs/export"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Disposition"), "attachment;") || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("invalid download response: %d %v", response.Code, response.Header())
		}
		if path == "/logs/export" {
			archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
			if err != nil || len(archive.File) != 1 || response.Header().Get("Content-Type") != "application/zip" {
				t.Fatalf("invalid ZIP download: %v", err)
			}
		} else {
			var exported contracts.ProfileLog
			if err := json.Unmarshal(response.Body.Bytes(), &exported); err != nil || len(exported.Events) != 1 || response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("invalid JSON download: %v", err)
			}
		}
	}
	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/logs/missing/export", nil))
	if missing.Code != http.StatusNotFound || strings.Contains(missing.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("missing export returned a download: %d", missing.Code)
	}
}

func TestProfileLogExportsAreIsolated(t *testing.T) {
	dataDir := t.TempDir()
	profiles := services.NewProfileService(repositories.NewProfileRepository(dataDir), dataDir)
	first, err := profiles.EnsureDefaultProfile()
	if err != nil {
		t.Fatal(err)
	}
	second, err := profiles.Create("Other network")
	if err != nil {
		t.Fatal(err)
	}
	manager := services.NewProfileRuntimeManager(dataDir, profiles, types.Options{}, nil)
	for _, profile := range []contracts.Profile{first, second} {
		runtime, err := manager.Runtime(profile.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err := runtime.Simulation.ImportLogs([]contracts.ProfileLog{{Run: contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: profile.ID}}, Events: []contracts.SimulationEvent{{Sequence: 1, Message: profile.Name}}}}); err != nil {
			t.Fatal(err)
		}
	}
	handler := NewProfileScopedHandler(manager, validator.New())
	router := gin.New()
	router.GET("/profiles/:profileID/logs/export", handler.ExportSimulationLogs)
	router.GET("/profiles/:profileID/logs/:id/export", handler.ExportSimulationLogs)
	router.GET("/profiles/:profileID/logs/telemetry", handler.ExportSimulationTelemetry)
	router.GET("/profiles/:profileID/logs/:id/telemetry", handler.ExportSimulationTelemetry)
	for _, profile := range []contracts.Profile{first, second} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/profiles/"+profile.ID+"/logs/export", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("profile export failed: %d %s", response.Code, response.Body.String())
		}
		archive, err := zip.NewReader(bytes.NewReader(response.Body.Bytes()), int64(response.Body.Len()))
		if err != nil || len(archive.File) != 1 {
			t.Fatalf("profile archive contains other runs: %v", err)
		}
		reader, err := archive.File[0].Open()
		if err != nil {
			t.Fatal(err)
		}
		var log contracts.ProfileLog
		err = json.NewDecoder(reader).Decode(&log)
		reader.Close()
		if err != nil || log.Run.Summary.Id != profile.ID || log.Events[0].Message != profile.Name {
			t.Fatalf("profile export leaked another network: %+v %v", log, err)
		}
	}
	for _, download := range []struct {
		path, contentType, filename string
	}{
		{"/profiles/" + first.ID + "/logs/telemetry", "text/csv; charset=utf-8", "simulation-telemetry-summary.csv"},
		{"/profiles/" + first.ID + "/logs/" + first.ID + "/telemetry", "application/zip", "simulation-" + first.ID + "-telemetry.zip"},
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, download.path, nil))
		if response.Code != http.StatusOK ||
			response.Header().Get("Content-Type") != download.contentType ||
			!strings.Contains(response.Header().Get("Content-Disposition"), download.filename) {
			t.Fatalf("invalid telemetry attachment: %d %v", response.Code, response.Header())
		}
		if strings.HasPrefix(download.contentType, "text/csv") &&
			(!strings.Contains(response.Body.String(), first.ID) || strings.Contains(response.Body.String(), second.ID)) {
			t.Fatalf("summary CSV contains runs from another profile: %s", response.Body.String())
		}
	}
	for _, endpoint := range []string{"export", "telemetry"} {
		response := httptest.NewRecorder()
		path := "/profiles/" + first.ID + "/logs/" + second.ID + "/" + endpoint
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("another profile's run was accessible through %s: %d", endpoint, response.Code)
		}
	}
}

type failingSimulationLogExportService struct{ fakeSimulationService }

func (*failingSimulationLogExportService) ExportLogs(_ context.Context, _ string, destination io.Writer) error {
	_, _ = io.WriteString(destination, "partial archive")
	return io.ErrClosedPipe
}

func TestSimulationLogExportDoesNotDownloadPartialFailures(t *testing.T) {
	for _, test := range []struct {
		service SimulationService
		status  int
	}{
		{service: &fakeSimulationService{}, status: http.StatusNotImplemented},
		{service: &failingSimulationLogExportService{}, status: http.StatusInternalServerError},
	} {
		handler := NewSimulationHandler(test.service, validator.New())
		router := gin.New()
		router.GET("/export", handler.ExportLogs)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/export", nil))
		if response.Code != test.status || strings.Contains(response.Body.String(), "partial archive") || response.Header().Get("Content-Disposition") != "" {
			t.Fatalf("failed export returned a partial download: %d %s", response.Code, response.Body.String())
		}
	}
}

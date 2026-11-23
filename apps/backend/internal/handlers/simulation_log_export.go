package handlers

import (
	"context"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type SimulationLogExportService interface {
	ExportLogs(context.Context, string, io.Writer) error
}

func (h *SimulationHandler) ExportLogs(c *gin.Context) {
	service, ok := h.service.(SimulationLogExportService)
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "simulation log export is not configured"})
		return
	}
	id := c.Param("id")
	filename, contentType := "simulation-logs.zip", "application/zip"
	if id != "" {
		filename, contentType = "simulation-"+url.PathEscape(id)+".json", "application/json"
	}
	serveSimulationDownload(c, id, filename, contentType, service.ExportLogs)
}

type simulationTelemetryExportService interface {
	ExportTelemetry(context.Context, string, io.Writer) error
}

func (h *SimulationHandler) ExportTelemetry(c *gin.Context) {
	service, ok := h.service.(simulationTelemetryExportService)
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "simulation telemetry export is not configured"})
		return
	}
	id := c.Param("id")
	filename, contentType := "simulation-telemetry-summary.csv", "text/csv; charset=utf-8"
	if id != "" {
		filename, contentType = "simulation-"+url.PathEscape(id)+"-telemetry.zip", "application/zip"
	}
	serveSimulationDownload(c, id, filename, contentType, service.ExportTelemetry)
}

func serveSimulationDownload(c *gin.Context, id, filename, contentType string, export func(context.Context, string, io.Writer) error) {
	file, err := os.CreateTemp("", "lwn-logs-*")
	if err != nil {
		writeServiceError(c, err)
		return
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := export(c.Request.Context(), id, file); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "simulation log not found"})
			return
		}
		writeServiceError(c, err)
		return
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	c.Header("Cache-Control", "no-store")
	http.ServeContent(c.Writer, c.Request, filename, time.Time{}, file)
}

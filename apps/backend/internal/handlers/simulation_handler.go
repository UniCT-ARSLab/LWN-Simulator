package handlers

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"lwn-simulator-backend/internal/apperrors"
	"lwn-simulator-backend/internal/services"
	"lwn-simulator-backend/internal/telemetry"
	ws "lwn-simulator-backend/internal/websocket"
)

type SimulationService interface {
	Start(context.Context, contracts.SimulationConfig) (contracts.SimulationSnapshot, error)
	SetSpeed(float64) (contracts.SimulationSnapshot, error)
	Pause() (contracts.SimulationSnapshot, error)
	Resume() (contracts.SimulationSnapshot, error)
	Stop(context.Context) (contracts.SimulationSnapshot, error)
	Snapshot() (contracts.SimulationSnapshot, error)
	QueueUplink(contracts.SimulationUplinkRequest) (contracts.SimulationActionResponse, error)
	QueueDownlink(contracts.SimulationDownlinkRequest) (contracts.SimulationActionResponse, error)
	QueueMACCommand(contracts.SimulationMACCommandRequest) (contracts.SimulationActionResponse, error)
	Events(int64, int) (contracts.SimulationEventsResponse, error)
	EventsFiltered(int64, int, services.SimulationEventFilters) (contracts.SimulationEventsResponse, error)
	Subscribe() (services.SimulationSubscription, error)
}

type SimulationLogService interface {
	ListLogs() (contracts.SimulationRunsResponse, error)
	GetLogRun(string) (contracts.SimulationRun, error)
	GetLogEvents(string, services.SimulationLogEventQuery) (contracts.SimulationEventsResponse, error)
}

const maximumSimulationLogLimit = 1000

type SimulationHandler struct {
	service   SimulationService
	validator *validator.Validate
	websocket *ws.Server
	profileID string
}

func NewSimulationHandler(service SimulationService, validator *validator.Validate) *SimulationHandler {
	return &SimulationHandler{
		service:   service,
		validator: validator,
		websocket: ws.NewServer(allowWebSocketOrigin),
	}
}

func NewSimulationHandlerForProfile(service SimulationService, validator *validator.Validate, profileID string) *SimulationHandler {
	handler := NewSimulationHandler(service, validator)
	handler.profileID = profileID
	return handler
}

func allowWebSocketOrigin(request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}

	return strings.EqualFold(parsed.Hostname(), "localhost") ||
		parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
}

func (h *SimulationHandler) Start(c *gin.Context) {
	var req contracts.SimulationConfig
	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	res, err := h.service.Start(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *SimulationHandler) SetSpeed(c *gin.Context) {
	var req contracts.SimulationSpeedRequest
	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	res, err := h.service.SetSpeed(req.Speed)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *SimulationHandler) Pause(c *gin.Context) { h.control(c, h.service.Pause) }

func (h *SimulationHandler) Resume(c *gin.Context) { h.control(c, h.service.Resume) }

func (h *SimulationHandler) Stop(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	res, err := h.service.Stop(ctx)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *SimulationHandler) Snapshot(c *gin.Context) {
	res, err := h.service.Snapshot()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *SimulationHandler) QueueUplink(c *gin.Context) {
	var req contracts.SimulationUplinkRequest
	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	response, err := h.service.QueueUplink(req)
	h.writeAcceptedAction(c, response, err)
}

func (h *SimulationHandler) QueueDownlink(c *gin.Context) {
	var req contracts.SimulationDownlinkRequest
	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	response, err := h.service.QueueDownlink(req)
	h.writeAcceptedAction(c, response, err)
}

func (h *SimulationHandler) QueueMACCommand(c *gin.Context) {
	var req contracts.SimulationMACCommandRequest
	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	response, err := h.service.QueueMACCommand(req)
	h.writeAcceptedAction(c, response, err)
}

func (h *SimulationHandler) Events(c *gin.Context) {
	afterSequence, ok := parseNonNegativeQuery(c, "afterSequence", 0)
	if !ok {
		return
	}
	limit, ok := parseNonNegativeQuery(c, "limit", 0)
	if !ok {
		return
	}
	filters := services.SimulationEventFilters{
		DeviceID:  strings.TrimSpace(c.Query("deviceID")),
		GatewayID: strings.TrimSpace(c.Query("gatewayID")),
	}
	if typeQuery := strings.TrimSpace(c.Query("type")); typeQuery != "" {
		filters.Types = make(map[contracts.SimulationEventType]struct{})
		for _, value := range strings.Split(typeQuery, ",") {
			eventType := contracts.SimulationEventType(strings.TrimSpace(value))
			if !eventType.Valid() {
				writeServiceError(c, apperrors.Invalid("invalid event type %q", value))
				return
			}
			filters.Types[eventType] = struct{}{}
		}
	}
	res, err := h.service.EventsFiltered(afterSequence, int(limit), filters)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *SimulationHandler) Metrics(c *gin.Context) {
	snapshot, err := h.service.Snapshot()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", telemetry.Prometheus(snapshot))
}

func (h *SimulationHandler) Logs(c *gin.Context) {
	service, ok := h.service.(SimulationLogService)
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "simulation logs are not configured"})
		return
	}
	res, err := service.ListLogs()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *SimulationHandler) Log(c *gin.Context) {
	service, ok := h.service.(SimulationLogService)
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "simulation logs are not configured"})
		return
	}
	run, err := service.GetLogRun(c.Param("id"))
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "simulation log not found"})
			return
		}
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, run)
}

func (h *SimulationHandler) LogEvents(c *gin.Context) {
	service, ok := h.service.(SimulationLogService)
	if !ok {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "simulation logs are not configured"})
		return
	}
	afterSequence, ok := parseNonNegativeQuery(c, "afterSequence", 0)
	if !ok {
		return
	}
	beforeSequence, ok := parseNonNegativeQuery(c, "beforeSequence", 0)
	if !ok {
		return
	}
	limit, ok := parseNonNegativeQuery(c, "limit", maximumSimulationLogLimit)
	if !ok {
		return
	}
	if limit == 0 || limit > maximumSimulationLogLimit {
		writeServiceError(c, apperrors.Invalid("limit must be between 1 and %d", maximumSimulationLogLimit))
		return
	}
	deviceID := strings.TrimSpace(c.Query("deviceID"))
	gatewayID := strings.TrimSpace(c.Query("gatewayID"))
	types := map[contracts.SimulationEventType]struct{}{}
	if typeQuery := strings.TrimSpace(c.Query("type")); typeQuery != "" {
		for _, value := range strings.Split(typeQuery, ",") {
			eventType := contracts.SimulationEventType(strings.TrimSpace(value))
			if !eventType.Valid() {
				writeServiceError(c, apperrors.Invalid("invalid event type %q", value))
				return
			}
			types[eventType] = struct{}{}
		}
	}
	response, err := service.GetLogEvents(c.Param("id"), services.SimulationLogEventQuery{
		AfterSequence:  afterSequence,
		BeforeSequence: beforeSequence,
		Limit:          int(limit),
		DeviceID:       deviceID,
		GatewayID:      gatewayID,
		Types:          types,
	})
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "simulation log not found"})
			return
		}
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *SimulationHandler) WebSocket(c *gin.Context) {
	connection, err := h.websocket.Upgrade(c.Writer, c.Request)
	if err != nil {
		return
	}
	defer connection.Close()

	subscription, err := h.service.Subscribe()
	if err != nil {
		_ = connection.SendJSONAndClose(contracts.SimulationWebSocketMessage{
			Type:                  contracts.SimulationErrorMessage,
			TimestampMilliseconds: time.Now().UnixMilli(),
			ProfileID:             stringPointerOrNil(h.profileID),
			Error:                 stringPointer(err.Error()),
		})
		return
	}
	defer subscription.Close()

	if snapshot, snapshotErr := h.service.Snapshot(); snapshotErr == nil {
		if err := connection.SendJSON(contracts.SimulationWebSocketMessage{
			Type:                  contracts.SimulationSnapshotMessage,
			TimestampMilliseconds: time.Now().UnixMilli(),
			ProfileID:             stringPointerOrNil(h.profileID),
			Snapshot:              &snapshot,
		}); err != nil {
			return
		}
	}

	done := connection.Done()
	for {
		select {
		case <-done:
			return
		case update, ok := <-subscription.Updates:
			if !ok {
				return
			}
			event := update.Event
			message := contracts.SimulationWebSocketMessage{
				Type:                  contracts.SimulationEventMessage,
				TimestampMilliseconds: time.Now().UnixMilli(),
				ProfileID:             stringPointerOrNil(h.profileID),
				Event:                 &event,
			}
			if update.IncludeSnapshot {
				message.Snapshot = &update.Snapshot
			}
			if err := connection.SendJSON(message); err != nil {
				return
			}
		}
	}
}

func (h *SimulationHandler) control(
	c *gin.Context,
	action func() (contracts.SimulationSnapshot, error),
) {
	res, err := action()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *SimulationHandler) writeAcceptedAction(
	c *gin.Context,
	response contracts.SimulationActionResponse,
	err error,
) {
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, response)
}

func parseNonNegativeQuery(c *gin.Context, name string, fallback int64) (int64, bool) {
	raw := c.Query(name)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": name + " must be a non-negative integer"})
		return 0, false
	}
	return value, true
}

func stringPointer(value string) *string { return &value }

func stringPointerOrNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

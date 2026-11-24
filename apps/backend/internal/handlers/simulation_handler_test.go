package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
	"lwn-simulator-backend/internal/apperrors"
	"lwn-simulator-backend/internal/services"
)

type fakeSimulationService struct {
	snapshot  contracts.SimulationSnapshot
	updates   chan services.SimulationUpdate
	action    contracts.SimulationActionResponse
	events    contracts.SimulationEventsResponse
	err       error
	logRun    contracts.SimulationRun
	logEvents []contracts.SimulationEvent
}

func (service *fakeSimulationService) ListLogs() (contracts.SimulationRunsResponse, error) {
	return contracts.SimulationRunsResponse{Runs: []contracts.SimulationRun{service.logRun}, Total: 1}, service.err
}

func (service *fakeSimulationService) GetLogRun(string) (contracts.SimulationRun, error) {
	return service.logRun, service.err
}

func (service *fakeSimulationService) GetLogEvents(_ string, query services.SimulationLogEventQuery) (contracts.SimulationEventsResponse, error) {
	if service.err != nil {
		return contracts.SimulationEventsResponse{}, service.err
	}
	filtered := make([]contracts.SimulationEvent, 0)
	for _, event := range service.logEvents {
		if event.Sequence <= query.AfterSequence || (query.DeviceID != "" && stringValue(event.DeviceID) != query.DeviceID) || (query.GatewayID != "" && stringValue(event.GatewayID) != query.GatewayID) {
			continue
		}
		if len(query.Types) > 0 {
			if _, exists := query.Types[event.Type]; !exists {
				continue
			}
		}
		filtered = append(filtered, event)
		if query.Limit > 0 && len(filtered) >= query.Limit {
			break
		}
	}
	return contracts.SimulationEventsResponse{Events: filtered, LastSequence: int64(service.logRun.Summary.EventCount)}, nil
}

func TestSimulationHandlerRuntimeEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeSimulationService{
		action: contracts.SimulationActionResponse{ID: "550e8400-e29b-41d4-a716-446655440000"},
		events: contracts.SimulationEventsResponse{Events: []contracts.SimulationEvent{}, LastSequence: 4},
		snapshot: contracts.SimulationSnapshot{
			State: contracts.SimulationState{Status: contracts.SimulationStatusRunning, Speed: 1},
		},
	}
	handler := NewSimulationHandler(service, validator.New())
	router := gin.New()
	router.POST("/uplinks", handler.QueueUplink)
	router.POST("/downlinks", handler.QueueDownlink)
	router.POST("/mac-commands", handler.QueueMACCommand)
	router.GET("/events", handler.Events)
	router.GET("/metrics", handler.Metrics)

	uplink := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/uplinks", bytes.NewBufferString(`{"deviceID":"550e8400-e29b-41d4-a716-446655440000"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(uplink, request)
	if uplink.Code != http.StatusAccepted || !strings.Contains(uplink.Body.String(), service.action.ID) {
		t.Fatalf("unexpected uplink response: %d %s", uplink.Code, uplink.Body.String())
	}
	service.err = apperrors.Invalid("invalid runtime action")
	rejected := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/uplinks", bytes.NewBufferString(`{"deviceID":"550e8400-e29b-41d4-a716-446655440000"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rejected, request)
	if rejected.Code != http.StatusBadRequest {
		t.Fatalf("invalid runtime action status = %d, want 400", rejected.Code)
	}
	service.err = nil
	for path, body := range map[string]string{
		"/downlinks":    `{"deviceID":"550e8400-e29b-41d4-a716-446655440000","payload":"ZG93bmxpbms="}`,
		"/mac-commands": `{"deviceID":"550e8400-e29b-41d4-a716-446655440000","command":{"type":"dev-status-req"}}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusAccepted || !strings.Contains(recorder.Body.String(), service.action.ID) {
			t.Fatalf("unexpected %s response: %d %s", path, recorder.Code, recorder.Body.String())
		}
	}

	invalidEvents := httptest.NewRecorder()
	router.ServeHTTP(invalidEvents, httptest.NewRequest(http.MethodGet, "/events?limit=-1", nil))
	if invalidEvents.Code != http.StatusBadRequest {
		t.Fatalf("invalid event query status = %d, want 400", invalidEvents.Code)
	}

	events := httptest.NewRecorder()
	router.ServeHTTP(events, httptest.NewRequest(http.MethodGet, "/events?afterSequence=2&limit=10", nil))
	if events.Code != http.StatusOK || !strings.Contains(events.Body.String(), `"lastSequence":4`) {
		t.Fatalf("unexpected events response: %d %s", events.Code, events.Body.String())
	}
	invalidType := httptest.NewRecorder()
	router.ServeHTTP(invalidType, httptest.NewRequest(http.MethodGet, "/events?type=not-an-event", nil))
	if invalidType.Code != http.StatusBadRequest {
		t.Fatalf("invalid event type status = %d, want 400", invalidType.Code)
	}

	metrics := httptest.NewRecorder()
	router.ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metrics.Code != http.StatusOK || !strings.Contains(metrics.Body.String(), "lwn_simulation_running 1") {
		t.Fatalf("unexpected metrics response: %d %s", metrics.Code, metrics.Body.String())
	}
}

func (service *fakeSimulationService) Start(context.Context, contracts.SimulationConfig) (contracts.SimulationSnapshot, error) {
	return service.snapshot, service.err
}

func (service *fakeSimulationService) SetSpeed(float64) (contracts.SimulationSnapshot, error) {
	return service.snapshot, service.err
}

func (service *fakeSimulationService) Pause() (contracts.SimulationSnapshot, error) {
	return service.snapshot, service.err
}

func (service *fakeSimulationService) Resume() (contracts.SimulationSnapshot, error) {
	return service.snapshot, service.err
}

func (service *fakeSimulationService) Stop(context.Context) (contracts.SimulationSnapshot, error) {
	return service.snapshot, service.err
}

func (service *fakeSimulationService) Snapshot() (contracts.SimulationSnapshot, error) {
	return service.snapshot, service.err
}

func TestSimulationHandlerLifecycleEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeSimulationService{snapshot: contracts.SimulationSnapshot{
		State: contracts.SimulationState{Status: contracts.SimulationStatusRunning, Speed: 1},
	}}
	handler := NewSimulationHandler(service, validator.New())
	router := gin.New()
	router.POST("/start", handler.Start)
	router.POST("/pause", handler.Pause)
	router.POST("/resume", handler.Resume)
	router.POST("/stop", handler.Stop)
	router.GET("/snapshot", handler.Snapshot)

	requests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/start", `{"speed":1}`},
		{http.MethodPost, "/pause", ""},
		{http.MethodPost, "/resume", ""},
		{http.MethodPost, "/stop", ""},
		{http.MethodGet, "/snapshot", ""},
	}
	for _, item := range requests {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(item.method, item.path, strings.NewReader(item.body))
		if item.body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Errorf("%s %s status = %d, body=%s", item.method, item.path, recorder.Code, recorder.Body.String())
		}
	}

	invalid := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/start", strings.NewReader(`{"speed":0}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(invalid, request)
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid start status = %d, want 400", invalid.Code)
	}
}

func TestSimulationHandlerLogEndpointsFilterHistoricalEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deviceID := "550e8400-e29b-41d4-a716-446655440001"
	service := &fakeSimulationService{
		logRun: contracts.SimulationRun{Summary: contracts.SimulationRunSummary{Id: "550e8400-e29b-41d4-a716-446655440000", EventCount: 2}},
		logEvents: []contracts.SimulationEvent{
			{ID: "550e8400-e29b-41d4-a716-446655440002", Sequence: 1, Type: contracts.DeviceUplinkTransmitted, DeviceID: &deviceID, Message: "uplink"},
			{ID: "550e8400-e29b-41d4-a716-446655440003", Sequence: 2, Type: contracts.SimulationPaused, Message: "paused"},
		},
	}
	handler := NewSimulationHandler(service, validator.New())
	router := gin.New()
	router.GET("/logs", handler.Logs)
	router.GET("/logs/:id", handler.Log)
	router.GET("/logs/:id/events", handler.LogEvents)

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/logs", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), service.logRun.Summary.Id) {
		t.Fatalf("unexpected logs response: %d %s", list.Code, list.Body.String())
	}
	filtered := httptest.NewRecorder()
	router.ServeHTTP(filtered, httptest.NewRequest(http.MethodGet, "/logs/550e8400-e29b-41d4-a716-446655440000/events?deviceID="+deviceID, nil))
	if filtered.Code != http.StatusOK || !strings.Contains(filtered.Body.String(), "uplink") || strings.Contains(filtered.Body.String(), "paused") {
		t.Fatalf("unexpected filtered events response: %d %s", filtered.Code, filtered.Body.String())
	}
	tooMany := httptest.NewRecorder()
	router.ServeHTTP(tooMany, httptest.NewRequest(http.MethodGet, "/logs/550e8400-e29b-41d4-a716-446655440000/events?limit=1001", nil))
	if tooMany.Code != http.StatusBadRequest {
		t.Fatalf("oversized log page status = %d, want 400", tooMany.Code)
	}
	negativeBefore := httptest.NewRecorder()
	router.ServeHTTP(negativeBefore, httptest.NewRequest(http.MethodGet, "/logs/550e8400-e29b-41d4-a716-446655440000/events?beforeSequence=-1", nil))
	if negativeBefore.Code != http.StatusBadRequest {
		t.Fatalf("negative beforeSequence status = %d, want 400", negativeBefore.Code)
	}
	detail := httptest.NewRecorder()
	router.ServeHTTP(detail, httptest.NewRequest(http.MethodGet, "/logs/550e8400-e29b-41d4-a716-446655440000", nil))
	if detail.Code != http.StatusOK {
		t.Fatalf("unexpected log detail status: %d", detail.Code)
	}
}

func (service *fakeSimulationService) Subscribe() (services.SimulationSubscription, error) {
	return services.SimulationSubscription{Updates: service.updates, Close: func() {}}, nil
}

func (service *fakeSimulationService) QueueUplink(contracts.SimulationUplinkRequest) (contracts.SimulationActionResponse, error) {
	return service.action, service.err
}

func (service *fakeSimulationService) QueueDownlink(contracts.SimulationDownlinkRequest) (contracts.SimulationActionResponse, error) {
	return service.action, service.err
}

func (service *fakeSimulationService) QueueMACCommand(contracts.SimulationMACCommandRequest) (contracts.SimulationActionResponse, error) {
	return service.action, service.err
}

func (service *fakeSimulationService) Events(int64, int) (contracts.SimulationEventsResponse, error) {
	return service.events, service.err
}

func (service *fakeSimulationService) EventsFiltered(int64, int, services.SimulationEventFilters) (contracts.SimulationEventsResponse, error) {
	return service.events, service.err
}

func TestSimulationHandlerWebSocketStreamsSnapshotsAndEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &fakeSimulationService{
		snapshot: contracts.SimulationSnapshot{
			State: contracts.SimulationState{Status: contracts.SimulationStatusRunning, Speed: 1},
		},
		updates: make(chan services.SimulationUpdate, 1),
	}
	handler := NewSimulationHandler(service, validator.New())
	router := gin.New()
	router.GET("/ws", handler.WebSocket)
	server := httptest.NewServer(router)
	defer server.Close()

	connection, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):]+"/ws", nil)
	if err != nil {
		t.Fatalf("dial simulation websocket: %v", err)
	}
	defer connection.Close()

	var initial contracts.SimulationWebSocketMessage
	if err := connection.ReadJSON(&initial); err != nil {
		t.Fatalf("read initial snapshot: %v", err)
	}
	if initial.Type != contracts.SimulationSnapshotMessage || initial.Snapshot == nil {
		t.Fatalf("unexpected initial websocket message: %+v", initial)
	}

	service.updates <- services.SimulationUpdate{
		Snapshot: service.snapshot,
		Event: contracts.SimulationEvent{
			ID: "event-1", Sequence: 1, Type: contracts.SimulationPaused, Message: "paused",
		},
	}
	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	var event contracts.SimulationWebSocketMessage
	if err := connection.ReadJSON(&event); err != nil {
		t.Fatalf("read event update: %v", err)
	}
	if event.Type != contracts.SimulationEventMessage || event.Event == nil || event.Event.Type != contracts.SimulationPaused {
		t.Fatalf("unexpected event websocket message: %+v", event)
	}
}

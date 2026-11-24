package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"lwn-simulator-backend/internal/apperrors"
)

type simulationActivityServiceStub struct {
	activity contracts.SimulationActivity
	err      error
}

func (service simulationActivityServiceStub) ActiveSimulation() (contracts.SimulationActivity, error) {
	return service.activity, service.err
}

func TestSimulationActivityHandlerReturnsBackendWideActivity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	profileID := "550e8400-e29b-41d4-a716-446655440000"
	handler := NewSimulationActivityHandler(simulationActivityServiceStub{
		activity: contracts.SimulationActivity{
			Active:    true,
			ProfileID: &profileID,
			Status:    contracts.SimulationStatusRunning,
		},
	})
	router := gin.New()
	router.GET("/activity", handler.Get)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/activity", nil))

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), profileID) {
		t.Fatalf("unexpected activity response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestSimulationActivityHandlerMapsServiceErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSimulationActivityHandler(simulationActivityServiceStub{
		err: apperrors.Conflict("a simulation is already active"),
	})
	router := gin.New()
	router.GET("/activity", handler.Get)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/activity", nil))

	if recorder.Code != http.StatusConflict {
		t.Fatalf("activity error status = %d, want %d", recorder.Code, http.StatusConflict)
	}
}

func TestSimulationActivityHandlerReturnsInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewSimulationActivityHandler(simulationActivityServiceStub{
		err: errors.New("storage unavailable"),
	})
	router := gin.New()
	router.GET("/activity", handler.Get)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/activity", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("activity error status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

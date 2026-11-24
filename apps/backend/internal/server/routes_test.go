package server

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	engineTypes "github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/gin-gonic/gin"
	"lwn-simulator-backend/internal/database"
	"lwn-simulator-backend/internal/services"
)

func TestRegisterRoutesIncludesSimulationRuntimeAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerRoutes(router, "8080", Services{})
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{
		"GET /api/profile/get-profiles",
		"GET /api/profile/get-profile/:id",
		"POST /api/profile/create-profile",
		"PUT /api/profile/update-profile/:id",
		"DELETE /api/profile/delete-profile/:id",
		"GET /api/profile/export-profile/:id",
		"POST /api/profile/import-profile",
		"POST /api/simulation/start",
		"POST /api/simulation/pause",
		"POST /api/simulation/resume",
		"POST /api/simulation/stop",
		"GET /api/simulation/snapshot",
		"GET /api/simulation/events",
		"GET /api/simulation/logs",
		"GET /api/simulation/logs/export",
		"GET /api/simulation/logs/:id/export",
		"GET /api/simulation/logs/telemetry",
		"GET /api/simulation/logs/:id/telemetry",
		"GET /api/simulation/logs/:id",
		"GET /api/simulation/logs/:id/events",
		"GET /api/simulation/metrics",
		"POST /api/simulation/uplinks",
		"POST /api/simulation/downlinks",
		"POST /api/simulation/mac-commands",
		"GET /api/simulation/ws",
		"GET /api/ws",
	} {
		if _, exists := routes[expected]; !exists {
			t.Errorf("route %q is not registered", expected)
		}
	}
}

func TestRegisterRoutesIncludesProfileConfigurationAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	registerRoutes(router, "8080", Services{
		Profiles: services.NewProfileRuntimeManager(t.TempDir(), nil, engineTypes.Options{}, nil),
	})
	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{
		"GET /api/profiles/:profileID/configuration/gateway-bridge",
		"GET /api/profiles/:profileID/simulation/logs/export",
		"GET /api/profiles/:profileID/simulation/logs/:id/export",
		"GET /api/profiles/:profileID/simulation/logs/telemetry",
		"GET /api/profiles/:profileID/simulation/logs/:id/telemetry",
		"PUT /api/profiles/:profileID/configuration/gateway-bridge",
		"GET /api/simulation/activity",
	} {
		if _, exists := routes[expected]; !exists {
			t.Errorf("route %q is not registered", expected)
		}
	}
}

func TestDeletedDefaultProfileStaysDeletedAcrossRestart(t *testing.T) {
	t.Setenv("LWN_ENV", "development")
	t.Chdir(t.TempDir())
	router, err := New("0")
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		t.Helper()
		result := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(result, req)
		if result.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, result.Code, status, result.Body)
		}
		return result
	}
	res := request("POST", "/api/profile/create-profile", `{"name":"Remaining network"}`, 201)
	var created contracts.CreateProfileResponse
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	request("GET", "/api/device/get-devices", "", 200)
	request("DELETE", "/api/profile/delete-profile/"+database.DefaultProfileID, "", 204)
	for attempt := 0; attempt < 2; attempt++ {
		if attempt == 1 {
			router, err = New("0")
			if err != nil {
				t.Fatal(err)
			}
		}
		request("GET", "/api/device/get-devices", "", 404)
		request("POST", "/api/simulation/start", `{"speed":1}`, 404)
		request("GET", "/api/profiles/"+created.Profile.ID+"/device/get-devices", "", 200)
		if _, err := os.Stat(filepath.Join("dev-db", "profiles", database.DefaultProfileID)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("deleted storage: %v", err)
		}
	}
}

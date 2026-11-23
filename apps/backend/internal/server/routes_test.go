package server

import (
	"testing"

	engineTypes "github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/gin-gonic/gin"
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

package server

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"lwn-simulator-backend/internal/database"
	"lwn-simulator-backend/internal/handlers"
	"lwn-simulator-backend/internal/services"
)

type Services struct {
	Profile  *services.ProfileService
	Profiles *services.ProfileRuntimeManager
	Realtime *handlers.RealtimeHandler
}

func registerRoutes(r *gin.Engine, port string, services Services) {

	validator := validator.New()

	api := r.Group("/api")

	appInfo := api.Group("/app-info")

	appInfoHandler := handlers.NewAppInfoHandler()

	appInfo.GET("/info", appInfoHandler.Info)
	appInfo.GET("/status", appInfoHandler.Status(port))

	profile := api.Group("/profile")
	profileHandler := handlers.NewProfileHandler(services.Profile, validator, services.Profiles)
	profile.GET("/get-profiles", profileHandler.GetProfiles)
	profile.GET("/get-profile/:id", profileHandler.GetProfile)
	profile.POST("/create-profile", profileHandler.CreateProfile)
	profile.PUT("/update-profile/:id", profileHandler.UpdateProfile)
	profile.DELETE("/delete-profile/:id", profileHandler.DeleteProfile)
	profile.GET("/export-profile/:id", profileHandler.ExportProfile)
	profile.POST("/import-profile", profileHandler.ImportProfile)

	api.GET("/ws", services.Realtime.WebSocket)
	api.GET("/simulation/activity", handlers.NewSimulationActivityHandler(services.Profiles).Get)
	handler := handlers.NewProfileScopedHandler(services.Profiles, validator)
	legacy := api.Group("", func(c *gin.Context) { c.AddParam("profileID", database.DefaultProfileID) })
	registerHardwareAndSimulation(legacy, handler)
	routes := api.Group("/profiles/:profileID")
	registerHardwareAndSimulation(routes, handler)
	routes.PATCH("/device/:id/position", handler.UpdateDevicePosition)
	routes.PATCH("/device/:id/active", handler.UpdateDeviceActive)
	routes.PATCH("/device/:id/payload", handler.UpdateDevicePayload)
	routes.PATCH("/gateway/:id/position", handler.UpdateGatewayPosition)
	routes.PATCH("/gateway/:id/active", handler.UpdateGatewayActive)
	routes.GET("/configuration/gateway-bridge", handler.GetGatewayBridge)
	routes.PUT("/configuration/gateway-bridge", handler.UpdateGatewayBridge)
}

func registerHardwareAndSimulation(routes *gin.RouterGroup, handler *handlers.ProfileScopedHandler) {
	routes.POST("/device/create-device", handler.CreateDevice)
	routes.GET("/device/get-device/:id", handler.GetDevice)
	routes.GET("/device/get-devices", handler.GetDevices)
	routes.PUT("/device/update-device/:id", handler.UpdateDevice)
	routes.DELETE("/device/delete-device/:id", handler.DeleteDevice)
	routes.POST("/gateway/create-gateway", handler.CreateGateway)
	routes.GET("/gateway/get-gateway/:id", handler.GetGateway)
	routes.GET("/gateway/get-gateways", handler.GetGateways)
	routes.PUT("/gateway/update-gateway/:id", handler.UpdateGateway)
	routes.DELETE("/gateway/delete-gateway/:id", handler.DeleteGateway)
	routes.POST("/simulation/start", handler.StartSimulation)
	routes.POST("/simulation/speed", handler.SetSimulationSpeed)
	routes.POST("/simulation/pause", handler.PauseSimulation)
	routes.POST("/simulation/resume", handler.ResumeSimulation)
	routes.POST("/simulation/stop", handler.StopSimulation)
	routes.GET("/simulation/snapshot", handler.SimulationSnapshot)
	routes.GET("/simulation/events", handler.SimulationEvents)
	routes.GET("/simulation/logs", handler.SimulationLogs)
	routes.GET("/simulation/logs/export", handler.ExportSimulationLogs)
	routes.GET("/simulation/logs/:id/export", handler.ExportSimulationLogs)
	routes.GET("/simulation/logs/telemetry", handler.ExportSimulationTelemetry)
	routes.GET("/simulation/logs/:id/telemetry", handler.ExportSimulationTelemetry)
	routes.GET("/simulation/logs/:id", handler.SimulationLog)
	routes.GET("/simulation/logs/:id/events", handler.SimulationLogEvents)
	routes.GET("/simulation/metrics", handler.SimulationMetrics)
	routes.POST("/simulation/uplinks", handler.QueueUplink)
	routes.POST("/simulation/downlinks", handler.QueueDownlink)
	routes.POST("/simulation/mac-commands", handler.QueueMACCommand)
	routes.GET("/simulation/ws", handler.SimulationWebSocket)
}

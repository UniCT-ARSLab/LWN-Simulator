package server

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"lwn-simulator-backend/internal/handlers"
	"lwn-simulator-backend/internal/services"
)

type Services struct {
	Device     handlers.DeviceService
	Gateway    handlers.GatewayService
	Profile    *services.ProfileService
	ProfileID  string
	Simulation handlers.SimulationService
	Profiles   *services.ProfileRuntimeManager
	Realtime   *handlers.RealtimeHandler
}

func registerRoutes(r *gin.Engine, port string, services Services) {

	validator := validator.New()

	api := r.Group("/api")

	appInfo := api.Group("/app-info")

	appInfoHandler := handlers.NewAppInfoHandler()

	appInfo.GET("/info", appInfoHandler.Info)
	appInfo.GET("/status", appInfoHandler.Status(port))

	device := api.Group("/device")

	deviceHandler := handlers.NewDeviceHandler(services.Device, validator)

	device.POST("/create-device", deviceHandler.CreateDevice)
	device.GET("/get-device/:id", deviceHandler.GetDevice)
	device.GET("/get-devices", deviceHandler.GetDevices)
	device.PUT("/update-device/:id", deviceHandler.UpdateDevice)
	device.DELETE("/delete-device/:id", deviceHandler.DeleteDevice)

	gateway := api.Group("/gateway")

	gatewayHandler := handlers.NewGatewayHandler(services.Gateway, validator)

	gateway.POST("/create-gateway", gatewayHandler.CreateGateway)
	gateway.GET("/get-gateway/:id", gatewayHandler.GetGateway)
	gateway.GET("/get-gateways", gatewayHandler.GetGateways)
	gateway.PUT("/update-gateway/:id", gatewayHandler.UpdateGateway)
	gateway.DELETE("/delete-gateway/:id", gatewayHandler.DeleteGateway)

	profile := api.Group("/profile")
	profileHandler := handlers.NewProfileHandler(services.Profile, validator, services.Profiles)
	profile.GET("/get-profiles", profileHandler.GetProfiles)
	profile.GET("/get-profile/:id", profileHandler.GetProfile)
	profile.POST("/create-profile", profileHandler.CreateProfile)
	profile.PUT("/update-profile/:id", profileHandler.UpdateProfile)
	profile.DELETE("/delete-profile/:id", profileHandler.DeleteProfile)
	profile.GET("/export-profile/:id", profileHandler.ExportProfile)
	profile.POST("/import-profile", profileHandler.ImportProfile)

	simulation := api.Group("/simulation")
	simulationHandler := handlers.NewSimulationHandlerForProfile(services.Simulation, validator, services.ProfileID)
	if services.Profiles != nil {
		simulation.GET("/activity", handlers.NewSimulationActivityHandler(services.Profiles).Get)
	}
	simulation.POST("/start", simulationHandler.Start)
	simulation.POST("/speed", simulationHandler.SetSpeed)
	simulation.POST("/pause", simulationHandler.Pause)
	simulation.POST("/resume", simulationHandler.Resume)
	simulation.POST("/stop", simulationHandler.Stop)
	simulation.GET("/snapshot", simulationHandler.Snapshot)
	simulation.GET("/events", simulationHandler.Events)
	simulation.GET("/logs", simulationHandler.Logs)
	simulation.GET("/logs/export", simulationHandler.ExportLogs)
	simulation.GET("/logs/:id/export", simulationHandler.ExportLogs)
	simulation.GET("/logs/telemetry", simulationHandler.ExportTelemetry)
	simulation.GET("/logs/:id/telemetry", simulationHandler.ExportTelemetry)
	simulation.GET("/logs/:id", simulationHandler.Log)
	simulation.GET("/logs/:id/events", simulationHandler.LogEvents)
	simulation.GET("/metrics", simulationHandler.Metrics)
	simulation.POST("/uplinks", simulationHandler.QueueUplink)
	simulation.POST("/downlinks", simulationHandler.QueueDownlink)
	simulation.POST("/mac-commands", simulationHandler.QueueMACCommand)
	simulation.GET("/ws", simulationHandler.WebSocket)
	api.GET("/ws", services.Realtime.WebSocket)

	if services.Profiles != nil {
		profileScoped := api.Group("/profiles/:profileID")
		profileScopedHandler := handlers.NewProfileScopedHandler(services.Profiles, validator)

		profileScoped.POST("/device/create-device", profileScopedHandler.CreateDevice)
		profileScoped.GET("/device/get-device/:id", profileScopedHandler.GetDevice)
		profileScoped.GET("/device/get-devices", profileScopedHandler.GetDevices)
		profileScoped.PUT("/device/update-device/:id", profileScopedHandler.UpdateDevice)
		profileScoped.PATCH("/device/:id/position", profileScopedHandler.UpdateDevicePosition)
		profileScoped.PATCH("/device/:id/active", profileScopedHandler.UpdateDeviceActive)
		profileScoped.PATCH("/device/:id/payload", profileScopedHandler.UpdateDevicePayload)
		profileScoped.DELETE("/device/delete-device/:id", profileScopedHandler.DeleteDevice)

		profileScoped.POST("/gateway/create-gateway", profileScopedHandler.CreateGateway)
		profileScoped.GET("/gateway/get-gateway/:id", profileScopedHandler.GetGateway)
		profileScoped.GET("/gateway/get-gateways", profileScopedHandler.GetGateways)
		profileScoped.PUT("/gateway/update-gateway/:id", profileScopedHandler.UpdateGateway)
		profileScoped.PATCH("/gateway/:id/position", profileScopedHandler.UpdateGatewayPosition)
		profileScoped.PATCH("/gateway/:id/active", profileScopedHandler.UpdateGatewayActive)
		profileScoped.DELETE("/gateway/delete-gateway/:id", profileScopedHandler.DeleteGateway)
		profileScoped.GET("/configuration/gateway-bridge", profileScopedHandler.GetGatewayBridge)
		profileScoped.PUT("/configuration/gateway-bridge", profileScopedHandler.UpdateGatewayBridge)

		profileScoped.POST("/simulation/start", profileScopedHandler.StartSimulation)
		profileScoped.POST("/simulation/speed", profileScopedHandler.SetSimulationSpeed)
		profileScoped.POST("/simulation/pause", profileScopedHandler.PauseSimulation)
		profileScoped.POST("/simulation/resume", profileScopedHandler.ResumeSimulation)
		profileScoped.POST("/simulation/stop", profileScopedHandler.StopSimulation)
		profileScoped.GET("/simulation/snapshot", profileScopedHandler.SimulationSnapshot)
		profileScoped.GET("/simulation/events", profileScopedHandler.SimulationEvents)
		profileScoped.GET("/simulation/logs", profileScopedHandler.SimulationLogs)
		profileScoped.GET("/simulation/logs/export", profileScopedHandler.ExportSimulationLogs)
		profileScoped.GET("/simulation/logs/:id/export", profileScopedHandler.ExportSimulationLogs)
		profileScoped.GET("/simulation/logs/telemetry", profileScopedHandler.ExportSimulationTelemetry)
		profileScoped.GET("/simulation/logs/:id/telemetry", profileScopedHandler.ExportSimulationTelemetry)
		profileScoped.GET("/simulation/logs/:id", profileScopedHandler.SimulationLog)
		profileScoped.GET("/simulation/logs/:id/events", profileScopedHandler.SimulationLogEvents)
		profileScoped.GET("/simulation/metrics", profileScopedHandler.SimulationMetrics)
		profileScoped.POST("/simulation/uplinks", profileScopedHandler.QueueUplink)
		profileScoped.POST("/simulation/downlinks", profileScopedHandler.QueueDownlink)
		profileScoped.POST("/simulation/mac-commands", profileScopedHandler.QueueMACCommand)
		profileScoped.GET("/simulation/ws", profileScopedHandler.SimulationWebSocket)
	}
}

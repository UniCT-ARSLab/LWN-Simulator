package handlers

import (
	"net/http"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"lwn-simulator-backend/internal/services"
)

type ProfileRuntimeResolver interface {
	Runtime(string) (*services.ProfileRuntime, error)
}

type ProfileScopedHandler struct {
	resolver  ProfileRuntimeResolver
	validator *validator.Validate
}

func NewProfileScopedHandler(resolver ProfileRuntimeResolver, validator *validator.Validate) *ProfileScopedHandler {
	return &ProfileScopedHandler{resolver: resolver, validator: validator}
}

func (h *ProfileScopedHandler) runtime(c *gin.Context) (*services.ProfileRuntime, bool) {
	runtime, err := h.resolver.Runtime(c.Param("profileID"))
	if err != nil {
		writeServiceError(c, err)
		return nil, false
	}
	return runtime, true
}

func (h *ProfileScopedHandler) withRuntime(
	c *gin.Context,
	action func(*services.ProfileRuntime),
) {
	runtime, ok := h.runtime(c)
	if ok {
		action(runtime)
	}
}

func (h *ProfileScopedHandler) withDeviceHandler(
	c *gin.Context,
	action func(*DeviceHandler),
) {
	h.withRuntime(c, func(runtime *services.ProfileRuntime) {
		action(NewDeviceHandler(runtime.Device, h.validator))
	})
}

func (h *ProfileScopedHandler) withGatewayHandler(
	c *gin.Context,
	action func(*GatewayHandler),
) {
	h.withRuntime(c, func(runtime *services.ProfileRuntime) {
		action(NewGatewayHandler(runtime.Gateway, h.validator))
	})
}

func (h *ProfileScopedHandler) withSimulationHandler(
	c *gin.Context,
	action func(*SimulationHandler),
) {
	h.withRuntime(c, func(runtime *services.ProfileRuntime) {
		action(NewSimulationHandlerForProfile(
			runtime.Simulation,
			h.validator,
			runtime.Profile.ID,
		))
	})
}

func (h *ProfileScopedHandler) CreateDevice(c *gin.Context) {
	h.withDeviceHandler(c, func(handler *DeviceHandler) { handler.CreateDevice(c) })
}

func (h *ProfileScopedHandler) GetDevice(c *gin.Context) {
	h.withDeviceHandler(c, func(handler *DeviceHandler) { handler.GetDevice(c) })
}

func (h *ProfileScopedHandler) GetDevices(c *gin.Context) {
	h.withDeviceHandler(c, func(handler *DeviceHandler) { handler.GetDevices(c) })
}

func (h *ProfileScopedHandler) UpdateDevice(c *gin.Context) {
	h.withDeviceHandler(c, func(handler *DeviceHandler) { handler.UpdateDevice(c) })
}

func (h *ProfileScopedHandler) DeleteDevice(c *gin.Context) {
	h.withDeviceHandler(c, func(handler *DeviceHandler) { handler.DeleteDevice(c) })
}

func (h *ProfileScopedHandler) CreateGateway(c *gin.Context) {
	h.withGatewayHandler(c, func(handler *GatewayHandler) { handler.CreateGateway(c) })
}

func (h *ProfileScopedHandler) GetGateway(c *gin.Context) {
	h.withGatewayHandler(c, func(handler *GatewayHandler) { handler.GetGateway(c) })
}

func (h *ProfileScopedHandler) GetGateways(c *gin.Context) {
	h.withGatewayHandler(c, func(handler *GatewayHandler) { handler.GetGateways(c) })
}

func (h *ProfileScopedHandler) UpdateGateway(c *gin.Context) {
	h.withGatewayHandler(c, func(handler *GatewayHandler) { handler.UpdateGateway(c) })
}

func (h *ProfileScopedHandler) DeleteGateway(c *gin.Context) {
	h.withGatewayHandler(c, func(handler *GatewayHandler) { handler.DeleteGateway(c) })
}

func (h *ProfileScopedHandler) GetGatewayBridge(c *gin.Context) {
	h.withRuntime(c, func(runtime *services.ProfileRuntime) {
		config, err := runtime.Configuration.GetGatewayBridge()
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, config)
	})
}

func (h *ProfileScopedHandler) UpdateGatewayBridge(c *gin.Context) {
	var config contracts.GatewayBridgeConfig
	if !bindJSONAndValidate(c, h.validator, &config) {
		return
	}
	h.withRuntime(c, func(runtime *services.ProfileRuntime) {
		updated, err := runtime.Configuration.UpdateGatewayBridge(config)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, updated)
	})
}

func (h *ProfileScopedHandler) StartSimulation(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Start(c) })
}

func (h *ProfileScopedHandler) SetSimulationSpeed(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.SetSpeed(c) })
}

func (h *ProfileScopedHandler) PauseSimulation(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Pause(c) })
}

func (h *ProfileScopedHandler) ResumeSimulation(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Resume(c) })
}

func (h *ProfileScopedHandler) StopSimulation(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Stop(c) })
}

func (h *ProfileScopedHandler) SimulationSnapshot(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Snapshot(c) })
}

func (h *ProfileScopedHandler) SimulationEvents(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Events(c) })
}

func (h *ProfileScopedHandler) ExportSimulationLogs(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.ExportLogs(c) })
}

func (h *ProfileScopedHandler) ExportSimulationTelemetry(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.ExportTelemetry(c) })
}

func (h *ProfileScopedHandler) SimulationLogs(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Logs(c) })
}

func (h *ProfileScopedHandler) SimulationLog(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Log(c) })
}

func (h *ProfileScopedHandler) SimulationLogEvents(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.LogEvents(c) })
}

func (h *ProfileScopedHandler) SimulationMetrics(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.Metrics(c) })
}

func (h *ProfileScopedHandler) QueueUplink(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.QueueUplink(c) })
}

func (h *ProfileScopedHandler) QueueDownlink(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.QueueDownlink(c) })
}

func (h *ProfileScopedHandler) QueueMACCommand(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.QueueMACCommand(c) })
}

func (h *ProfileScopedHandler) SimulationWebSocket(c *gin.Context) {
	h.withSimulationHandler(c, func(handler *SimulationHandler) { handler.WebSocket(c) })
}

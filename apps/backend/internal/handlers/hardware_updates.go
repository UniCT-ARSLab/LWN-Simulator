package handlers

import (
	"net/http"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"lwn-simulator-backend/internal/services"
)

func (h *ProfileScopedHandler) UpdateDevicePosition(c *gin.Context) {
	updateHardware(c, h, func(runtime *services.ProfileRuntime, id string, req contracts.UpdatePositionRequest) (contracts.UpdateDeviceResponse, error) {
		return runtime.Device.UpdatePosition(id, req)
	})
}

func (h *ProfileScopedHandler) UpdateDeviceActive(c *gin.Context) {
	updateHardware(c, h, func(runtime *services.ProfileRuntime, id string, req contracts.UpdateActiveRequest) (contracts.UpdateDeviceResponse, error) {
		return runtime.Device.UpdateActive(id, *req.Active)
	})
}

func (h *ProfileScopedHandler) UpdateDevicePayload(c *gin.Context) {
	updateHardware(c, h, func(runtime *services.ProfileRuntime, id string, req contracts.UpdateDevicePayloadRequest) (contracts.UpdateDeviceResponse, error) {
		return runtime.Device.UpdatePayload(id, *req.Payload, req.Base64Encoded)
	})
}

func (h *ProfileScopedHandler) UpdateGatewayPosition(c *gin.Context) {
	updateHardware(c, h, func(runtime *services.ProfileRuntime, id string, req contracts.UpdatePositionRequest) (contracts.UpdateGatewayResponse, error) {
		return runtime.Gateway.UpdatePosition(id, req)
	})
}

func (h *ProfileScopedHandler) UpdateGatewayActive(c *gin.Context) {
	updateHardware(c, h, func(runtime *services.ProfileRuntime, id string, req contracts.UpdateActiveRequest) (contracts.UpdateGatewayResponse, error) {
		return runtime.Gateway.UpdateActive(id, *req.Active)
	})
}

func updateHardware[Request, Response any](
	c *gin.Context,
	h *ProfileScopedHandler,
	update func(*services.ProfileRuntime, string, Request) (Response, error),
) {
	var path contracts.GetDeviceRequest
	var req Request
	if !bindUriAndValidate(c, h.validator, &path) || !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	h.withRuntime(c, func(runtime *services.ProfileRuntime) {
		result, err := update(runtime, path.ID, req)
		if err != nil {
			writeServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, result)
	})
}

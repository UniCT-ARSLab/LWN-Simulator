package handlers

import (
	"net/http"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"lwn-simulator-backend/internal/services"
)

type DeviceService interface {
	CreateDevice(
		req contracts.CreateDeviceRequest,
	) (contracts.CreateDeviceResponse, error)

	GetDevice(
		req contracts.GetDeviceRequest,
	) (contracts.GetDeviceResponse, error)

	GetDevices(
		req contracts.GetDevicesRequest,
	) (contracts.GetDevicesResponse, error)

	UpdateDevice(
		req contracts.UpdateDeviceRequest,
	) (contracts.UpdateDeviceResponse, error)

	DeleteDevice(
		req contracts.DeleteDeviceRequest,
	) (contracts.DeleteDeviceResponse, error)
}

type DeviceHandler struct {
	service   DeviceService
	validator *validator.Validate
}

func NewDeviceHandler(
	service DeviceService,
	validator *validator.Validate,
) *DeviceHandler {
	return &DeviceHandler{
		service:   service,
		validator: validator,
	}
}

func (h *DeviceHandler) CreateDevice(c *gin.Context) {
	var req contracts.CreateDeviceRequest

	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	if err := services.ValidateDeviceRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.CreateDevice(req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *DeviceHandler) GetDevice(c *gin.Context) {
	var req contracts.GetDeviceRequest

	if !bindUriAndValidate(c, h.validator, &req) {
		return
	}

	res, err := h.service.GetDevice(req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *DeviceHandler) GetDevices(c *gin.Context) {
	var req contracts.GetDevicesRequest

	res, err := h.service.GetDevices(req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *DeviceHandler) UpdateDevice(c *gin.Context) {
	var req contracts.UpdateDeviceRequest

	if !bindURIJSONAndValidate(c, h.validator, &req) {
		return
	}

	if req.Device.ID != req.ID {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device ID in body does not match device ID in URI",
		})
		return
	}
	if err := services.ValidateDevice(req.Device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.UpdateDevice(req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}

func (h *DeviceHandler) DeleteDevice(c *gin.Context) {
	var req contracts.DeleteDeviceRequest

	if !bindUriAndValidate(c, h.validator, &req) {
		return
	}

	_, err := h.service.DeleteDevice(req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

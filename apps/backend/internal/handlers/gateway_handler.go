package handlers

import (
	"net/http"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"lwn-simulator-backend/internal/services"
)

type GatewayHandler struct {
	service   GatewayService
	validator *validator.Validate
}

type GatewayService interface {
	CreateGateway(req contracts.CreateGatewayRequest) (contracts.Gateway, error)
	GetGateway(req contracts.GetGatewayRequest) (contracts.GetGatewayResponse, error)
	GetGateways(req contracts.GetGatewaysRequest) (contracts.GetGatewaysResponse, error)
	UpdateGateway(req contracts.UpdateGatewayRequest) (contracts.UpdateGatewayResponse, error)
	DeleteGateway(req contracts.DeleteGatewayRequest) (contracts.DeleteGatewayResponse, error)
}

func NewGatewayHandler(service GatewayService, validator *validator.Validate) *GatewayHandler {
	return &GatewayHandler{service: service, validator: validator}
}

func (h *GatewayHandler) CreateGateway(c *gin.Context) {
	var req contracts.CreateGatewayRequest
	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	if err := services.ValidateGatewayRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gateway, err := h.service.CreateGateway(req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, contracts.CreateGatewayResponse{Gateway: gateway})
}

func (h *GatewayHandler) GetGateway(c *gin.Context) {
	var req contracts.GetGatewayRequest
	if !bindUriAndValidate(c, h.validator, &req) {
		return
	}
	res, err := h.service.GetGateway(req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *GatewayHandler) GetGateways(c *gin.Context) {
	res, err := h.service.GetGateways(contracts.GetGatewaysRequest{})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *GatewayHandler) UpdateGateway(c *gin.Context) {
	var req contracts.UpdateGatewayRequest
	if !bindURIJSONAndValidate(c, h.validator, &req) {
		return
	}
	if req.Gateway.ID != req.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateway ID in body does not match gateway ID in URI"})
		return
	}
	if err := services.ValidateGateway(req.Gateway); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.UpdateGateway(req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *GatewayHandler) DeleteGateway(c *gin.Context) {
	var req contracts.DeleteGatewayRequest
	if !bindUriAndValidate(c, h.validator, &req) {
		return
	}
	if _, err := h.service.DeleteGateway(req); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

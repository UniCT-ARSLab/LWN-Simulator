package handlers

import (
	"net/http"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
)

type SimulationActivityService interface {
	ActiveSimulation() (contracts.SimulationActivity, error)
}

type SimulationActivityHandler struct {
	service SimulationActivityService
}

func NewSimulationActivityHandler(service SimulationActivityService) *SimulationActivityHandler {
	return &SimulationActivityHandler{service: service}
}

func (h *SimulationActivityHandler) Get(c *gin.Context) {
	activity, err := h.service.ActiveSimulation()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, activity)
}

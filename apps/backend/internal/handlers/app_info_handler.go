package handlers

import (
	"lwn-simulator-backend/version"
	"net/http"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
)

type AppInfoHandler struct {
}

func NewAppInfoHandler() *AppInfoHandler {
	return &AppInfoHandler{}
}

func (h *AppInfoHandler) Status(port string) gin.HandlerFunc {
	return func(c *gin.Context) {

		res := contracts.StatusResponse{
			Port: port,
		}

		c.JSON(http.StatusOK, res)
	}
}

func (h *AppInfoHandler) Info(c *gin.Context) {

	res := contracts.AppInfoResponse{
		App:     "lwn-simulator",
		Version: version.AppVersion,
	}

	c.JSON(http.StatusOK, res)
}

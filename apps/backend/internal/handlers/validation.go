package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"lwn-simulator-backend/internal/apperrors"
)

func bindJSONAndValidate[T any](
	c *gin.Context,
	validator *validator.Validate,
	req *T,
) bool {

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return false
	}

	if err := validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return false
	}

	return true
}

func bindUriAndValidate[T any](
	c *gin.Context,
	validator *validator.Validate,
	req *T,
) bool {

	if err := c.ShouldBindUri(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request path",
		})
		return false
	}

	if err := validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return false
	}

	return true
}

func bindURIJSONAndValidate[T any](
	c *gin.Context,
	validator *validator.Validate,
	req *T,
) bool {
	if err := c.ShouldBindUri(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request path"})
		return false
	}

	return bindJSONAndValidate(c, validator, req)
}

func writeServiceError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"

	switch {
	case errors.Is(err, apperrors.ErrInvalid):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, apperrors.ErrNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, apperrors.ErrConflict):
		status = http.StatusConflict
		message = err.Error()
	}

	c.JSON(status, gin.H{"error": message})
}

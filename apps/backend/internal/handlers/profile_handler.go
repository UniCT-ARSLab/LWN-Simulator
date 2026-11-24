package handlers

import (
	"net/http"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"lwn-simulator-backend/internal/services"
)

type ProfileService interface {
	List() ([]contracts.Profile, error)
	Get(string) (contracts.Profile, error)
	Create(string) (contracts.Profile, error)
	Rename(string, string) (contracts.Profile, error)
	Delete(string) error
}

type ProfileArchiveService interface {
	ExportArchive(string) (contracts.ProfileArchive, error)
	ImportArchive(contracts.ProfileArchive) (contracts.Profile, error)
}

type ProfileHandler struct {
	service        ProfileService
	archiveService ProfileArchiveService
	validator      *validator.Validate
}

func NewProfileHandler(service ProfileService, validator *validator.Validate, archiveServices ...ProfileArchiveService) *ProfileHandler {
	var archiveService ProfileArchiveService
	if len(archiveServices) > 0 {
		archiveService = archiveServices[0]
	}
	return &ProfileHandler{service: service, archiveService: archiveService, validator: validator}
}

func (h *ProfileHandler) GetProfiles(c *gin.Context) {
	profiles, err := h.service.List()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, contracts.GetProfilesResponse{Profiles: profiles, Total: int32(len(profiles))})
}

func (h *ProfileHandler) GetProfile(c *gin.Context) {
	profile, err := h.service.Get(c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, contracts.GetProfileResponse{Profile: profile})
}

func (h *ProfileHandler) CreateProfile(c *gin.Context) {
	var req contracts.CreateProfileRequest
	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	profile, err := h.service.Create(req.Name)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, contracts.CreateProfileResponse{Profile: profile})
}

func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	var req contracts.UpdateProfileRequest
	if !bindURIJSONAndValidate(c, h.validator, &req) {
		return
	}
	profile, err := h.service.Rename(req.ID, req.Name)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, contracts.UpdateProfileResponse{Profile: profile})
}

func (h *ProfileHandler) DeleteProfile(c *gin.Context) {
	if err := h.service.Delete(c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ProfileHandler) ExportProfile(c *gin.Context) {
	if h.archiveService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "profile archive service is not configured"})
		return
	}
	archive, err := h.archiveService.ExportArchive(c.Param("id"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	if c.Query("download") == "1" {
		c.Header("Content-Disposition", `attachment; filename="lwn-profile.json"`)
		c.Header("Content-Type", "application/json")
	}
	c.JSON(http.StatusOK, archive)
}

func (h *ProfileHandler) ImportProfile(c *gin.Context) {
	if h.archiveService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "profile archive service is not configured"})
		return
	}
	var req contracts.ImportProfileRequest
	if !bindJSONAndValidate(c, h.validator, &req) {
		return
	}
	profile, err := h.archiveService.ImportArchive(req.Archive)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, contracts.ImportProfileResponse{Profile: profile})
}

var _ ProfileService = (*services.ProfileService)(nil)

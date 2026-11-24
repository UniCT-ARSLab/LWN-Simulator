package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"lwn-simulator-backend/internal/apperrors"
)

type profileHandlerProfileServiceStub struct{}

func (profileHandlerProfileServiceStub) List() ([]contracts.Profile, error) {
	return nil, nil
}

func (profileHandlerProfileServiceStub) Get(string) (contracts.Profile, error) {
	return contracts.Profile{}, nil
}

func (profileHandlerProfileServiceStub) Create(string) (contracts.Profile, error) {
	return contracts.Profile{}, nil
}

func (profileHandlerProfileServiceStub) Rename(string, string) (contracts.Profile, error) {
	return contracts.Profile{}, nil
}

func (profileHandlerProfileServiceStub) Delete(string) error {
	return nil
}

type profileHandlerArchiveServiceStub struct{}

func (profileHandlerArchiveServiceStub) ExportArchive(string) (contracts.ProfileArchive, error) {
	return contracts.ProfileArchive{}, nil
}

func (profileHandlerArchiveServiceStub) ImportArchive(contracts.ProfileArchive) (contracts.Profile, error) {
	return contracts.Profile{}, nil
}

type profileHandlerMockProfileService struct {
	profiles    []contracts.Profile
	profile     contracts.Profile
	listErr     error
	getErr      error
	createErr   error
	renameErr   error
	deleteErr   error
	getID       string
	createName  string
	renameID    string
	renameName  string
	deleteID    string
	listCalls   int
	getCalls    int
	createCalls int
	renameCalls int
	deleteCalls int
}

func (m *profileHandlerMockProfileService) List() ([]contracts.Profile, error) {
	m.listCalls++
	return m.profiles, m.listErr
}

func (m *profileHandlerMockProfileService) Get(id string) (contracts.Profile, error) {
	m.getCalls++
	m.getID = id
	return m.profile, m.getErr
}

func (m *profileHandlerMockProfileService) Create(name string) (contracts.Profile, error) {
	m.createCalls++
	m.createName = name
	return m.profile, m.createErr
}

func (m *profileHandlerMockProfileService) Rename(id string, name string) (contracts.Profile, error) {
	m.renameCalls++
	m.renameID = id
	m.renameName = name
	return m.profile, m.renameErr
}

func (m *profileHandlerMockProfileService) Delete(id string) error {
	m.deleteCalls++
	m.deleteID = id
	return m.deleteErr
}

type profileHandlerMockArchiveService struct {
	archive       contracts.ProfileArchive
	profile       contracts.Profile
	exportErr     error
	importErr     error
	exportID      string
	importArchive contracts.ProfileArchive
	exportCalls   int
	importCalls   int
}

func (m *profileHandlerMockArchiveService) ExportArchive(id string) (contracts.ProfileArchive, error) {
	m.exportCalls++
	m.exportID = id
	return m.archive, m.exportErr
}

func (m *profileHandlerMockArchiveService) ImportArchive(archive contracts.ProfileArchive) (contracts.Profile, error) {
	m.importCalls++
	m.importArchive = archive
	return m.profile, m.importErr
}

func newProfileHandlerRouter(
	profileService ProfileService,
	archiveService ProfileArchiveService,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewProfileHandler(profileService, validator.New(), archiveService)
	router.GET("/profiles", handler.GetProfiles)
	router.GET("/profiles/:id", handler.GetProfile)
	router.POST("/profiles", handler.CreateProfile)
	router.PUT("/profiles/:id", handler.UpdateProfile)
	router.DELETE("/profiles/:id", handler.DeleteProfile)
	router.GET("/archives/:id", handler.ExportProfile)
	router.POST("/archives", handler.ImportProfile)
	return router
}

func performProfileRequest(
	router *gin.Engine,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func decodeProfileResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()
	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
}

func TestProfileHandlerListsAndGetsProfiles(t *testing.T) {
	service := &profileHandlerMockProfileService{
		profiles: []contracts.Profile{{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "Field"}},
		profile:  contracts.Profile{ID: "550e8400-e29b-41d4-a716-446655440001", Name: "Core"},
	}
	router := newProfileHandlerRouter(service, nil)

	listRecorder := performProfileRequest(router, http.MethodGet, "/profiles", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRecorder.Code, http.StatusOK)
	}
	var listResponse contracts.GetProfilesResponse
	decodeProfileResponse(t, listRecorder, &listResponse)
	if listResponse.Total != 1 || len(listResponse.Profiles) != 1 || service.listCalls != 1 {
		t.Fatalf("unexpected list response: %+v, calls=%d", listResponse, service.listCalls)
	}

	getRecorder := performProfileRequest(router, http.MethodGet, "/profiles/profile-id", "")
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRecorder.Code, http.StatusOK)
	}
	var getResponse contracts.GetProfileResponse
	decodeProfileResponse(t, getRecorder, &getResponse)
	if getResponse.Profile.Name != "Core" || service.getID != "profile-id" || service.getCalls != 1 {
		t.Fatalf("unexpected get response or request: %+v, id=%q, calls=%d", getResponse, service.getID, service.getCalls)
	}
}

func TestProfileHandlerCreatesUpdatesAndDeletesProfiles(t *testing.T) {
	service := &profileHandlerMockProfileService{
		profile: contracts.Profile{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "Updated"},
	}
	router := newProfileHandlerRouter(service, nil)

	createRecorder := performProfileRequest(router, http.MethodPost, "/profiles", `{"name":"New network"}`)
	if createRecorder.Code != http.StatusCreated || service.createName != "New network" {
		t.Fatalf("create status/name = %d/%q", createRecorder.Code, service.createName)
	}
	var createResponse contracts.CreateProfileResponse
	decodeProfileResponse(t, createRecorder, &createResponse)
	if createResponse.Profile.Name != "Updated" {
		t.Fatalf("unexpected create response: %+v", createResponse)
	}

	updateRecorder := performProfileRequest(router, http.MethodPut, "/profiles/550e8400-e29b-41d4-a716-446655440000", `{"name":"Renamed"}`)
	if updateRecorder.Code != http.StatusOK || service.renameID != "550e8400-e29b-41d4-a716-446655440000" || service.renameName != "Renamed" {
		t.Fatalf("update status/request = %d/%q/%q", updateRecorder.Code, service.renameID, service.renameName)
	}
	var updateResponse contracts.UpdateProfileResponse
	decodeProfileResponse(t, updateRecorder, &updateResponse)
	if updateResponse.Profile.Name != "Updated" {
		t.Fatalf("unexpected update response: %+v", updateResponse)
	}

	deleteRecorder := performProfileRequest(router, http.MethodDelete, "/profiles/profile-id", "")
	if deleteRecorder.Code != http.StatusNoContent || service.deleteID != "profile-id" || service.deleteCalls != 1 {
		t.Fatalf("delete status/id/calls = %d/%q/%d", deleteRecorder.Code, service.deleteID, service.deleteCalls)
	}
}

func TestProfileHandlerRejectsInvalidRequests(t *testing.T) {
	service := &profileHandlerMockProfileService{}
	router := newProfileHandlerRouter(service, profileHandlerArchiveServiceStub{})

	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "invalid create json", method: http.MethodPost, path: "/profiles", body: "{"},
		{name: "invalid update path", method: http.MethodPut, path: "/profiles/not-a-valid-uuid", body: `{"name":"Renamed"}`},
		{name: "invalid update json", method: http.MethodPut, path: "/profiles/550e8400-e29b-41d4-a716-446655440000", body: "{"},
		{name: "invalid import json", method: http.MethodPost, path: "/archives", body: "{"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := performProfileRequest(router, test.method, test.path, test.body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
		})
	}
	if service.createCalls != 0 || service.renameCalls != 0 {
		t.Fatalf("invalid requests reached the service: create=%d rename=%d", service.createCalls, service.renameCalls)
	}
}

func TestProfileHandlerMapsServiceErrors(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		setup func(*profileHandlerMockProfileService)
	}{
		{name: "list not found", path: "/profiles", setup: func(service *profileHandlerMockProfileService) {
			service.listErr = apperrors.NotFound("profiles unavailable")
		}},
		{name: "get conflict", path: "/profiles/profile-id", setup: func(service *profileHandlerMockProfileService) {
			service.getErr = apperrors.Conflict("profile is locked")
		}},
		{name: "create invalid", path: "/profiles", setup: func(service *profileHandlerMockProfileService) {
			service.createErr = apperrors.Invalid("invalid profile name")
		}},
		{name: "update conflict", path: "/profiles/550e8400-e29b-41d4-a716-446655440000", setup: func(service *profileHandlerMockProfileService) {
			service.renameErr = apperrors.Conflict("profile name already exists")
		}},
		{name: "delete internal error", path: "/profiles/profile-id", setup: func(service *profileHandlerMockProfileService) {
			service.deleteErr = errors.New("storage unavailable")
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &profileHandlerMockProfileService{}
			test.setup(service)
			router := newProfileHandlerRouter(service, nil)
			method := http.MethodGet
			body := ""
			switch test.name {
			case "create invalid":
				method = http.MethodPost
				body = `{"name":"Invalid"}`
			case "update conflict":
				method = http.MethodPut
				body = `{"name":"Existing"}`
			case "delete internal error":
				method = http.MethodDelete
			}
			recorder := performProfileRequest(router, method, test.path, body)
			wantStatus := http.StatusInternalServerError
			if test.name == "list not found" {
				wantStatus = http.StatusNotFound
			}
			if test.name == "get conflict" {
				wantStatus = http.StatusConflict
			}
			if test.name == "create invalid" {
				wantStatus = http.StatusBadRequest
			}
			if test.name == "update conflict" {
				wantStatus = http.StatusConflict
			}
			if recorder.Code != wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestProfileHandlerMapsArchiveServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		setup      func(*profileHandlerMockArchiveService)
		wantStatus int
	}{
		{
			name:   "export not found",
			method: http.MethodGet,
			path:   "/archives/profile-id",
			setup: func(service *profileHandlerMockArchiveService) {
				service.exportErr = apperrors.NotFound("profile not found")
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "import conflict",
			method: http.MethodPost,
			path:   "/archives",
			body:   `{"archive":{"profile":{"id":"550e8400-e29b-41d4-a716-446655440000","name":"Source"}}}`,
			setup: func(service *profileHandlerMockArchiveService) {
				service.importErr = apperrors.Conflict("profile already exists")
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archiveService := &profileHandlerMockArchiveService{}
			test.setup(archiveService)
			router := newProfileHandlerRouter(&profileHandlerMockProfileService{}, archiveService)
			recorder := performProfileRequest(router, test.method, test.path, test.body)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
		})
	}
}

func TestProfileHandlerImportsProfiles(t *testing.T) {
	archiveService := &profileHandlerMockArchiveService{
		profile: contracts.Profile{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "Imported"},
	}
	router := newProfileHandlerRouter(&profileHandlerMockProfileService{}, archiveService)
	body := `{"archive":{"format":"lwn-simulator-profile","version":1,"profile":{"id":"550e8400-e29b-41d4-a716-446655440001","name":"Source"},"devices":[],"gateways":[],"logs":[]}}`

	recorder := performProfileRequest(router, http.MethodPost, "/archives", body)
	if recorder.Code != http.StatusCreated || archiveService.importCalls != 1 {
		t.Fatalf("import status/calls = %d/%d", recorder.Code, archiveService.importCalls)
	}
	if archiveService.importArchive.Profile.Name != "Source" {
		t.Fatalf("archive was not forwarded: %+v", archiveService.importArchive)
	}
	var response contracts.ImportProfileResponse
	decodeProfileResponse(t, recorder, &response)
	if response.Profile.Name != "Imported" {
		t.Fatalf("unexpected import response: %+v", response)
	}
}

func TestProfileHandlerRequiresArchiveService(t *testing.T) {
	router := newProfileHandlerRouter(&profileHandlerMockProfileService{}, nil)

	for _, test := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "export", method: http.MethodGet, path: "/archives/profile-id"},
		{name: "import", method: http.MethodPost, path: "/archives"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := performProfileRequest(router, test.method, test.path, `{}`)
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestExportProfileMarksDownloadResponsesAsAttachments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewProfileHandler(
		profileHandlerProfileServiceStub{},
		validator.New(),
		profileHandlerArchiveServiceStub{},
	)
	router.GET("/profiles/:id/export", handler.ExportProfile)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/profiles/profile-id/export?download=1",
		nil,
	)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Disposition"); got != `attachment; filename="lwn-profile.json"` {
		t.Fatalf("expected attachment header, got %q", got)
	}
}

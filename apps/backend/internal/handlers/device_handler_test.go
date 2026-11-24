package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type mockDeviceService struct {
	createDeviceResponse contracts.CreateDeviceResponse
	getDeviceResponse    contracts.GetDeviceResponse
	getDevicesResponse   contracts.GetDevicesResponse
	updateDeviceResponse contracts.UpdateDeviceResponse
	deleteDeviceResponse contracts.DeleteDeviceResponse

	createDeviceErr error
	getDeviceErr    error
	getDevicesErr   error
	updateDeviceErr error
	deleteDeviceErr error

	createDeviceCalls int
	getDeviceCalls    int
	getDevicesCalls   int
	updateDeviceCalls int
	deleteDeviceCalls int

	lastCreateRequest contracts.CreateDeviceRequest
	lastGetRequest    contracts.GetDeviceRequest
	lastGetDevicesReq contracts.GetDevicesRequest
	lastUpdateRequest contracts.UpdateDeviceRequest
	lastDeleteRequest contracts.DeleteDeviceRequest
}

const (
	testDeviceID1 = "550e8400-e29b-41d4-a716-446655440000"
	testDeviceID2 = "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
)

func (m *mockDeviceService) CreateDevice(
	req contracts.CreateDeviceRequest,
) (contracts.CreateDeviceResponse, error) {
	m.createDeviceCalls++
	m.lastCreateRequest = req

	return m.createDeviceResponse, m.createDeviceErr
}

func (m *mockDeviceService) GetDevice(
	req contracts.GetDeviceRequest,
) (contracts.GetDeviceResponse, error) {
	m.getDeviceCalls++
	m.lastGetRequest = req

	return m.getDeviceResponse, m.getDeviceErr
}

func (m *mockDeviceService) GetDevices(
	req contracts.GetDevicesRequest,
) (contracts.GetDevicesResponse, error) {
	m.getDevicesCalls++
	m.lastGetDevicesReq = req

	return m.getDevicesResponse, m.getDevicesErr
}

func (m *mockDeviceService) UpdateDevice(
	req contracts.UpdateDeviceRequest,
) (contracts.UpdateDeviceResponse, error) {
	m.updateDeviceCalls++
	m.lastUpdateRequest = req

	return m.updateDeviceResponse, m.updateDeviceErr
}

func (m *mockDeviceService) DeleteDevice(
	req contracts.DeleteDeviceRequest,
) (contracts.DeleteDeviceResponse, error) {
	m.deleteDeviceCalls++
	m.lastDeleteRequest = req

	return m.deleteDeviceResponse, m.deleteDeviceErr
}

func newTestDeviceHandler(
	service *mockDeviceService,
) *DeviceHandler {
	return NewDeviceHandler(
		service,
		validator.New(),
	)
}

func setupRouter(handler *DeviceHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	router.POST("/devices", handler.CreateDevice)
	router.GET("/devices/:id", handler.GetDevice)
	router.GET("/devices", handler.GetDevices)
	router.PUT("/devices/:id", handler.UpdateDevice)
	router.DELETE("/devices/:id", handler.DeleteDevice)

	return router
}

// validCreateDeviceJSON returns a body satisfying all validation tags
// currently generated in contracts.CreateDeviceRequest.
func validCreateDeviceJSON() string {
	return `{
		"devEUI": "0102030405060708",
		"name": "test-device",
		"activation": "oota",
		"radioProtocol": "lorawan",
		"radioReceiveMode": "lorawan",
		"active": true,
		"class": "class-a",
		"OOTAConfig": {
			"joinEUI": "0102030405060708",
			"appKey": "00112233445566778899AABBCCDDEEFF"
		},

		"locationConfig": {
			"altitude": 100,
			"latitude": 37.5,
			"longitude": 15.0,
			"region": "EU868"
		},

		"RX1Config": {
			"dataRateOffset": 0,
			"delay": 1000,
			"duration": 3000
		},

		"RX2Config": {
			"ACKTimeout": 1000,
			"channelFrequency": 869525000,
			"dataRate": 0,
			"delay": 2000,
			"duration": 3000
		},

		"advancedConfig": {
			"ADREnabled": true,
			"antennaRange": 10
		},

		"frameConfig": {
			"FCntDown": 0,
			"FCntUp": 0,
			"disableFrameCounterValidation": false,
			"fPort": 1,
			"retransmission": 0
		},

		"payloadConfig": {
			"MType": "unconfirmed-data-up",
			"base64Encoded": false,
			"oversizedPayloadBehavior": "truncate",
			"payload": "",
			"uplinkInterval": 10
		}
	}`
}

// validDeviceJSON returns a complete valid Device object.
// This is important for UpdateDeviceRequest because the generated
// contracts.Device contains required validation tags.
func validDeviceJSON(id string) string {
	return fmt.Sprintf(`{
		"id": "%s",
		"devEUI": "0102030405060708",
		"name": "test-device",
		"activation": "oota",
		"active": true,
		"class": "class-a",
		"OOTAConfig": {
			"joinEUI": "0102030405060708",
			"appKey": "00112233445566778899AABBCCDDEEFF"
		},

		"locationConfig": {
			"altitude": 100,
			"latitude": 37.5,
			"longitude": 15.0,
			"region": "EU868"
		},

		"RX1Config": {
			"dataRateOffset": 0,
			"delay": 1000,
			"duration": 3000
		},

		"RX2Config": {
			"ACKTimeout": 1000,
			"channelFrequency": 869525000,
			"dataRate": 0,
			"delay": 2000,
			"duration": 3000
		},

		"advancedConfig": {
			"ADREnabled": true,
			"antennaRange": 10
		},

		"frameConfig": {
			"FCntDown": 0,
			"FCntUp": 0,
			"disableFrameCounterValidation": false,
			"fPort": 1,
			"retransmission": 0
		},

		"payloadConfig": {
			"MType": "unconfirmed-data-up",
			"base64Encoded": false,
			"oversizedPayloadBehavior": "truncate",
			"payload": "",
			"uplinkInterval": 10
		}
	}`, id)
}

// validUpdateDeviceJSON creates an UpdateDeviceRequest with a complete
// and therefore valid Device object.
func validUpdateDeviceJSON(id, devEUI, name string, active bool) string {
	return fmt.Sprintf(`{
		"device": {
			"id": "%s",
			"devEUI": "%s",
			"name": "%s",
			"activation": "oota",
			"radioProtocol": "raw-lora",
			"radioReceiveMode": "auto",
			"active": %t,
			"class": "class-a",
			"OOTAConfig": {
				"joinEUI": "0102030405060708",
				"appKey": "00112233445566778899AABBCCDDEEFF"
			},

			"locationConfig": {
				"altitude": 100,
				"latitude": 37.5,
				"longitude": 15.0,
				"region": "EU868"
			},

			"RX1Config": {
				"dataRateOffset": 0,
				"delay": 1000,
				"duration": 3000
			},

			"RX2Config": {
				"ACKTimeout": 1000,
				"channelFrequency": 869525000,
				"dataRate": 0,
				"delay": 2000,
				"duration": 3000
			},

			"advancedConfig": {
				"ADREnabled": true,
				"antennaRange": 10
			},

			"frameConfig": {
				"FCntDown": 0,
				"FCntUp": 0,
				"disableFrameCounterValidation": false,
				"fPort": 1,
				"retransmission": 0
			},

			"payloadConfig": {
				"MType": "unconfirmed-data-up",
				"base64Encoded": false,
				"oversizedPayloadBehavior": "truncate",
				"payload": "",
				"uplinkInterval": 10
			}
		}
	}`, id, devEUI, name, active)
}

func performRequest(
	router *gin.Engine,
	method string,
	path string,
	body string,
) *httptest.ResponseRecorder {
	var reader *strings.Reader

	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(
		method,
		path,
		reader,
	)

	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	return recorder
}

func TestDeviceHandler_CreateDevice(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		service        *mockDeviceService
		wantStatusCode int
		wantCalls      int
	}{
		{
			name: "creates device",
			body: validCreateDeviceJSON(),
			service: &mockDeviceService{
				createDeviceResponse: contracts.CreateDeviceResponse{
					Device: contracts.Device{
						ID:         testDeviceID1,
						DevEUI:     "0102030405060708",
						Name:       "test-device",
						Activation: contracts.OOTA,
						Active:     true,
						Class:      contracts.ClassA,
					},
				},
			},
			wantStatusCode: http.StatusCreated,
			wantCalls:      1,
		},
		{
			name: "returns bad request for invalid JSON",
			body: `{
				"devEUI": "0102030405060708"
			`,
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name: "returns bad request for validation error",
			body: `{
				"devEUI": "invalid"
			}`,
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name: "returns bad request when required fields are missing",
			body: `{
				"devEUI": "0102030405060708"
			}`,
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for a zero RX1 delay",
			body:           strings.Replace(validCreateDeviceJSON(), `"delay": 1000`, `"delay": 0`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for a zero RX1 duration",
			body:           strings.Replace(validCreateDeviceJSON(), `"duration": 3000`, `"duration": 0`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for an out of range RX1 data rate offset",
			body:           strings.Replace(validCreateDeviceJSON(), `"dataRateOffset": 0`, `"dataRateOffset": 6`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for a zero RX2 delay",
			body:           strings.Replace(validCreateDeviceJSON(), `"delay": 2000`, `"delay": 0`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name: "returns bad request for a zero RX2 duration",
			body: strings.Replace(validCreateDeviceJSON(), `"delay": 2000,
			"duration": 3000`, `"delay": 2000,
			"duration": 0`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for a zero RX2 frequency",
			body:           strings.Replace(validCreateDeviceJSON(), `"channelFrequency": 869525000`, `"channelFrequency": 0`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for an out of range RX2 data rate",
			body:           strings.Replace(validCreateDeviceJSON(), `"dataRate": 0`, `"dataRate": 16`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for an ACK timeout expressed as seconds",
			body:           strings.Replace(validCreateDeviceJSON(), `"ACKTimeout": 1000`, `"ACKTimeout": 1`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for a zero uplink interval",
			body:           strings.Replace(validCreateDeviceJSON(), `"uplinkInterval": 10`, `"uplinkInterval": 0`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for an unsupported radio protocol",
			body:           strings.Replace(validCreateDeviceJSON(), `"radioProtocol": "lorawan"`, `"radioProtocol": "unsupported"`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request for an unsupported receive mode",
			body:           strings.Replace(validCreateDeviceJSON(), `"radioReceiveMode": "lorawan"`, `"radioReceiveMode": "unsupported"`, 1),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name: "returns internal server error",
			body: validCreateDeviceJSON(),
			service: &mockDeviceService{
				createDeviceErr: errors.New("database error"),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantCalls:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestDeviceHandler(tt.service)
			router := setupRouter(handler)

			recorder := performRequest(
				router,
				http.MethodPost,
				"/devices",
				tt.body,
			)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf(
					"status code = %d, want %d; body = %s",
					recorder.Code,
					tt.wantStatusCode,
					recorder.Body.String(),
				)
			}

			if tt.service.createDeviceCalls != tt.wantCalls {
				t.Errorf(
					"CreateDevice() calls = %d, want %d",
					tt.service.createDeviceCalls,
					tt.wantCalls,
				)
			}

			if tt.wantCalls > 0 {
				got := tt.service.lastCreateRequest

				if got.DevEUI != "0102030405060708" {
					t.Errorf(
						"CreateDevice() DevEUI = %q, want %q",
						got.DevEUI,
						"0102030405060708",
					)
				}

				if got.Activation != contracts.OOTA {
					t.Errorf(
						"CreateDevice() Activation = %q, want %q",
						got.Activation,
						contracts.OOTA,
					)
				}

				if got.Class != contracts.ClassA {
					t.Errorf(
						"CreateDevice() Class = %q, want %q",
						got.Class,
						contracts.ClassA,
					)
				}

				if got.RadioProtocol == nil || *got.RadioProtocol != contracts.LoRaWAN {
					t.Errorf("CreateDevice() RadioProtocol = %v, want %q", got.RadioProtocol, contracts.LoRaWAN)
				}

				if got.RadioReceiveMode == nil || *got.RadioReceiveMode != contracts.ReceiveLoRaWAN {
					t.Errorf("CreateDevice() RadioReceiveMode = %v, want %q", got.RadioReceiveMode, contracts.ReceiveLoRaWAN)
				}
			}
		})
	}
}

func TestDeviceHandler_GetDevice(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		service        *mockDeviceService
		wantStatusCode int
		wantCalls      int
	}{
		{
			name: "returns device",
			id:   testDeviceID1,
			service: &mockDeviceService{
				getDeviceResponse: contracts.GetDeviceResponse{
					Device: contracts.Device{
						ID:     testDeviceID1,
						DevEUI: "0102030405060708",
						Name:   "device-1",
					},
				},
			},
			wantStatusCode: http.StatusOK,
			wantCalls:      1,
		},
		{
			name: "returns error when service fails",
			id:   testDeviceID1,
			service: &mockDeviceService{
				getDeviceErr: errors.New("device not found"),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantCalls:      1,
		},
		{
			name:           "returns bad request for invalid id",
			id:             "invalid",
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestDeviceHandler(tt.service)
			router := setupRouter(handler)

			recorder := performRequest(
				router,
				http.MethodGet,
				"/devices/"+tt.id,
				"",
			)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf(
					"status code = %d, want %d; body = %s",
					recorder.Code,
					tt.wantStatusCode,
					recorder.Body.String(),
				)
			}

			if tt.service.getDeviceCalls != tt.wantCalls {
				t.Errorf(
					"GetDevice() calls = %d, want %d",
					tt.service.getDeviceCalls,
					tt.wantCalls,
				)
			}

			if tt.wantCalls > 0 &&
				tt.service.lastGetRequest.ID != tt.id {
				t.Errorf(
					"GetDevice() ID = %q, want %q",
					tt.service.lastGetRequest.ID,
					tt.id,
				)
			}
		})
	}
}

func TestDeviceHandler_GetDevices(t *testing.T) {
	tests := []struct {
		name           string
		service        *mockDeviceService
		wantStatusCode int
		wantCalls      int
	}{
		{
			name: "returns devices",
			service: &mockDeviceService{
				getDevicesResponse: contracts.GetDevicesResponse{
					Devices: []contracts.Device{
						{
							ID:     testDeviceID1,
							DevEUI: "0102030405060708",
						},
						{
							ID:     testDeviceID2,
							DevEUI: "1122334455667788",
						},
					},
				},
			},
			wantStatusCode: http.StatusOK,
			wantCalls:      1,
		},
		{
			name: "returns internal server error",
			service: &mockDeviceService{
				getDevicesErr: errors.New("database error"),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantCalls:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestDeviceHandler(tt.service)
			router := setupRouter(handler)

			recorder := performRequest(
				router,
				http.MethodGet,
				"/devices",
				"",
			)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf(
					"status code = %d, want %d; body = %s",
					recorder.Code,
					tt.wantStatusCode,
					recorder.Body.String(),
				)
			}

			if tt.service.getDevicesCalls != tt.wantCalls {
				t.Errorf(
					"GetDevices() calls = %d, want %d",
					tt.service.getDevicesCalls,
					tt.wantCalls,
				)
			}
		})
	}
}

func TestDeviceHandler_UpdateDevice(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		body           string
		service        *mockDeviceService
		wantStatusCode int
		wantCalls      int
	}{
		{
			name: "updates device",
			id:   testDeviceID1,
			body: validUpdateDeviceJSON(
				testDeviceID1,
				"1122334455667788",
				"updated-device",
				false,
			),
			service: &mockDeviceService{
				updateDeviceResponse: contracts.UpdateDeviceResponse{
					Device: contracts.Device{
						ID:         testDeviceID1,
						DevEUI:     "1122334455667788",
						Name:       "updated-device",
						Activation: contracts.OOTA,
						Class:      contracts.ClassA,
					},
				},
			},
			wantStatusCode: http.StatusOK,
			wantCalls:      1,
		},
		{
			name: "updates complete device",
			id:   testDeviceID1,
			body: validUpdateDeviceJSON(
				testDeviceID1,
				"1122334455667788",
				"updated-device",
				true,
			),
			service: &mockDeviceService{
				updateDeviceResponse: contracts.UpdateDeviceResponse{
					Device: contracts.Device{
						ID:         testDeviceID1,
						DevEUI:     "1122334455667788",
						Name:       "updated-device",
						Activation: contracts.OOTA,
						Active:     true,
						Class:      contracts.ClassA,
					},
				},
			},
			wantStatusCode: http.StatusOK,
			wantCalls:      1,
		},
		{
			name: "returns bad request for invalid device",
			id:   testDeviceID1,
			body: `{
				"device": {
					"id": "550e8400-e29b-41d4-a716-446655440000",
					"devEUI": "invalid"
				}
			}`,
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name:           "returns bad request when device is missing",
			id:             testDeviceID1,
			body:           `{}`,
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
		{
			name: "returns internal server error",
			id:   testDeviceID1,
			body: validUpdateDeviceJSON(
				testDeviceID1,
				"1122334455667788",
				"updated-device",
				true,
			),
			service: &mockDeviceService{
				updateDeviceErr: errors.New("database error"),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantCalls:      1,
		},
		{
			name: "returns bad request for invalid uri id",
			id:   "invalid",
			body: validUpdateDeviceJSON(
				testDeviceID1,
				"1122334455667788",
				"updated-device",
				true,
			),
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestDeviceHandler(tt.service)
			router := setupRouter(handler)

			recorder := performRequest(
				router,
				http.MethodPut,
				"/devices/"+tt.id,
				tt.body,
			)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf(
					"status code = %d, want %d; body = %s",
					recorder.Code,
					tt.wantStatusCode,
					recorder.Body.String(),
				)
			}

			if tt.service.updateDeviceCalls != tt.wantCalls {
				t.Errorf(
					"UpdateDevice() calls = %d, want %d",
					tt.service.updateDeviceCalls,
					tt.wantCalls,
				)
			}

			if tt.wantCalls > 0 {
				got := tt.service.lastUpdateRequest

				if got.ID != tt.id {
					t.Errorf(
						"UpdateDevice() ID = %q, want %q",
						got.ID,
						tt.id,
					)
				}

				if got.Device.ID != tt.id {
					t.Errorf(
						"UpdateDevice() Device.ID = %q, want %q",
						got.Device.ID,
						tt.id,
					)
				}

				if got.Device.DevEUI != "1122334455667788" {
					t.Errorf(
						"UpdateDevice() Device.DevEUI = %q, want %q",
						got.Device.DevEUI,
						"1122334455667788",
					)
				}

				if got.Device.RadioProtocol == nil || *got.Device.RadioProtocol != contracts.RawLoRa {
					t.Errorf("UpdateDevice() Device.RadioProtocol = %v, want %q", got.Device.RadioProtocol, contracts.RawLoRa)
				}

				if got.Device.RadioReceiveMode == nil || *got.Device.RadioReceiveMode != contracts.ReceiveAuto {
					t.Errorf("UpdateDevice() Device.RadioReceiveMode = %v, want %q", got.Device.RadioReceiveMode, contracts.ReceiveAuto)
				}
			}
		})
	}
}

func TestDeviceHandler_DeleteDevice(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		service        *mockDeviceService
		wantStatusCode int
		wantCalls      int
	}{
		{
			name: "deletes device",
			id:   testDeviceID1,
			service: &mockDeviceService{
				deleteDeviceResponse: contracts.DeleteDeviceResponse{},
			},
			wantStatusCode: http.StatusNoContent,
			wantCalls:      1,
		},
		{
			name: "returns internal server error",
			id:   testDeviceID1,
			service: &mockDeviceService{
				deleteDeviceErr: errors.New("database error"),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantCalls:      1,
		},
		{
			name:           "returns bad request for invalid id",
			id:             "invalid",
			service:        &mockDeviceService{},
			wantStatusCode: http.StatusBadRequest,
			wantCalls:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := newTestDeviceHandler(tt.service)
			router := setupRouter(handler)

			recorder := performRequest(
				router,
				http.MethodDelete,
				"/devices/"+tt.id,
				"",
			)

			if recorder.Code != tt.wantStatusCode {
				t.Fatalf(
					"status code = %d, want %d; body = %s",
					recorder.Code,
					tt.wantStatusCode,
					recorder.Body.String(),
				)
			}

			if tt.service.deleteDeviceCalls != tt.wantCalls {
				t.Errorf(
					"DeleteDevice() calls = %d, want %d",
					tt.service.deleteDeviceCalls,
					tt.wantCalls,
				)
			}

			if tt.wantCalls > 0 &&
				tt.service.lastDeleteRequest.ID != tt.id {
				t.Errorf(
					"DeleteDevice() ID = %q, want %q",
					tt.service.lastDeleteRequest.ID,
					tt.id,
				)
			}
		})
	}
}

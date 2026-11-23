package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	engineTypes "github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/gin-gonic/gin"
	"lwn-simulator-backend/internal/repositories"
	"lwn-simulator-backend/internal/services"
)

func TestScopedHardwareUpdates(t *testing.T) {
	dir := t.TempDir()
	profiles := services.NewProfileService(repositories.NewProfileRepository(dir), dir)
	profile, err := profiles.EnsureDefaultProfile()
	if err != nil {
		t.Fatal(err)
	}
	other, err := profiles.Create("Other network")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	registerRoutes(router, "8080", Services{
		Profiles: services.NewProfileRuntimeManager(dir, profiles, engineTypes.Options{Clock: engineTypes.NewManualClock(0)}, nil),
	})
	base := "/api/profiles/" + profile.ID
	request := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	check := func(method, path, body string, status int, result any) {
		t.Helper()
		w := request(method, path, body)
		if w.Code != status {
			t.Fatalf("%s %s: got %d, want %d: %s", method, path, w.Code, status, w.Body)
		}
		if result != nil {
			if err := json.Unmarshal(w.Body.Bytes(), result); err != nil {
				t.Fatal(err)
			}
		}
	}
	var createdDevice contracts.CreateDeviceResponse
	check(http.MethodPost, base+"/device/create-device", `{
		"name":"External sensor","devEUI":"70B3D57ED0001001","class":"class-a","activation":"abp",
		"ABPConfig":{"devAddr":"26011BDA","nwkSKey":"00112233445566778899AABBCCDDEEFF","appSKey":"FFEEDDCCBBAA99887766554433221100"},
		"locationConfig":{"latitude":37.5,"longitude":15.1,"altitude":100,"region":"EU868"},
		"RX1Config":{"delay":1000,"duration":1000,"dataRateOffset":0},
		"RX2Config":{"delay":2000,"duration":1000,"channelFrequency":869525000,"dataRate":0,"ACKTimeout":2000},
		"frameConfig":{"fPort":10,"retransmission":0,"FCntUp":0,"FCntDown":0},
		"payloadConfig":{"uplinkInterval":10,"oversizedPayloadBehavior":"truncate","MType":"unconfirmed-data-up","payload":"initial","base64Encoded":false},
		"advancedConfig":{"antennaRange":1500,"ADREnabled":true}
	}`, 201, &createdDevice)
	var createdGateway contracts.CreateGatewayResponse
	check(http.MethodPost, base+"/gateway/create-gateway", `{
		"name":"Gateway","type":"virtual","gatewayEUI":"A840410001001001",
		"keepAlive":30,"latitude":37.5,"longitude":15.1,"altitude":100
	}`, 201, &createdGateway)

	deviceID, gatewayID := createdDevice.Device.ID, createdGateway.Gateway.ID
	device := base + "/device/" + deviceID
	gateway := base + "/gateway/" + gatewayID
	check(http.MethodPost, base+"/simulation/start", `{"speed":1}`, 200, nil)
	t.Cleanup(func() { request(http.MethodPost, base+"/simulation/stop", "") })
	check(http.MethodPatch, device+"/position", `{"latitude":0,"longitude":0,"altitude":0}`, 200, nil)
	check(http.MethodPatch, gateway+"/position", `{"latitude":1,"longitude":2,"altitude":3}`, 200, nil)
	check(http.MethodPost, base+"/simulation/pause", "", 200, nil)

	// The payload and position patches must merge with the latest stored device.
	results := make(chan *httptest.ResponseRecorder, 2)
	go func() {
		results <- request(http.MethodPatch, device+"/payload", `{"payload":"SGVsbG8=","base64Encoded":true}`)
	}()
	go func() {
		results <- request(http.MethodPatch, device+"/position", `{"latitude":4,"longitude":5,"altitude":6}`)
	}()
	for range 2 {
		w := <-results
		if w.Code != http.StatusOK {
			t.Fatalf("concurrent update returned %d: %s", w.Code, w.Body)
		}
	}

	check(http.MethodPatch, device+"/payload", `{"payload":"!invalid!"}`, 400, nil)
	check(http.MethodPatch, device+"/payload", `{}`, 400, nil)
	check(http.MethodPatch, device+"/position", `{"latitude":91,"longitude":1,"altitude":1}`, 400, nil)
	check(http.MethodPatch, gateway+"/position", `{"latitude":0,"longitude":0}`, 400, nil)
	check(http.MethodPatch, device+"/active", `{"active":null}`, 400, nil)
	check(http.MethodPatch, gateway+"/active", `{}`, 400, nil)
	check(http.MethodPatch, "/api/profiles/"+other.ID+"/device/"+deviceID+"/active", `{"active":false}`, 404, nil)

	check(http.MethodPatch, device+"/active", `{"active":false}`, 200, nil)
	check(http.MethodPatch, gateway+"/active", `{"active":false}`, 200, nil)
	var currentDevice contracts.GetDeviceResponse
	check(http.MethodGet, base+"/device/get-device/"+deviceID, "", 200, &currentDevice)
	got := currentDevice.Device
	if got.Active || got.Name != createdDevice.Device.Name || got.LocationConfig.Region != contracts.EU868 ||
		*got.LocationConfig.Latitude != 4 || *got.LocationConfig.Longitude != 5 || *got.LocationConfig.Altitude != 6 ||
		got.PayloadConfig.Payload != "SGVsbG8=" || !got.PayloadConfig.Base64Encoded {
		t.Fatalf("unexpected device after patches: %+v", got)
	}
	var currentGateway contracts.GetGatewayResponse
	check(http.MethodGet, base+"/gateway/get-gateway/"+gatewayID, "", 200, &currentGateway)
	if got := currentGateway.Gateway; got.Active || got.Name != createdGateway.Gateway.Name ||
		*got.Latitude != 1 || *got.Longitude != 2 || *got.Altitude != 3 {
		t.Fatalf("unexpected gateway after patches: %+v", got)
	}
	var snapshot contracts.SimulationSnapshot
	check(http.MethodGet, base+"/simulation/snapshot", "", 200, &snapshot)
	if snapshot.Metrics.ActiveDevices != 0 || snapshot.Metrics.ActiveGateways != 0 {
		t.Fatalf("runtime active counts were not updated: %+v", snapshot.Metrics)
	}
	response := request(http.MethodPatch, device+"/payload", `{"payload":""}`)
	if response.Code != 200 || !bytes.Contains(response.Body.Bytes(), []byte(`"payload":""`)) {
		t.Fatalf("empty payload was not saved: %d %s", response.Code, response.Body)
	}
}

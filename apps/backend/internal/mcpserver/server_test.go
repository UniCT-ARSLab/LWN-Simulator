package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	appserver "lwn-simulator-backend/internal/server"
)

const profileID = "00000000-0000-4000-8000-000000000001"

func sessionFor(t *testing.T, endpoint *httptest.Server, writable bool) *mcp.ClientSession {
	t.Helper()
	_, port, err := net.SplitHostPort(endpoint.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	server, err := New(port, writable)
	if err != nil {
		t.Fatal(err)
	}
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	t.Cleanup(cancel)
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "lwn-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any, wantError bool) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		if wantError {
			return err.Error()
		}
		t.Fatalf("%s: %v", name, err)
	}
	if result.IsError != wantError {
		t.Fatalf("%s: unexpected result %+v", name, result)
	}
	output, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !wantError && result.StructuredContent == nil {
		t.Fatalf("%s: missing structured output", name)
	}
	return string(output)
}

func TestReadOnlyToolsValidateAndRedact(t *testing.T) {
	var requests atomic.Int32
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("read-only server sent %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/profiles/" + profileID + "/device/get-devices":
			fmt.Fprint(w, `{"devices":[{"id":"device-one","name":"First","devEUI":"1234","active":true,"class":"class-a","activation":"oota","OOTAConfig":{"appKey":"private-key"},"payloadConfig":{"payload":"private-payload"}},{"id":"device-two","name":"Second","devEUI":"5678","active":true,"class":"class-a","activation":"abp"}]}`)
		case "/api/profiles/" + profileID + "/simulation/events":
			if r.URL.Query().Get("afterSequence") != "2" || r.URL.Query().Get("limit") != "1" {
				t.Errorf("unexpected event query %s", r.URL.RawQuery)
			}
			fmt.Fprint(w, `{"events":[{"sequence":3,"timestampMilliseconds":1,"type":"gateway-packet-ingress","message":"private-event","decodedPayload":{"key":"private-decoded"}}],"lastSequence":100}`)
		default:
			http.Error(w, "internal-secret", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(endpoint.Close)
	session := sessionFor(t, endpoint, false)
	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) != 7 {
		t.Fatalf("unexpected read-only tool inventory: %d", len(tools.Tools))
	}
	for _, tool := range tools.Tools {
		if !tool.Annotations.ReadOnlyHint || tool.InputSchema == nil || tool.OutputSchema == nil {
			t.Fatalf("missing read-only annotation or schemas for %s", tool.Name)
		}
	}
	callTool(t, session, "start_simulation", map[string]any{"profileID": profileID}, true)
	callTool(t, session, "list_devices", map[string]any{}, true)
	callTool(t, session, "list_devices", map[string]any{"profileID": "../unscoped"}, true)
	callTool(t, session, "list_devices", map[string]any{"profileID": profileID, "limit": 101}, true)
	if requests.Load() != 0 {
		t.Fatal("invalid or unregistered tool calls reached the backend")
	}
	devices := callTool(t, session, "list_devices", map[string]any{"profileID": profileID, "limit": 1}, false)
	if !strings.Contains(devices, "device-one") || !strings.Contains(devices, "nextOffset") ||
		strings.Contains(devices, "private-") || strings.Contains(devices, "device-two") {
		t.Fatalf("unexpected devices page: %s", devices)
	}
	events := callTool(t, session, "list_simulation_events", map[string]any{"profileID": profileID, "afterSequence": 2, "limit": 1}, false)
	if !strings.Contains(events, `"nextAfterSequence":3`) || strings.Contains(events, "private-") {
		t.Fatalf("unexpected events page: %s", events)
	}
	failure := callTool(t, session, "list_profiles", nil, true)
	if !strings.Contains(failure, "HTTP 500") || strings.Contains(failure, "internal-secret") {
		t.Fatalf("unsafe backend error: %s", failure)
	}
}

func TestMCPUsesRealProfileRuntime(t *testing.T) {
	t.Setenv("LWN_ENV", "development")
	t.Setenv("LWN_GATEWAY_UDP_LISTEN_ADDR", "127.0.0.1:0")
	t.Chdir(t.TempDir())
	router, err := appserver.New("0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := httptest.NewServer(router)
	t.Cleanup(endpoint.Close)
	session := sessionFor(t, endpoint, true)
	inventory, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(inventory.Tools, func(tool *mcp.Tool) bool {
		return tool.Name == "control_simulation" && !tool.Annotations.ReadOnlyHint && *tool.Annotations.DestructiveHint
	}) {
		t.Fatal("write-mode tooling is missing lifecycle metadata")
	}

	send := func(path, body string) map[string]json.RawMessage {
		t.Helper()
		res, err := endpoint.Client().Post(endpoint.URL+path, "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		raw, err := io.ReadAll(res.Body)
		if err != nil || res.StatusCode >= 300 {
			t.Fatalf("setup POST %s: %s (%v)", path, raw, err)
		}
		var response map[string]json.RawMessage
		if err := json.Unmarshal(raw, &response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	createProfile := func(name string) string {
		t.Helper()
		data := send("/api/profile/create-profile", fmt.Sprintf(`{"name":%q}`, name))
		var profile struct{ ID string }
		if err := json.Unmarshal(data["profile"], &profile); err != nil || profile.ID == "" {
			t.Fatalf("missing profile ID: %v", err)
		}
		return profile.ID
	}
	first, second := createProfile("MCP test network"), createProfile("Other MCP network")
	gateway := `{"name":"Gate","type":"virtual","gatewayEUI":"A840410001001001","keepAlive":30,"latitude":37.5,"longitude":15.1,"altitude":100}`
	for _, id := range []string{first, second} {
		send("/api/profiles/"+id+"/gateway/create-gateway", gateway)
	}

	device := send("/api/profiles/"+first+"/device/create-device", `{
		"name":"MCP sensor","devEUI":"70B3D57ED0001001","class":"class-a","activation":"abp",
		"ABPConfig":{"devAddr":"26011BDA","nwkSKey":"00112233445566778899AABBCCDDEEFF","appSKey":"FFEEDDCCBBAA99887766554433221100"},
		"locationConfig":{"latitude":37.5,"longitude":15.1,"altitude":100,"region":"EU868"},
		"RX1Config":{"delay":1000,"duration":1000,"dataRateOffset":0},
		"RX2Config":{"delay":2000,"duration":1000,"channelFrequency":869525000,"dataRate":0,"ACKTimeout":2000},
		"frameConfig":{"fPort":10,"retransmission":0,"FCntUp":0,"FCntDown":0},
		"payloadConfig":{"uplinkInterval":30,"oversizedPayloadBehavior":"truncate","MType":"unconfirmed-data-up","payload":"initial","base64Encoded":false},
		"advancedConfig":{"antennaRange":1500,"ADREnabled":true}
	}`)
	var sensor struct{ ID string }
	if err := json.Unmarshal(device["device"], &sensor); err != nil || sensor.ID == "" {
		t.Fatalf("missing device ID: %v", err)
	}

	expect := func(name string, input map[string]any, value string) {
		t.Helper()
		result := callTool(t, session, name, input, false)
		if !strings.Contains(result, value) {
			t.Fatalf("%s did not return %s: %s", name, value, result)
		}
	}
	scope := map[string]any{"profileID": first}
	other := map[string]any{"profileID": second}
	expect("list_profiles", nil, first)
	expect("list_devices", scope, sensor.ID)
	expect("list_gateways", scope, "A840410001001001")
	expect("get_simulation_status", scope, `"status":"idle"`)
	expect("start_simulation", scope, `"status":"running"`)
	t.Cleanup(func() {
		response, err := endpoint.Client().Post(endpoint.URL+"/api/profiles/"+first+"/simulation/stop", "", nil)
		if err == nil {
			response.Body.Close()
		}
	})
	expect("get_simulation_activity", nil, first)
	callTool(t, session, "start_simulation", other, true)
	callTool(t, session, "queue_uplink", map[string]any{"profileID": second, "deviceID": sensor.ID}, true)
	expect("queue_uplink", map[string]any{"profileID": first, "deviceID": sensor.ID}, `"id":`)
	callTool(t, session, "queue_downlink", map[string]any{"profileID": first, "deviceID": sensor.ID, "payload": "?", "base64Encoded": true}, true)
	expect("queue_downlink", map[string]any{"profileID": first, "deviceID": sensor.ID, "payload": "AQI=", "base64Encoded": true, "fPort": 10}, `"id":`)
	expect("control_simulation", map[string]any{"profileID": first, "action": "pause"}, `"status":"paused"`)
	expect("control_simulation", map[string]any{"profileID": first, "action": "resume"}, `"status":"running"`)
	expect("control_simulation", map[string]any{"profileID": first, "action": "stop"}, `"status":"stopped"`)
	expect("list_simulation_events", scope, "simulation-started")
	runs := callTool(t, session, "list_simulation_runs", scope, false)
	var persisted struct {
		StructuredContent struct {
			Items []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"items"`
		} `json:"structuredContent"`
	}
	if err := json.Unmarshal([]byte(runs), &persisted); err != nil {
		t.Fatal(err)
	}
	items := persisted.StructuredContent.Items
	if len(items) != 1 || items[0].Status != "stopped" {
		t.Fatalf("unexpected run history: %+v", items)
	}
	expect("list_simulation_events", map[string]any{"profileID": first, "runID": items[0].ID}, "simulation-started")
	expect("start_simulation", other, `"status":"running"`)
	expect("control_simulation", map[string]any{"profileID": second, "action": "stop"}, `"status":"stopped"`)
}

func TestBackendDoesNotFollowRedirects(t *testing.T) {
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("MCP followed a backend redirect")
		http.Error(w, "unexpected destination", http.StatusInternalServerError)
	}))
	t.Cleanup(destination.Close)
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(endpoint.Close)
	session := sessionFor(t, endpoint, false)
	if output := callTool(t, session, "list_profiles", nil, true); !strings.Contains(output, "HTTP 307") {
		t.Fatalf("unexpected redirect response: %s", output)
	}
	if _, err := New("example.com:8080", false); err == nil {
		t.Fatal("MCP accepted a non-port backend address")
	}
}

func TestStartUsesSavedGatewayBridge(t *testing.T) {
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "GET /api/profiles/" + profileID + "/configuration/gateway-bridge":
			fmt.Fprint(w, `{"enabled":true,"address":"127.0.0.1","port":17800}`)
		case "POST /api/profiles/" + profileID + "/simulation/start":
			var request struct {
				Speed         float64
				Seed          int64
				GatewayBridge struct {
					Enabled bool
					Address string
					Port    int
				}
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Speed != 1 ||
				request.Seed != 42 || !request.GatewayBridge.Enabled ||
				request.GatewayBridge.Address != "127.0.0.1" || request.GatewayBridge.Port != 17800 {
				t.Errorf("unexpected simulation start: %+v (%v)", request, err)
			}
			fmt.Fprint(w, `{"state":{"status":"running","speed":1},"metrics":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(endpoint.Close)
	session := sessionFor(t, endpoint, true)
	callTool(t, session, "start_simulation", map[string]any{"profileID": profileID, "seed": 42}, false)
}

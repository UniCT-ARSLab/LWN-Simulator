package mcpserver

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type startInput struct {
	profileInput
	Speed *float64 `json:"speed,omitempty" jsonschema:"Positive speed multiplier; defaults to 1 and must be 1 with external gateway transport"`
	Seed  *int64   `json:"seed,omitempty" jsonschema:"Optional seed for repeatable simulation runs"`
}

type controlInput struct {
	profileInput
	Action string `json:"action" jsonschema:"One of pause, resume, stop"`
}

type uplinkInput struct {
	profileInput
	DeviceID string `json:"deviceID" jsonschema:"UUID of an active device in this profile"`
}

type downlinkInput struct {
	uplinkInput
	Payload       string `json:"payload" jsonschema:"Nonempty UTF-8 text or an already Base64-encoded payload"`
	Base64Encoded bool   `json:"base64Encoded,omitempty" jsonschema:"Whether payload uses standard Base64 instead of UTF-8"`
	FPort         *int   `json:"fPort,omitempty" jsonschema:"Application port 1–223 for a LoRaWAN device; omit for raw LoRa"`
}

func registerWriteTools(server *mcp.Server, api backend) {
	addTool(server, "start_simulation", "Start simulation", "Start a run with an active device or gateway. Uses the profile’s saved bridge settings.", false, false,
		func(ctx context.Context, input startInput) (statusResult, error) {
			path, err := profilePath(input.ProfileID, "")
			if err != nil {
				return statusResult{}, err
			}
			config := contracts.SimulationConfig{Speed: 1, Seed: input.Seed}
			if input.Speed != nil {
				config.Speed = *input.Speed
				if config.Speed <= 0 {
					return statusResult{}, errors.New("speed must be positive")
				}
			}
			bridge, err := call[contracts.GatewayBridgeConfig](ctx, api, http.MethodGet, path+"/configuration/gateway-bridge", nil)
			if err != nil {
				return statusResult{}, err
			}
			config.GatewayBridge = &bridge
			return call[statusResult](ctx, api, http.MethodPost, path+"/simulation/start", config)
		})
	addTool(server, "control_simulation", "Pause, resume or stop a simulation", "Change the lifecycle of a run in the specified profile.", false, true,
		func(ctx context.Context, input controlInput) (statusResult, error) {
			switch input.Action {
			case "pause", "resume", "stop":
			default:
				return statusResult{}, errors.New("action must be pause, resume or stop")
			}
			path, err := profilePath(input.ProfileID, "/simulation/"+input.Action)
			if err != nil {
				return statusResult{}, err
			}
			return call[statusResult](ctx, api, http.MethodPost, path, nil)
		})
	addTool(server, "queue_uplink", "Queue an uplink", "Schedule one uplink from an active device while the simulation is running.", false, false,
		func(ctx context.Context, input uplinkInput) (contracts.SimulationActionResponse, error) {
			path, err := profilePath(input.ProfileID, "/simulation/uplinks")
			if err != nil {
				return contracts.SimulationActionResponse{}, err
			}
			device, err := checkedID(input.DeviceID, "deviceID")
			if err != nil {
				return contracts.SimulationActionResponse{}, err
			}
			return call[contracts.SimulationActionResponse](ctx, api, http.MethodPost, path, contracts.SimulationUplinkRequest{DeviceID: device})
		})
	addTool(server, "queue_downlink", "Queue a downlink", "Queue application bytes for an active device. Use fPort for LoRaWAN and omit it for raw LoRa.", false, false,
		func(ctx context.Context, input downlinkInput) (contracts.SimulationActionResponse, error) {
			path, err := profilePath(input.ProfileID, "/simulation/downlinks")
			if err != nil {
				return contracts.SimulationActionResponse{}, err
			}
			device, err := checkedID(input.DeviceID, "deviceID")
			if err != nil {
				return contracts.SimulationActionResponse{}, err
			}
			if input.FPort != nil && (*input.FPort < 1 || *input.FPort > 223) {
				return contracts.SimulationActionResponse{}, errors.New("fPort must be between 1 and 223")
			}
			if len(input.Payload) > base64.StdEncoding.EncodedLen(255) {
				return contracts.SimulationActionResponse{}, errors.New("payload is too large")
			}
			payload := []byte(input.Payload)
			if input.Base64Encoded {
				payload, err = base64.StdEncoding.Strict().DecodeString(input.Payload)
				if err != nil {
					return contracts.SimulationActionResponse{}, errors.New("payload must use standard Base64")
				}
			}
			if len(payload) == 0 || len(payload) > 255 {
				return contracts.SimulationActionResponse{}, errors.New("payload must contain between 1 and 255 bytes")
			}
			request := contracts.SimulationDownlinkRequest{DeviceID: device, Payload: &payload, FPort: input.FPort}
			return call[contracts.SimulationActionResponse](ctx, api, http.MethodPost, path, request)
		})
}

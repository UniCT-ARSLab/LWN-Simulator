package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type profileInput struct {
	ProfileID string `json:"profileID" jsonschema:"Profile UUID returned by list_profiles"`
}

type pageInput struct {
	Offset int `json:"offset,omitempty" jsonschema:"Zero-based offset; defaults to zero"`
	Limit  int `json:"limit,omitempty" jsonschema:"Maximum entries, between 1 and 100; defaults to 50"`
}

type profilePage struct {
	profileInput
	pageInput
}

type page[T any] struct {
	Items      []T  `json:"items"`
	Total      int  `json:"total"`
	NextOffset *int `json:"nextOffset,omitempty"`
}

type deviceSummary struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	DevEUI           string `json:"devEUI"`
	Active           bool   `json:"active"`
	Class            string `json:"class"`
	Activation       string `json:"activation"`
	RadioProtocol    string `json:"radioProtocol,omitempty"`
	RadioReceiveMode string `json:"radioReceiveMode,omitempty"`
}

type gatewaySummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	GatewayEUI string `json:"gatewayEUI"`
	Active     bool   `json:"active"`
	Type       string `json:"type"`
}

type statusResult struct {
	State   contracts.SimulationState   `json:"state"`
	Metrics contracts.SimulationMetrics `json:"metrics"`
}

type eventInput struct {
	profileInput
	RunID         string `json:"runID,omitempty" jsonschema:"Saved run UUID; omit to read events for the current simulation"`
	AfterSequence int64  `json:"afterSequence,omitempty" jsonschema:"Exclusive event sequence cursor; defaults to zero"`
	Limit         int    `json:"limit,omitempty" jsonschema:"Maximum events, between 1 and 100; defaults to 50"`
}

type eventSummary struct {
	Sequence              int64    `json:"sequence"`
	TimestampMilliseconds int64    `json:"timestampMilliseconds"`
	Type                  string   `json:"type"`
	DeviceID              string   `json:"deviceID,omitempty"`
	GatewayID             string   `json:"gatewayID,omitempty"`
	RadioProtocol         string   `json:"radioProtocol,omitempty"`
	RadioLossReason       string   `json:"radioLossReason,omitempty"`
	UplinkOutcome         string   `json:"uplinkOutcome,omitempty"`
	PayloadSize           *int64   `json:"payloadSize,omitempty"`
	RSSI                  *float64 `json:"rssi,omitempty"`
	SNR                   *float64 `json:"snr,omitempty"`
}

type eventPage struct {
	Events            []eventSummary `json:"events"`
	NextAfterSequence int64          `json:"nextAfterSequence"`
}

func pageLimit(limit int) (int, error) {
	if limit < 0 || limit > 100 {
		return 0, errors.New("limit must be between 1 and 100")
	}
	if limit == 0 {
		return 50, nil
	}
	return limit, nil
}

func paginate[T any](input pageInput, fetch func() ([]T, error)) (page[T], error) {
	limit, err := pageLimit(input.Limit)
	if err != nil {
		return page[T]{}, err
	}
	if input.Offset < 0 {
		return page[T]{}, errors.New("offset must be nonnegative")
	}
	items, err := fetch()
	if err != nil {
		return page[T]{}, err
	}
	lower := min(input.Offset, len(items))
	upper := lower + min(limit, len(items)-lower)
	result := page[T]{Items: append([]T{}, items[lower:upper]...), Total: len(items)}
	if upper < len(items) {
		result.NextOffset = &upper
	}
	return result, nil
}

func registerReadTools(server *mcp.Server, api backend) {
	addTool(server, "list_profiles", "List profiles", "List simulator profiles and their IDs.", true, false,
		func(ctx context.Context, input pageInput) (page[contracts.Profile], error) {
			return paginate(input, func() ([]contracts.Profile, error) {
				response, err := call[contracts.GetProfilesResponse](ctx, api, http.MethodGet, "/profile/get-profiles", nil)
				return response.Profiles, err
			})
		})
	addTool(server, "list_devices", "List devices", "List device identities, activation, radio mode and active status in a profile.", true, false,
		func(ctx context.Context, input profilePage) (page[deviceSummary], error) {
			return paginate(input.pageInput, func() ([]deviceSummary, error) {
				path, err := profilePath(input.ProfileID, "/device/get-devices")
				if err != nil {
					return nil, err
				}
				response, err := call[struct {
					Devices []deviceSummary `json:"devices"`
				}](ctx, api, http.MethodGet, path, nil)
				return response.Devices, err
			})
		})
	addTool(server, "list_gateways", "List gateways", "List gateway identities, types and active status in a profile.", true, false,
		func(ctx context.Context, input profilePage) (page[gatewaySummary], error) {
			return paginate(input.pageInput, func() ([]gatewaySummary, error) {
				path, err := profilePath(input.ProfileID, "/gateway/get-gateways")
				if err != nil {
					return nil, err
				}
				response, err := call[struct {
					Gateways []gatewaySummary `json:"gateways"`
				}](ctx, api, http.MethodGet, path, nil)
				return response.Gateways, err
			})
		})
	addTool(server, "get_simulation_activity", "Get active simulation", "Find which profile currently has a running or paused simulation.", true, false,
		func(ctx context.Context, _ struct{}) (contracts.SimulationActivity, error) {
			return call[contracts.SimulationActivity](ctx, api, http.MethodGet, "/simulation/activity", nil)
		})
	addTool(server, "get_simulation_status", "Get simulation status", "Read the simulation clock, lifecycle state and packet metrics for a profile.", true, false,
		func(ctx context.Context, input profileInput) (statusResult, error) {
			path, err := profilePath(input.ProfileID, "/simulation/snapshot")
			if err != nil {
				return statusResult{}, err
			}
			status, err := call[statusResult](ctx, api, http.MethodGet, path, nil)
			if errors.Is(err, errNoSimulation) {
				status.State.Status = contracts.SimulationStatusIdle
				return status, nil
			}
			return status, err
		})
	addTool(server, "list_simulation_runs", "List saved runs", "Read saved run IDs, durations, status and telemetry for a profile.", true, false,
		func(ctx context.Context, input profilePage) (page[contracts.SimulationRunSummary], error) {
			return paginate(input.pageInput, func() ([]contracts.SimulationRunSummary, error) {
				path, err := profilePath(input.ProfileID, "/simulation/logs")
				if err != nil {
					return nil, err
				}
				type runsResponse struct {
					Runs []struct {
						Summary contracts.SimulationRunSummary `json:"summary"`
					} `json:"runs"`
				}
				response, err := call[runsResponse](ctx, api, http.MethodGet, path, nil)
				items := make([]contracts.SimulationRunSummary, len(response.Runs))
				for index, run := range response.Runs {
					items[index] = run.Summary
				}
				return items, err
			})
		})
	addTool(server, "list_simulation_events", "List simulation events", "Read event metadata for the current or a saved run. Continue with nextAfterSequence.", true, false,
		func(ctx context.Context, input eventInput) (eventPage, error) { return listEvents(ctx, api, input) })
}

func listEvents(ctx context.Context, api backend, input eventInput) (eventPage, error) {
	limit, err := pageLimit(input.Limit)
	if err != nil {
		return eventPage{}, err
	}
	if input.AfterSequence < 0 {
		return eventPage{}, errors.New("afterSequence must be nonnegative")
	}
	suffix := "/simulation/events"
	if input.RunID != "" {
		run, err := checkedID(input.RunID, "runID")
		if err != nil {
			return eventPage{}, err
		}
		suffix = "/simulation/logs/" + run + "/events"
	}
	path, err := profilePath(input.ProfileID, suffix)
	if err != nil {
		return eventPage{}, err
	}
	query := url.Values{"limit": {strconv.Itoa(limit)}, "afterSequence": {strconv.FormatInt(input.AfterSequence, 10)}}
	response, err := call[eventPage](ctx, api, http.MethodGet, path+"?"+query.Encode(), nil)
	if response.Events == nil {
		response.Events = []eventSummary{}
	}
	response.NextAfterSequence = input.AfterSequence
	for _, event := range response.Events {
		response.NextAfterSequence = max(response.NextAfterSequence, event.Sequence)
	}
	return response, err
}

package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxResponseBytes = 16 << 20

var errNoSimulation = errors.New("no simulation has started in this profile")

type backend struct {
	base string
	http *http.Client
}

func newBackend(port string) (backend, error) {
	number, err := strconv.ParseUint(port, 10, 16)
	if err != nil || number == 0 {
		return backend{}, errors.New("backend port must be an integer from 1 to 65535")
	}
	return backend{
		base: "http://" + net.JoinHostPort("127.0.0.1", strconv.FormatUint(number, 10)) + "/api",
		http: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				DialContext:         (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
				MaxIdleConnsPerHost: 2,
				IdleConnTimeout:     30 * time.Second,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}, nil
}

func checkedID(value, field string) (string, error) {
	id, err := uuid.Parse(value)
	if err != nil || len(value) != 36 {
		return "", fmt.Errorf("%s must be a UUID", field)
	}
	return id.String(), nil
}

func profilePath(id, suffix string) (string, error) {
	profile, err := checkedID(id, "profileID")
	if err != nil {
		return "", err
	}
	return "/profiles/" + profile + suffix, nil
}

func call[T any](ctx context.Context, api backend, method, path string, input any) (T, error) {
	var output T
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return output, errors.New("could not encode the simulator request")
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, api.base+path, body)
	if err != nil {
		return output, errors.New("could not create the simulator request")
	}
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := api.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return output, ctx.Err()
		}
		return output, errors.New("could not contact the local simulator backend; check that it is running on the configured port")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		switch response.StatusCode {
		case http.StatusBadRequest, http.StatusUnprocessableEntity:
			return output, errors.New("the simulator rejected the input; check the tool arguments and profile configuration")
		case http.StatusNotFound:
			return output, errors.New("the simulator could not find the requested profile or object")
		case http.StatusConflict:
			if method == http.MethodGet && strings.HasSuffix(path, "/simulation/snapshot") {
				return output, errNoSimulation
			}
			return output, errors.New("the operation conflicts with the current simulation state; inspect its status and active profile")
		default:
			return output, fmt.Errorf("the local simulator returned HTTP %d", response.StatusCode)
		}
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return output, errors.New("could not read the simulator response")
	}
	if len(raw) > maxResponseBytes {
		return output, errors.New("the simulator response exceeded 16 MiB")
	}
	if err := json.Unmarshal(raw, &output); err != nil {
		return output, errors.New("the simulator returned an invalid JSON response")
	}
	return output, nil
}

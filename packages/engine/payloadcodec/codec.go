package payloadcodec

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

const defaultExecutionTimeout = 50 * time.Millisecond

// Input is the application payload passed to a device decoder.
type Input struct {
	Bytes []byte
	FPort int
}

// Decode executes a validated payload codec and returns its decoded data.
// The script is evaluated in a fresh runtime with no application objects
// exposed to it and is interrupted when it exceeds the execution timeout.
func Decode(script string, input Input) (map[string]interface{}, error) {
	return DecodeWithTimeout(script, input, defaultExecutionTimeout)
}

// DecodeWithTimeout is Decode with an explicit execution limit, primarily to
// make runtime behavior deterministic in tests.
func DecodeWithTimeout(script string, input Input, timeout time.Duration) (map[string]interface{}, error) {
	if timeout <= 0 {
		return nil, fmt.Errorf("payload codec execution timeout must be positive")
	}

	runtime := goja.New()
	interrupt := time.AfterFunc(timeout, func() {
		runtime.Interrupt("payload codec execution timed out")
	})
	defer interrupt.Stop()

	if _, err := runtime.RunString(script); err != nil {
		return nil, fmt.Errorf("load payload codec: %w", err)
	}

	decoder, ok := goja.AssertFunction(runtime.Get("decodeUplink"))
	if !ok {
		return nil, fmt.Errorf("payload codec does not expose decodeUplink")
	}
	bytes := make([]interface{}, len(input.Bytes))
	for index, value := range input.Bytes {
		bytes[index] = int(value)
	}

	value, err := decoder(goja.Undefined(), runtime.ToValue(map[string]interface{}{
		"bytes": runtime.NewArray(bytes...),
		"fPort": input.FPort,
	}))
	if err != nil {
		return nil, fmt.Errorf("decodeUplink failed: %w", err)
	}

	if err := runtime.Set("__payloadCodecResult", value); err != nil {
		return nil, fmt.Errorf("prepare decoded payload: %w", err)
	}
	encoded, err := runtime.RunString("JSON.stringify(__payloadCodecResult)")
	if err != nil {
		return nil, fmt.Errorf("encode decoded payload: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(encoded.String()), &result); err != nil {
		return nil, fmt.Errorf("decoded payload must be a JSON object: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("decoded payload must be a JSON object")
	}
	if errors, ok := result["errors"].([]interface{}); ok && len(errors) > 0 {
		return nil, fmt.Errorf("decodeUplink reported errors: %v", errors)
	}
	if data, exists := result["data"]; exists {
		decodedData, ok := data.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("decoded payload data must be a JSON object")
		}
		return decodedData, nil
	}
	return result, nil
}

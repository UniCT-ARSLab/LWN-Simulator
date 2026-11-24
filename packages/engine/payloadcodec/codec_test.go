package payloadcodec

import (
	"strings"
	"testing"
	"time"
)

func TestDecodeReturnsDataObject(t *testing.T) {
	decoded, err := Decode(`function decodeUplink(input) {
  return { data: { temperature: input.bytes[0], port: input.fPort } };
}`, Input{Bytes: []byte{21}, FPort: 10})
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded["temperature"] != float64(21) || decoded["port"] != float64(10) {
		t.Fatalf("unexpected decoded data: %#v", decoded)
	}
}

func TestDecodeAcceptsDirectObjectResult(t *testing.T) {
	decoded, err := Decode("function decodeUplink() { return { value: 'ok' }; }", Input{})
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if decoded["value"] != "ok" {
		t.Fatalf("unexpected decoded data: %#v", decoded)
	}
}

func TestDecodeReportsRuntimeErrors(t *testing.T) {
	_, err := Decode("function decodeUplink() { throw new Error('bad payload'); }", Input{})
	if err == nil || !strings.Contains(err.Error(), "decodeUplink failed") {
		t.Fatalf("expected decoder error, got %v", err)
	}
}

func TestDecodeInterruptsLongRunningCode(t *testing.T) {
	_, err := DecodeWithTimeout("function decodeUplink() { while (true) {} }", Input{}, 5*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestDecodeRejectsNonObjectData(t *testing.T) {
	_, err := Decode("function decodeUplink() { return { data: 42 }; }", Input{})
	if err == nil || !strings.Contains(err.Error(), "data must be a JSON object") {
		t.Fatalf("expected object validation error, got %v", err)
	}
}

func TestDecodeSupportsAllAcceptedDeclarations(t *testing.T) {
	for _, script := range []string{
		`function decodeUplink(input) { return { data: { port: input.fPort } }; }`,
		`const decodeUplink = (input) => ({ data: { port: input.fPort } });`,
		`let decodeUplink = function(input) { return { data: { port: input.fPort } }; };`,
		`var decodeUplink = input => ({ data: { port: input.fPort } });`,
	} {
		t.Run(script, func(t *testing.T) {
			decoded, err := Decode(script, Input{FPort: 7})
			if err != nil || decoded["port"] != float64(7) {
				t.Fatalf("accepted declaration could not run: %#v, %v", decoded, err)
			}
		})
	}
}

func TestDecodeProvidesAnIsolatedJavaScriptArray(t *testing.T) {
	input := Input{Bytes: []byte{0, 128, 255}, FPort: 1}
	decoded, err := Decode(`function decodeUplink(input) {
  if (!Array.isArray(input.bytes)) throw new Error('bytes must be a JavaScript array');
  const result = input.bytes.map(value => value + 1);
  input.bytes[0] = 99;
  input.bytes.push(42);
  return { data: { result, length: input.bytes.length } };
}`, input)
	if err != nil {
		t.Fatal(err)
	}
	if input.Bytes[0] != 0 || len(input.Bytes) != 3 {
		t.Fatal("codec mutated the original application payload")
	}
	if decoded["length"] != float64(4) {
		t.Fatalf("array operations did not work: %#v", decoded)
	}
}

func TestDecodeRejectsInvalidResultsAndRecovers(t *testing.T) {
	for _, script := range []string{
		`function decodeUplink() { return null; }`,
		`function decodeUplink() { return undefined; }`,
		`function decodeUplink() { return []; }`,
		`function decodeUplink() { return 42; }`,
		`function decodeUplink() { return { errors: ['bad payload'] }; }`,
		`function decodeUplink() { const result = {}; result.self = result; return { data: result }; }`,
	} {
		t.Run(script, func(t *testing.T) {
			if result, err := Decode(script, Input{}); err == nil {
				t.Fatalf("invalid decoder result was accepted: %#v", result)
			}
		})
	}
	if _, err := Decode(`function decodeUplink() { return { data: { recovered: true } }; }`, Input{}); err != nil {
		t.Fatalf("a failing codec affected the next decoder: %v", err)
	}
}

func TestDecodeInterruptsSerialization(t *testing.T) {
	for _, script := range []string{
		`function decodeUplink() { return { get data() { while (true) {} } }; }`,
		`function decodeUplink() { return { toJSON() { while (true) {} } }; }`,
	} {
		t.Run(script, func(t *testing.T) {
			if _, err := DecodeWithTimeout(script, Input{}, 10*time.Millisecond); err == nil {
				t.Fatal("serialization timeout was ignored")
			}
		})
	}
}

func TestDecodeDoesNotShareGlobalsBetweenDevicesOrCalls(t *testing.T) {
	script := `let counter = 0; function decodeUplink(input) { return { data: { counter: ++counter, requireType: typeof require, fetchType: typeof fetch, processType: typeof process } }; }`
	for index := 0; index < 2; index++ {
		decoded, err := Decode(script, Input{})
		if err != nil || decoded["counter"] != float64(1) || decoded["requireType"] != "undefined" || decoded["fetchType"] != "undefined" || decoded["processType"] != "undefined" {
			t.Fatalf("codec runtime was not isolated: %#v, %v", decoded, err)
		}
	}
}

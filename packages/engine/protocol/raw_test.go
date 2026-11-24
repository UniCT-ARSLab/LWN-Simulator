package protocol

import (
	"errors"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
)

func TestRawLoRaHandlerRoundTripsPayloadWithoutLoRaWANEnvelope(t *testing.T) {
	handler := RawLoRaHandler{}
	payload := []byte{0x01, 0x00, 0xff, 0x42}

	encoded, err := handler.EncodeDataFrame(DataFrame{Payload: payload})
	if err != nil {
		t.Fatalf("encode raw frame: %v", err)
	}
	if string(encoded) != string(payload) {
		t.Fatalf("raw frame was changed during encoding: %x", encoded)
	}

	decoded, err := handler.Decode(encoded)
	if err != nil {
		t.Fatalf("decode raw frame: %v", err)
	}
	if string(decoded.Raw) != string(payload) || string(decoded.FRMPayload) != string(payload) {
		t.Fatalf("unexpected decoded raw frame: %+v", decoded)
	}
	if decoded.MType != 7 {
		t.Fatalf("raw frame must not be classified as a LoRaWAN frame: MType=%d", decoded.MType)
	}
	if handler.Protocol() != contracts.RawLoRa {
		t.Fatalf("unexpected protocol: %s", handler.Protocol())
	}
}

func TestRawLoRaHandlerRejectsLoRaWANOnlyOperations(t *testing.T) {
	handler := RawLoRaHandler{}

	if _, err := handler.EncodeJoinRequest(JoinRequest{}); !errors.Is(err, ErrUnsupportedOperation) {
		t.Fatalf("expected join request to be unsupported, got %v", err)
	}
	if _, err := handler.EncodeJoinAccept(JoinAccept{}); !errors.Is(err, ErrUnsupportedOperation) {
		t.Fatalf("expected join accept to be unsupported, got %v", err)
	}
	if _, err := handler.EncodeMACCommands(nil); !errors.Is(err, ErrUnsupportedOperation) {
		t.Fatalf("expected MAC commands to be unsupported, got %v", err)
	}
	if handler.VerifyDataMIC(lorawan.Packet{}, nil, 0) {
		t.Fatal("raw LoRa must not report a LoRaWAN MIC as valid")
	}
}

func TestRawLoRaHandlerRejectsEmptyPayloadAndOwnsItsBuffers(t *testing.T) {
	handler := RawLoRaHandler{}
	if _, err := handler.EncodeDataFrame(DataFrame{}); err == nil {
		t.Fatal("empty raw transmission was accepted")
	}
	if _, err := handler.Decode(nil); err == nil {
		t.Fatal("empty raw reception was accepted")
	}
	if _, err := handler.DecryptPayload(lorawan.Packet{}, nil, 0); err == nil {
		t.Fatal("empty raw payload was accepted")
	}
	payload := []byte{0, 1, 255}
	encoded, err := handler.EncodeDataFrame(DataFrame{Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	payload[0] = 99
	if encoded[0] != 0 {
		t.Fatal("encoded raw payload shares caller buffer")
	}
	for _, packet := range []lorawan.Packet{{Raw: encoded}, {FRMPayload: encoded}} {
		decoded, err := handler.DecryptPayload(packet, nil, 0)
		if err != nil || string(decoded) != string(encoded) {
			t.Fatalf("raw bytes changed: %x, %v", decoded, err)
		}
		decoded[0] = 42
		if encoded[0] != 0 {
			t.Fatal("returned raw payload shares packet buffer")
		}
	}
}

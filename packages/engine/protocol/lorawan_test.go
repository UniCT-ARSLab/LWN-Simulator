package protocol

import (
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestLoRaWANHandlerEncodesAndDecodesDataFrames(t *testing.T) {
	key := make([]byte, 16)
	port := byte(1)
	handler := LoRaWANHandler{}

	frame, err := handler.EncodeDataFrame(DataFrame{
		DevAddr: 0x26011BDA,
		FCnt:    1,
		FPort:   &port,
		Payload: []byte{1, 2, 3},
		NwkSKey: key,
		AppSKey: key,
	})
	if err != nil {
		t.Fatalf("encode data frame: %v", err)
	}
	decoded, err := handler.Decode(frame)
	if err != nil {
		t.Fatalf("decode data frame: %v", err)
	}
	if decoded.DevAddr != 0x26011BDA || decoded.FCnt != 1 {
		t.Fatalf("unexpected decoded frame: %+v", decoded)
	}
	if handler.Protocol() != contracts.LoRaWAN {
		t.Fatalf("unexpected protocol: %s", handler.Protocol())
	}
}

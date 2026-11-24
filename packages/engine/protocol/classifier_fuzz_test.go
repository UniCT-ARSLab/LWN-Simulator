package protocol

import (
	"bytes"
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"testing"
)

func FuzzReceiveModeClassification(f *testing.F) {
	for _, payload := range [][]byte{nil, {}, {0}, {255, 0, 128}, []byte("Hello LoRa world"), make([]byte, 23), make([]byte, 255)} {
		f.Add(payload)
	}
	frame, err := (LoRaWANHandler{}).EncodeDataFrame(DataFrame{DevAddr: 0x26011bda, Payload: []byte("data"), FPort: bytePointer(1), NwkSKey: make([]byte, 16), AppSKey: make([]byte, 16)})
	if err != nil {
		f.Fatal(err)
	}
	f.Add(frame)
	f.Fuzz(func(t *testing.T, payload []byte) {
		raw, rawErr := Classify(payload, contracts.ReceiveRawLoRa)
		strict, strictErr := Classify(payload, contracts.ReceiveLoRaWAN)
		auto, autoErr := Classify(payload, contracts.ReceiveAuto)
		if len(payload) == 0 {
			if rawErr == nil || strictErr == nil || autoErr == nil {
				t.Fatal("empty frame accepted")
			}
			return
		}
		if rawErr != nil || autoErr != nil {
			t.Fatal("nonempty raw frame rejected")
		}
		if !bytes.Equal(raw.Raw, payload) {
			t.Fatal("raw bytes changed")
		}
		if strictErr == nil {
			if auto.Protocol != contracts.LoRaWAN || strict.Protocol != auto.Protocol {
				t.Fatal("valid LoRaWAN misclassified")
			}
		} else if auto.Protocol != contracts.RawLoRa || !bytes.Equal(auto.Raw, payload) {
			t.Fatal("malformed LoRaWAN lost instead of raw fallback")
		}
		raw.Raw[0] ^= 255
		if raw.Raw[0] == payload[0] {
			t.Fatal("raw result aliases source bytes")
		}
	})
}

func TestClassifyRejectsUnknownReceiveMode(t *testing.T) {
	if _, err := Classify([]byte("payload"), contracts.RadioReceiveMode("unknown")); err == nil {
		t.Fatal("unknown receive mode was accepted")
	}
}

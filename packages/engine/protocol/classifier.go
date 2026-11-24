package protocol

import (
	"fmt"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
)

// ClassifiedFrame is the result of decoding a radio payload at the protocol
// boundary. The LoRaWAN packet is retained for the existing session logic;
// raw payloads remain available without being interpreted as LoRaWAN.
type ClassifiedFrame struct {
	Protocol contracts.RadioProtocol
	Packet   lorawan.Packet
	Raw      []byte
}

// Classify decodes a payload according to the receive mode configured for the
// listening device. Auto mode first accepts structurally valid LoRaWAN frames
// and falls back to a raw LoRa frame for every other payload.
func Classify(payload []byte, mode contracts.RadioReceiveMode) (ClassifiedFrame, error) {
	if len(payload) == 0 {
		return ClassifiedFrame{}, fmt.Errorf("radio frame payload cannot be empty")
	}

	if mode == "" {
		mode = contracts.ReceiveLoRaWAN
	}

	switch mode {
	case contracts.ReceiveLoRaWAN:
		packet, err := (LoRaWANHandler{}).Decode(payload)
		if err != nil {
			return ClassifiedFrame{}, fmt.Errorf("decode LoRaWAN frame: %w", err)
		}
		return ClassifiedFrame{Protocol: contracts.LoRaWAN, Packet: packet}, nil
	case contracts.ReceiveRawLoRa:
		packet, err := (RawLoRaHandler{}).Decode(payload)
		if err != nil {
			return ClassifiedFrame{}, fmt.Errorf("decode raw LoRa frame: %w", err)
		}
		return ClassifiedFrame{
			Protocol: contracts.RawLoRa,
			Packet:   packet,
			Raw:      append([]byte(nil), payload...),
		}, nil
	case contracts.ReceiveAuto:
		if packet, err := (LoRaWANHandler{}).Decode(payload); err == nil {
			return ClassifiedFrame{Protocol: contracts.LoRaWAN, Packet: packet}, nil
		}
		packet, err := (RawLoRaHandler{}).Decode(payload)
		if err != nil {
			return ClassifiedFrame{}, fmt.Errorf("decode radio frame: %w", err)
		}
		return ClassifiedFrame{
			Protocol: contracts.RawLoRa,
			Packet:   packet,
			Raw:      append([]byte(nil), payload...),
		}, nil
	default:
		return ClassifiedFrame{}, fmt.Errorf("unsupported radio receive mode %q", mode)
	}
}

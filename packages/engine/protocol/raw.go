package protocol

import (
	"errors"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

// ErrUnsupportedOperation indicates that an operation only meaningful to
// LoRaWAN was requested from the raw LoRa adapter.
var ErrUnsupportedOperation = errors.New("operation is not supported by raw LoRa")

// RawLoRaHandler transports the complete payload as a raw LoRa frame. It does
// not add LoRaWAN headers, counters, MICs, or application ports.
type RawLoRaHandler struct{}

func (RawLoRaHandler) Protocol() contracts.RadioProtocol {
	return contracts.RawLoRa
}

func (RawLoRaHandler) EncodeJoinRequest(JoinRequest) ([]byte, error) {
	return nil, ErrUnsupportedOperation
}

func (RawLoRaHandler) EncodeJoinAccept(JoinAccept) ([]byte, error) {
	return nil, ErrUnsupportedOperation
}

func (RawLoRaHandler) EncodeDataFrame(request DataFrame) ([]byte, error) {
	if len(request.Payload) == 0 {
		return nil, errors.New("raw LoRa frame payload cannot be empty")
	}
	return append([]byte(nil), request.Payload...), nil
}

func (RawLoRaHandler) EncodeMACCommands([]types.MACCommand) ([]byte, error) {
	return nil, ErrUnsupportedOperation
}

func (RawLoRaHandler) Decode(payload []byte) (lorawan.Packet, error) {
	if len(payload) == 0 {
		return lorawan.Packet{}, errors.New("raw LoRa frame payload cannot be empty")
	}
	return lorawan.Packet{
		// Keep the temporary compatibility value outside all LoRaWAN MTypes so
		// callers cannot mistake a raw frame for a join or data frame.
		MType:      7,
		FRMPayload: append([]byte(nil), payload...),
		Raw:        append([]byte(nil), payload...),
	}, nil
}

func (RawLoRaHandler) DecryptPayload(packet lorawan.Packet, _ []byte, _ byte) ([]byte, error) {
	if len(packet.Raw) == 0 && len(packet.FRMPayload) == 0 {
		return nil, errors.New("raw LoRa frame payload cannot be empty")
	}
	if len(packet.FRMPayload) > 0 {
		return append([]byte(nil), packet.FRMPayload...), nil
	}
	return append([]byte(nil), packet.Raw...), nil
}

func (RawLoRaHandler) VerifyDataMIC(lorawan.Packet, []byte, byte) bool {
	return false
}

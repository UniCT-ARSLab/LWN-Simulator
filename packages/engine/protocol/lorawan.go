package protocol

import (
	"fmt"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

// LoRaWANHandler adapts the existing LoRaWAN encoder and decoder to the
// protocol boundary used by the radio runtime.
type LoRaWANHandler struct{}

func (LoRaWANHandler) Protocol() contracts.RadioProtocol {
	return contracts.LoRaWAN
}

func (LoRaWANHandler) EncodeJoinRequest(request JoinRequest) ([]byte, error) {
	return lorawan.BuildJoinRequest(lorawan.JoinRequestOptions{
		JoinEUI:  request.JoinEUI,
		DevEUI:   request.DevEUI,
		DevNonce: request.DevNonce,
		AppKey:   request.AppKey,
	})
}

func (LoRaWANHandler) EncodeJoinAccept(request JoinAccept) ([]byte, error) {
	return lorawan.BuildJoinAccept(lorawan.JoinAcceptOptions{
		AppNonce:          request.AppNonce,
		NetID:             request.NetID,
		DevAddr:           request.DevAddr,
		RX2DataRate:       request.RX2DataRate,
		RX1DataRateOffset: request.RX1DataRateOffset,
		RXDelay:           request.RXDelay,
		AppKey:            request.AppKey,
	})
}

func (LoRaWANHandler) EncodeDataFrame(request DataFrame) ([]byte, error) {
	return lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr:   request.DevAddr,
		FCnt:      request.FCnt,
		FPort:     request.FPort,
		Payload:   request.Payload,
		Confirmed: request.Confirmed,
		ACK:       request.ACK,
		FPending:  request.FPending,
		ADR:       request.ADR,
		Direction: request.Direction,
		NwkSKey:   request.NwkSKey,
		AppSKey:   request.AppSKey,
		FOpts:     request.FOpts,
	})
}

func (LoRaWANHandler) EncodeMACCommands(commands []types.MACCommand) ([]byte, error) {
	return lorawan.EncodeMACCommands(commands)
}

func (LoRaWANHandler) Decode(payload []byte) (lorawan.Packet, error) {
	if len(payload) > 0 && payload[0]&0x1f != 0 {
		return lorawan.Packet{}, fmt.Errorf("unsupported LoRaWAN header version or reserved bits")
	}
	packet, err := lorawan.Parse(payload)
	if err != nil {
		return lorawan.Packet{}, err
	}
	switch packet.MType {
	case lorawan.MTypeJoinAccept:
		if len(payload) != 17 && len(payload) != 33 {
			return lorawan.Packet{}, fmt.Errorf("LoRaWAN Join-Accept must contain 17 or 33 bytes")
		}
		return packet, nil
	case lorawan.MTypeJoinRequest,
		lorawan.MTypeUnconfirmedDataUp,
		lorawan.MTypeUnconfirmedDataDown,
		lorawan.MTypeConfirmedDataUp,
		lorawan.MTypeConfirmedDataDown:
		return packet, nil
	default:
		return lorawan.Packet{}, fmt.Errorf("unsupported LoRaWAN MType %d", packet.MType)
	}
}

func (LoRaWANHandler) DecryptPayload(packet lorawan.Packet, key []byte, direction byte) ([]byte, error) {
	return lorawan.DecryptPayload(packet, key, direction)
}

func (LoRaWANHandler) VerifyDataMIC(packet lorawan.Packet, key []byte, direction byte) bool {
	return lorawan.VerifyDataMIC(packet, key, direction)
}

package packets

import (
	"encoding/json"

	"github.com/brocaar/lorawan"
)

const (
	NONE             = "NONE"
	TOO_LATE         = "TOO_LATE"
	TOO_EARLY        = "TOO_EARLY"
	COLLISION_PACKET = "COLLISION_PACKET"
	COLLISION_BEACON = "COLLISION_BEACON"
	TX_FREQ          = "TX_FREQ"
	TX_POWER         = "TX_POWER"
	GPS_UNLOCKED     = "GPS_UNLOCKED"
)

type TxAckPacket struct {
	Header  []byte
	Payload TXACKPayload
}

type TXACKPayload struct {
	TXPKACK TXPKACK `json:"txpk_ack"`
}

type TXPKACK struct {
	Error string `json:"error"`
}

func SetTXACKPayload() TXACKPayload {

	var payload TXACKPayload
	payload.TXPKACK.Error = NONE

	return payload
}

func CreateTXPacket(GatewayMACAddr lorawan.EUI64, Token uint16) ([]byte, error) {

	header := GetHeader(TypeTxAck, GatewayMACAddr, Token)
	payload := SetTXACKPayload()
	packet := TxAckPacket{
		header,
		payload,
	}

	return packet.MarshalBinary()

}

func (p *TxAckPacket) MarshalBinary() ([]byte, error) {

	payloadBytes, err := json.Marshal(p.Payload)
	if err != nil {
		return nil, err
	}

	out := append(p.Header, payloadBytes...)

	return out, nil
}

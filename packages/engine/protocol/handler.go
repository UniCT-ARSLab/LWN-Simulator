package protocol

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

// Handler represents the protocol operations needed at the radio boundary.
// The radio medium does not depend on the concrete LoRaWAN implementation;
// protocol-specific adapters provide encoding and decoding instead.
type Handler interface {
	Protocol() contracts.RadioProtocol
	EncodeJoinRequest(JoinRequest) ([]byte, error)
	EncodeJoinAccept(JoinAccept) ([]byte, error)
	EncodeDataFrame(DataFrame) ([]byte, error)
	EncodeMACCommands([]types.MACCommand) ([]byte, error)
	Decode([]byte) (lorawan.Packet, error)
	DecryptPayload(lorawan.Packet, []byte, byte) ([]byte, error)
	VerifyDataMIC(lorawan.Packet, []byte, byte) bool
}

// JoinRequest contains the protocol-independent inputs required to create a
// join request at the protocol adapter boundary.
type JoinRequest struct {
	JoinEUI  uint64
	DevEUI   uint64
	DevNonce uint16
	AppKey   []byte
}

// JoinAccept contains the session parameters required to create a join accept
// at the protocol adapter boundary.
type JoinAccept struct {
	AppNonce          uint32
	NetID             uint32
	DevAddr           uint32
	RX2DataRate       int
	RX1DataRateOffset int
	RXDelay           int
	AppKey            []byte
}

// DataFrame contains the session and application values required to create a
// LoRaWAN data frame. The adapter owns the wire-format details and MIC logic.
type DataFrame struct {
	DevAddr   uint32
	FCnt      uint32
	FPort     *byte
	Payload   []byte
	Confirmed bool
	ACK       bool
	FPending  bool
	ADR       bool
	Direction byte
	NwkSKey   []byte
	AppSKey   []byte
	FOpts     []byte
}

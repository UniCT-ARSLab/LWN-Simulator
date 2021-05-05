package packets

import (
	"encoding/binary"
	"math/rand"

	"github.com/brocaar/lorawan"
)

const (
	SizeHeader = 12
)

type Header struct {
	ProtocolVersion uint8
	RandomToken     uint16
	IDPacket        uint8
	GatewayMACAddr  lorawan.EUI64
}

func GetHeader(IDPacket uint8, GatewayMACAddr lorawan.EUI64, token uint16) []byte {

	randomNumber := rand.Int()
	randomToken := token
	if token == 0 {
		randomToken = uint16(randomNumber)
	}

	header := Header{
		ProtocolVersion: PVersion,
		RandomToken:     randomToken,
		IDPacket:        IDPacket,
		GatewayMACAddr:  GatewayMACAddr,
	}

	return header.MarshalBinary()
}

func (h *Header) MarshalBinary() []byte {

	out := make([]byte, 4, SizeHeader)

	out[0] = h.ProtocolVersion
	binary.LittleEndian.PutUint16(out[1:3], h.RandomToken)
	out[3] = byte(h.IDPacket)
	out = append(out, h.GatewayMACAddr[0:len(h.GatewayMACAddr)]...)

	return out
}

package packets

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/brocaar/lorawan"
)

const (
	MinLenPushData = 4
	MinLenPushAck  = 4
	MinLenPullAck  = 4
	MinLenPullData = 12
	MinLenPullResp = 5
	MinLenTxAck    = 12

	PVersion = 0x02

	TypePushData     = 0x00
	TypePushAck      = 0x01
	TypePullData     = 0x02
	TypePullResp     = 0x03
	TypePullAck      = 0x04
	TypeTxAck        = 0x05
	TypeNotSupported = 0x08

	StringPushData = "PUSH DATA"
	StringPullData = "PULL DATA"
	StringTxAck    = "TX ACK"
	StringPushAck  = "PUSH ACK"
	StringPullAck  = "PULL ACK"
	StringPullResp = "PULL RESP"
)

type Packet []byte

func (packet Packet) IsSupportedProtocol() error {

	if packet[0] != byte(PVersion) {
		return errors.New("Protocol not supported")
	}
	return nil
}

func (packet Packet) IsSupportedType() error {

	str := ""

	switch packet[3] {

	case TypePushAck:
		if len(packet) < MinLenPushAck {
			str = fmt.Sprintf("Min PushData packet's legth is %d", MinLenPushAck)
			break
		}

	case TypePullAck:
		if len(packet) < MinLenPullAck {
			str = fmt.Sprintf("Min PushData packet's legth is %d", MinLenPullAck)
			break
		}

	case TypePullResp:
		if len(packet) < MinLenPullData {
			str = fmt.Sprintf("Min PushData packet's legth is %d", MinLenPullData)
			break
		}

	default:
		str = "Type packet not supported"
	}

	if str == "" {
		return nil
	}
	return errors.New(str)
}

func GetTokenFromPullResp(pkt []byte) uint16 {

	if pkt[3] == TypePullResp {

		numBytes := []byte{pkt[1], pkt[2]}
		token := binary.LittleEndian.Uint16(numBytes)

		return token
	}

	return 0
}

func PacketToString(IDPacket uint8) string {

	switch IDPacket {

	case TypePushData:
		return StringPushData
	case TypePushAck:
		return StringPushAck
	case TypePullData:
		return StringPullData
	case TypePullAck:
		return StringPullAck
	case TypePullResp:
		return StringPullResp
	case TypeTxAck:
		return StringTxAck

	}

	return "None Type"
}

func GetTypePacket(packet []byte) *byte {

	switch packet[3] {

	case TypePushData:
	case TypePushAck:
	case TypePullData:
	case TypePullAck:
	case TypePullResp:
	case TypeTxAck:
		break

	default:
		NotSupported := byte(TypeNotSupported)
		return &NotSupported

	}

	return &packet[3]
}

func ParseReceivePacket(p Packet) error {

	err := p.IsSupportedProtocol()
	if err != nil {
		return err
	}

	err = p.IsSupportedType()
	if err != nil {
		return err
	}

	return nil
}

func CreatePacket(id int, GatewayMACAddr lorawan.EUI64, stat Stat, info []RXPK, token uint16) ([]byte, error) {

	switch id {

	case TypePushData:
		return CreatePushDataPacket(GatewayMACAddr, stat, info)
	case TypePullData:
		return CreatePullDataPacket(GatewayMACAddr), nil
	case TypeTxAck:
		return CreateTXPacket(GatewayMACAddr, token)
	default:
		return []byte{}, nil

	}

}

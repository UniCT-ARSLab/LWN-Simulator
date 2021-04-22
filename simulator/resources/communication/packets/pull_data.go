package packets

import "github.com/brocaar/lorawan"

func CreatePullDataPacket(GatewayMACAddr lorawan.EUI64) []byte {

	packetBytes := GetHeader(TypePullData, GatewayMACAddr, 0)

	return packetBytes
}

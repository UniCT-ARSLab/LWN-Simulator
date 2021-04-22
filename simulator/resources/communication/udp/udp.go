package udp

import (
	"net"
)

func ConnectTo(BridgeAddress string) (*net.UDPConn, error) {

	var err error
	var addressRS *net.UDPAddr

	addressRS, err = net.ResolveUDPAddr("udp", BridgeAddress)
	connection, err := net.DialUDP("udp", nil, addressRS) //udp4

	if err != nil {
		return nil, err
	}

	return connection, nil
}

func SendDataUDP(connection *net.UDPConn, data []byte) (int, error) {

	n, err := connection.Write(data)
	if err != nil {
		return -1, err
	}

	return n, err
}

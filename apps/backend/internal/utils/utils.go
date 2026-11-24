package utils

import (
	"net"
)

func GetPrivateIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP

		if ip.IsLoopback() {
			continue
		}

		if ip.To4() != nil {
			return ip.String()
		}
	}

	return "unknown"
}

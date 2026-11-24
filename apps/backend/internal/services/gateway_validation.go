package services

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func ValidateGatewayRequest(req contracts.CreateGatewayRequest) error {
	return validateGatewayFields(
		req.Type,
		req.Name,
		req.GatewayEUI,
		req.KeepAlive,
		req.GatewayIPv4,
		req.GatewayPort,
		req.Latitude,
		req.Longitude,
		req.Altitude,
	)
}

func ValidateGateway(gateway contracts.Gateway) error {
	return validateGatewayFields(
		gateway.Type,
		gateway.Name,
		gateway.GatewayEUI,
		gateway.KeepAlive,
		gateway.GatewayIPv4,
		gateway.GatewayPort,
		gateway.Latitude,
		gateway.Longitude,
		gateway.Altitude,
	)
}

func validateGatewayFields(
	gatewayType contracts.GatewayType,
	name string,
	gatewayEUI string,
	keepAlive *int32,
	gatewayIPv4 *string,
	gatewayPort *int32,
	latitude *float32,
	longitude *float32,
	altitude *float32,
) error {
	if !gatewayType.Valid() {
		return fmt.Errorf("type must be either %q or %q", contracts.Virtual, contracts.Real)
	}

	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name is required")
	}

	if len(gatewayEUI) != 16 {
		return fmt.Errorf("gatewayEUI must contain 16 hexadecimal characters")
	}

	if _, err := hex.DecodeString(gatewayEUI); err != nil {
		return fmt.Errorf("gatewayEUI must contain 16 hexadecimal characters")
	}

	if latitude == nil || *latitude < -90 || *latitude > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}

	if longitude == nil || *longitude < -180 || *longitude > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}

	if altitude == nil {
		return fmt.Errorf("altitude is required")
	}

	switch gatewayType {
	case contracts.Virtual:
		if keepAlive == nil || *keepAlive <= 0 {
			return fmt.Errorf("keepAlive must be greater than zero seconds for virtual gateways")
		}
		if gatewayIPv4 != nil || gatewayPort != nil {
			return fmt.Errorf("gatewayIPv4 and gatewayPort are only valid for real gateways")
		}
	case contracts.Real:
		if keepAlive != nil {
			return fmt.Errorf("keepAlive is only valid for virtual gateways")
		}
		if gatewayIPv4 == nil || net.ParseIP(*gatewayIPv4) == nil || net.ParseIP(*gatewayIPv4).To4() == nil {
			return fmt.Errorf("gatewayIPv4 must be a valid IPv4 address for real gateways")
		}
		if gatewayPort == nil || *gatewayPort < 1 || *gatewayPort > 65535 {
			return fmt.Errorf("gatewayPort must be between 1 and 65535 for real gateways")
		}
	}

	return nil
}

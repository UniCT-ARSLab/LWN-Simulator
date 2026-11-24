package services

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"lwn-simulator-backend/internal/apperrors"
)

func (s *DeviceService) UpdatePosition(id string, req contracts.UpdatePositionRequest) (contracts.UpdateDeviceResponse, error) {
	return s.patchDevice(id, func(device *contracts.Device) {
		device.LocationConfig.Latitude = req.Latitude
		device.LocationConfig.Longitude = req.Longitude
		device.LocationConfig.Altitude = req.Altitude
	})
}

func (s *DeviceService) UpdateActive(id string, active bool) (contracts.UpdateDeviceResponse, error) {
	return s.patchDevice(id, func(device *contracts.Device) {
		device.Active = active
	})
}

func (s *DeviceService) UpdatePayload(id string, payload string, base64Encoded *bool) (contracts.UpdateDeviceResponse, error) {
	return s.patchDevice(id, func(device *contracts.Device) {
		device.PayloadConfig.Payload = payload
		if base64Encoded != nil {
			device.PayloadConfig.Base64Encoded = *base64Encoded
		}
	})
}

func (s *DeviceService) patchDevice(id string, apply func(*contracts.Device)) (contracts.UpdateDeviceResponse, error) {
	return s.updateDevice(id, func(device *contracts.Device) error {
		apply(device)
		if err := ValidateDevice(*device); err != nil {
			return apperrors.Invalid("%v", err)
		}
		return nil
	})
}

func (s *GatewayService) UpdatePosition(id string, req contracts.UpdatePositionRequest) (contracts.UpdateGatewayResponse, error) {
	return s.patchGateway(id, func(gateway *contracts.Gateway) {
		gateway.Latitude = req.Latitude
		gateway.Longitude = req.Longitude
		gateway.Altitude = req.Altitude
	})
}

func (s *GatewayService) UpdateActive(id string, active bool) (contracts.UpdateGatewayResponse, error) {
	return s.patchGateway(id, func(gateway *contracts.Gateway) {
		gateway.Active = active
	})
}

func (s *GatewayService) patchGateway(id string, apply func(*contracts.Gateway)) (contracts.UpdateGatewayResponse, error) {
	return s.updateGateway(id, func(gateway *contracts.Gateway) error {
		apply(gateway)
		if err := ValidateGateway(*gateway); err != nil {
			return apperrors.Invalid("%v", err)
		}
		return nil
	})
}

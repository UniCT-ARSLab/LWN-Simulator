package engine

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	validationruntime "github.com/Giuseppe-Compagnone/lwn-engine/validation"
)

func ValidateSimulationConfig(config contracts.SimulationConfig) error {
	return validationruntime.ValidateSimulationConfig(config)
}

func ValidateHardware(devices []contracts.Device, gateways []contracts.Gateway) error {
	return validationruntime.ValidateHardware(devices, gateways)
}

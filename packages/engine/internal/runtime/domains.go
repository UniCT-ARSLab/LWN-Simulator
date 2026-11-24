package runtime

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	registryruntime "github.com/Giuseppe-Compagnone/lwn-engine/registry"
	schedulerruntime "github.com/Giuseppe-Compagnone/lwn-engine/scheduler"
	validationruntime "github.com/Giuseppe-Compagnone/lwn-engine/validation"
)

// Registry and Scheduler are kept behind the runtime package so that the
// orchestration layer depends on domain packages without importing the public
// engine facade.
type Registry = registryruntime.Registry
type Scheduler = schedulerruntime.Scheduler

func NewRegistry(devices []contracts.Device, gateways []contracts.Gateway) (*Registry, error) {
	return registryruntime.New(devices, gateways)
}

func NewScheduler() *Scheduler {
	return schedulerruntime.New()
}

func ValidateSimulationConfig(config contracts.SimulationConfig) error {
	return validationruntime.ValidateSimulationConfig(config)
}

func ValidateHardware(devices []contracts.Device, gateways []contracts.Gateway) error {
	return validationruntime.ValidateHardware(devices, gateways)
}

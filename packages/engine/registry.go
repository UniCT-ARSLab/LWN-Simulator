package engine

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	registryruntime "github.com/Giuseppe-Compagnone/lwn-engine/registry"
)

type Registry = registryruntime.Registry

func NewRegistry(devices []contracts.Device, gateways []contracts.Gateway) (*Registry, error) {
	return registryruntime.New(devices, gateways)
}

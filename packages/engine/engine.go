// Package engine exposes the public simulation-engine API.
//
// The implementation lives in internal/runtime. Keeping this small facade
// preserves the existing import path while preventing the module root from
// becoming a dumping ground for runtime implementation details.
package engine

import (
	"context"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	runtime "github.com/Giuseppe-Compagnone/lwn-engine/internal/runtime"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

type Engine = runtime.Engine

func New(
	config contracts.SimulationConfig,
	devices []contracts.Device,
	gateways []contracts.Gateway,
	options types.Options,
) (*Engine, error) {
	return runtime.New(config, devices, gateways, options)
}

// Compile-time assertion documenting the lifecycle API exposed by Engine.
var _ interface {
	Start(context.Context) error
	SetSpeed(float64) error
} = (*Engine)(nil)

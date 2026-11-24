package engine

import runtime "github.com/Giuseppe-Compagnone/lwn-engine/internal/runtime"

var (
	ErrInvalidTransition           = runtime.ErrInvalidTransition
	ErrEngineStopped               = runtime.ErrEngineStopped
	ErrEngineNotRunning            = runtime.ErrEngineNotRunning
	ErrEventNotFound               = runtime.ErrEventNotFound
	ErrDeviceNotFound              = runtime.ErrDeviceNotFound
	ErrGatewayNotFound             = runtime.ErrGatewayNotFound
	ErrGatewayTransportUnavailable = runtime.ErrGatewayTransportUnavailable
)

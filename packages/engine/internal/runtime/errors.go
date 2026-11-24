package runtime

import "errors"

var (
	ErrInvalidTransition           = errors.New("invalid simulation lifecycle transition")
	ErrEngineStopped               = errors.New("simulation engine is stopped")
	ErrEngineNotRunning            = errors.New("simulation engine is not running")
	ErrEventNotFound               = errors.New("scheduled event not found")
	ErrDeviceNotFound              = errors.New("device is not registered")
	ErrGatewayNotFound             = errors.New("gateway is not registered")
	ErrGatewayTransportUnavailable = errors.New("gateway transport is unavailable")
)

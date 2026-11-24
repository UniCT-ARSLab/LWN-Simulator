package engine

import (
	"context"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestPublicFacadeConstructsCoreServices(t *testing.T) {
	if err := ValidateSimulationConfig(contracts.SimulationConfig{Speed: 1}); err != nil {
		t.Fatalf("public config validation failed: %v", err)
	}
	if err := ValidateHardware(nil, nil); err != nil {
		t.Fatalf("public hardware validation failed: %v", err)
	}
	registry, err := NewRegistry(nil, nil)
	if err != nil || registry == nil {
		t.Fatalf("public registry construction failed: %v", err)
	}
	if NewScheduler() == nil {
		t.Fatal("public scheduler construction returned nil")
	}
	engine, err := New(contracts.SimulationConfig{Speed: 1}, nil, nil, types.Options{Clock: types.NewManualClock(0)})
	if err != nil || engine == nil {
		t.Fatalf("public engine construction failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := engine.Start(ctx); err != nil {
		t.Fatalf("public engine start failed: %v", err)
	}
	if err := engine.Stop(ctx); err != nil {
		t.Fatalf("public engine stop failed: %v", err)
	}
}

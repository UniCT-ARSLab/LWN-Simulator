package runtime

import (
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestTelemetrySamplesUseSimulationTimeAndIncludeTerminalMetrics(t *testing.T) {
	clock := types.NewManualClock(0)
	e := &Engine{clock: clock}
	initial := e.newEventLocked(contracts.SimulationStarted, "start", "", "", "")
	if initial.Telemetry == nil || initial.Telemetry.Sequence != initial.Sequence {
		t.Fatalf("initial sample missing: %+v", initial)
	}
	if err := clock.Advance(9 * time.Second); err != nil {
		t.Fatal(err)
	}
	if event := e.newEventLocked(contracts.MetricsUpdated, "update", "", "", ""); event.Telemetry != nil {
		t.Fatal("sample emitted before the sampling interval")
	}
	if err := clock.Advance(time.Second); err != nil {
		t.Fatal(err)
	}
	e.metrics.TotalUplinks = 1
	event := e.newUplinkOutcomeEventLocked(contracts.DeviceUplinkTransmitted, "uplink", "device", "", "", "packet", contracts.UplinkStarted)
	if event.Telemetry == nil || event.Telemetry.ElapsedMilliseconds != 10_000 ||
		event.Telemetry.Metrics.TotalUplinks != 1 || event.UplinkOutcome == nil || *event.UplinkOutcome != contracts.UplinkStarted {
		t.Fatalf("periodic sample or logical uplink marker missing: %+v", event)
	}

	e.metrics.SuccessfulUplinks = 1
	stopped := e.newEventLocked(contracts.SimulationStopped, "stop", "", "", "")
	if stopped.Telemetry == nil || stopped.Telemetry.Metrics.SuccessfulUplinks != 1 || event.Telemetry.Metrics.SuccessfulUplinks != 0 {
		t.Fatalf("terminal sample omitted or mutated an earlier sample: %+v", stopped.Telemetry)
	}
}

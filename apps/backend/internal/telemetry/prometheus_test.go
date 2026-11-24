package telemetry

import (
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestPrometheusRendersSnapshotMetrics(t *testing.T) {
	snapshot := contracts.SimulationSnapshot{
		State:   contracts.SimulationState{Status: contracts.SimulationStatusRunning, Speed: 2, ElapsedMilliseconds: 1250},
		Metrics: contracts.SimulationMetrics{ActiveDevices: 3, TotalUplinks: 8, PacketSuccessRate: 0.75},
	}
	output := string(Prometheus(snapshot))
	for _, expected := range []string{
		"lwn_simulation_running 1",
		"lwn_simulation_elapsed_seconds 1.25",
		"lwn_simulation_speed 2",
		"lwn_simulation_active_devices 3",
		"lwn_simulation_total_uplinks 8",
		"lwn_simulation_packet_success_ratio 0.75",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("Prometheus() output does not contain %q\n%s", expected, output)
		}
	}
}

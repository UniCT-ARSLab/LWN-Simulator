package telemetry

import (
	"fmt"
	"strings"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

// Prometheus renders the current immutable engine snapshot using the
// Prometheus text exposition format. No collector state is kept in the
// backend, so restarting the HTTP layer cannot duplicate counters.
func Prometheus(snapshot contracts.SimulationSnapshot) []byte {
	var output strings.Builder
	metric := func(name, help, metricType string, value any) {
		fmt.Fprintf(&output, "# HELP %s %s\n# TYPE %s %s\n%s %v\n", name, help, name, metricType, name, value)
	}
	running := 0
	if snapshot.State.Status == contracts.SimulationStatusRunning {
		running = 1
	}
	metric("lwn_simulation_running", "Whether the simulation is currently running.", "gauge", running)
	metric("lwn_simulation_elapsed_seconds", "Elapsed simulation time in seconds.", "gauge", float64(snapshot.State.ElapsedMilliseconds)/1000)
	metric("lwn_simulation_speed", "Configured simulation clock multiplier.", "gauge", snapshot.State.Speed)
	metric("lwn_simulation_active_devices", "Number of active runtime devices.", "gauge", snapshot.Metrics.ActiveDevices)
	metric("lwn_simulation_active_gateways", "Number of active runtime gateways.", "gauge", snapshot.Metrics.ActiveGateways)
	metric("lwn_simulation_total_uplinks", "Total logical uplinks.", "counter", snapshot.Metrics.TotalUplinks)
	metric("lwn_simulation_successful_uplinks", "Successfully delivered logical uplinks.", "counter", snapshot.Metrics.SuccessfulUplinks)
	metric("lwn_simulation_dropped_uplinks", "Dropped logical uplinks.", "counter", snapshot.Metrics.DroppedUplinks)
	metric("lwn_simulation_total_transmissions", "Total radio transmissions.", "counter", snapshot.Metrics.TotalTransmissions)
	metric("lwn_simulation_packet_success_ratio", "Current packet success ratio from zero to one.", "gauge", snapshot.Metrics.PacketSuccessRate)
	metric("lwn_simulation_collisions", "Radio collisions detected.", "counter", snapshot.Metrics.Collisions)
	metric("lwn_simulation_radio_packet_losses", "Packets lost by the radio model.", "counter", snapshot.Metrics.RadioPacketLosses)
	metric("lwn_simulation_average_rssi_dbm", "Average received signal strength in dBm.", "gauge", snapshot.Metrics.AverageRSSI)
	metric("lwn_simulation_average_snr_db", "Average signal-to-noise ratio in dB.", "gauge", snapshot.Metrics.AverageSNR)
	metric("lwn_simulation_join_requests", "OTAA join requests.", "counter", snapshot.Metrics.JoinRequests)
	metric("lwn_simulation_join_accepts", "OTAA join accepts.", "counter", snapshot.Metrics.JoinAccepts)
	metric("lwn_simulation_retransmissions", "Protocol retransmissions.", "counter", snapshot.Metrics.Retransmissions)
	metric("lwn_simulation_gateway_network_errors", "Real gateway network errors.", "counter", snapshot.Metrics.GatewayNetworkErrors)
	metric("lwn_simulation_class_b_downlinks", "Class B downlinks transmitted.", "counter", snapshot.Metrics.ClassBDownlinks)
	return []byte(output.String())
}

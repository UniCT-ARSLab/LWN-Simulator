package services

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type simulationEventVisitor interface {
	visitRunEvents(context.Context, string, func(contracts.SimulationEvent)) (contracts.SimulationRun, error)
}

// ExportTelemetry writes a summary CSV for the profile or a ZIP for one run.
func (s *SimulationService) ExportTelemetry(ctx context.Context, id string, destination io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.flushPendingPersistence(ctx, false); err != nil {
		return err
	}
	if id == "" {
		runs, err := s.ListLogs()
		if err != nil {
			return err
		}
		return writeTelemetrySummary(ctx, destination, runs.Runs)
	}
	if s.logStore == nil {
		return os.ErrNotExist
	}

	report := newSimulationTelemetryReport()
	var run contracts.SimulationRun
	var err error
	if reader, ok := s.logStore.(simulationEventVisitor); ok {
		run, err = reader.visitRunEvents(ctx, id, report.observe)
	} else {
		var events []contracts.SimulationEvent
		run, events, err = s.GetLog(id)
		if err == nil {
			for _, event := range events {
				if err := ctx.Err(); err != nil {
					return err
				}
				report.observe(event)
			}
		}
	}
	if err != nil {
		return err
	}
	run.Summary.Telemetry = latestTelemetry(run.Summary.Telemetry, report.latest)
	if report.lastSequence > 0 {
		run.Summary.EventCount = int32(report.lastSequence)
	}
	return report.writeArchive(ctx, destination, run)
}

func (report *simulationTelemetryReport) writeArchive(ctx context.Context, destination io.Writer, run contracts.SimulationRun) (err error) {
	archive := zip.NewWriter(destination)
	defer func() { err = errors.Join(err, archive.Close()) }()
	files := []struct {
		name  string
		write func(io.Writer) error
	}{
		{"summary.csv", func(output io.Writer) error {
			return writeTelemetrySummary(ctx, output, []contracts.SimulationRun{run})
		}},
		{"timeline.csv", func(output io.Writer) error { return report.writeTimeline(ctx, output) }},
		{"radio.csv", func(output io.Writer) error { return report.writeRadio(ctx, output) }},
		{"devices.csv", func(output io.Writer) error { return report.writeDevices(ctx, output, run.Objects) }},
		{"gateways.csv", func(output io.Writer) error { return report.writeGateways(ctx, output, run.Objects) }},
		{"README.txt", func(output io.Writer) error { _, err := io.WriteString(output, telemetryReadme); return err }},
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		entry, err := archive.Create(file.name)
		if err != nil {
			return err
		}
		if err := file.write(entry); err != nil {
			return err
		}
	}
	return nil
}

func writeTelemetrySummary(ctx context.Context, output io.Writer, runs []contracts.SimulationRun) error {
	header := []string{
		"run_id", "status", "seed", "started_at_utc", "ended_at_utc", "simulated_duration_ms", "speed",
		"devices", "gateways", "events_through_sequence", "metrics_source", "metrics_sample_at_ms", "metrics_through_sequence",
		"packet_success_rate_pct", "uplinks_started", "uplinks_succeeded", "uplinks_dropped", "uplinks_without_outcome",
		"retransmissions", "acks", "radio_transmissions", "radio_receptions", "radio_losses", "collisions", "low_snr_losses",
		"join_requests", "join_accepts", "gateway_ingress", "gateway_egress", "gateway_reconnects", "gateway_timeouts", "gateway_errors",
	}
	return writeTelemetryCSV(ctx, output, header, func(write func([]string) error) error {
		for _, run := range runs {
			summary := run.Summary
			row := make([]string, len(header))
			copy(row, []string{
				telemetryText(summary.Id), string(summary.Status), telemetryOptionalInt(summary.Seed),
				time.UnixMilli(summary.StartedAtMilliseconds).UTC().Format(time.RFC3339Nano), "",
				telemetryInt(summary.DurationMilliseconds), telemetryFloat(summary.Speed),
				telemetryInt(int64(summary.DeviceCount)), telemetryInt(int64(summary.GatewayCount)),
				telemetryInt(int64(summary.EventCount)), "unavailable",
			})
			if summary.EndedAtMilliseconds != nil {
				row[4] = time.UnixMilli(*summary.EndedAtMilliseconds).UTC().Format(time.RFC3339Nano)
			}
			if sample := summary.Telemetry; sample != nil {
				metrics := sample.Metrics
				copy(row[10:], []string{
					"sampled", telemetryInt(sample.ElapsedMilliseconds), telemetryInt(sample.Sequence),
					telemetryRate(metrics.SuccessfulUplinks, metrics.TotalUplinks),
					telemetryInt(metrics.TotalUplinks), telemetryInt(metrics.SuccessfulUplinks), telemetryInt(metrics.DroppedUplinks),
					telemetryInt(max(0, metrics.TotalUplinks-metrics.SuccessfulUplinks-metrics.DroppedUplinks)),
					telemetryInt(metrics.Retransmissions), telemetryInt(metrics.AcknowledgedUplinks), telemetryInt(metrics.TotalTransmissions),
					telemetryInt(metrics.TotalPacketsReceived), telemetryInt(metrics.RadioPacketLosses), telemetryInt(metrics.Collisions),
					telemetryInt(metrics.LowSNRLosses), telemetryInt(metrics.JoinRequests), telemetryInt(metrics.JoinAccepts),
					telemetryInt(metrics.GatewayIngressPackets), telemetryInt(metrics.GatewayEgressPackets),
					telemetryInt(metrics.GatewayReconnects), telemetryInt(metrics.GatewayTimeouts), telemetryInt(metrics.GatewayNetworkErrors),
				})
			} else if summary.PacketSuccessRate > 0 && summary.PacketSuccessRate <= 1 {
				row[10] = "legacy-summary"
				row[13] = telemetryFloat(summary.PacketSuccessRate * 100)
			}
			if err := write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

func (report *simulationTelemetryReport) writeTimeline(ctx context.Context, output io.Writer) error {
	header := []string{
		"interval_start_ms", "interval_end_exclusive_ms", "uplinks_started", "uplinks_succeeded", "uplinks_dropped",
		"retransmissions_observed", "acks_observed", "radio_losses_observed", "collisions_observed",
		"cumulative_uplinks_started", "cumulative_uplinks_succeeded", "packet_success_rate_pct",
		"metrics_sample_at_ms", "active_devices_sampled", "connected_gateways_sampled",
	}
	starts := make([]int64, 0, len(report.intervals))
	for start := range report.intervals {
		starts = append(starts, start)
	}
	slices.Sort(starts)
	return writeTelemetryCSV(ctx, output, header, func(write func([]string) error) error {
		var totalStarted, totalSucceeded int64
		for _, start := range starts {
			interval := report.intervals[start]
			traffic := interval.traffic
			row := make([]string, len(header))
			copy(row, []string{telemetryInt(start), telemetryInt(start + telemetryIntervalMilliseconds)})
			copy(row[5:], []string{
				telemetryInt(traffic.retransmissions), telemetryInt(traffic.acknowledgements),
				telemetryInt(interval.losses), telemetryInt(interval.collisions),
			})
			if report.completeUplinkLog {
				copy(row[2:], []string{telemetryInt(traffic.started), telemetryInt(traffic.succeeded), telemetryInt(traffic.dropped)})
				totalStarted += traffic.started
				totalSucceeded += traffic.succeeded
				copy(row[9:], []string{telemetryInt(totalStarted), telemetryInt(totalSucceeded), telemetryRate(totalSucceeded, totalStarted)})
			}
			if sample := interval.sample; sample != nil {
				copy(row[12:], []string{telemetryInt(sample.ElapsedMilliseconds), telemetryInt(sample.Metrics.ActiveDevices), telemetryInt(sample.ConnectedGateways)})
				if !report.completeUplinkLog {
					metrics := sample.Metrics
					copy(row[9:], []string{
						telemetryInt(metrics.TotalUplinks), telemetryInt(metrics.SuccessfulUplinks),
						telemetryRate(metrics.SuccessfulUplinks, metrics.TotalUplinks),
					})
				}
			}
			if err := write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

func (report *simulationTelemetryReport) writeRadio(ctx context.Context, output io.Writer) error {
	header := []string{
		"protocol", "frequency_hz", "spreading_factor", "transmissions_completed", "gateway_receptions", "transport_frames",
		"collision_losses", "low_snr_losses", "no_gateway_losses", "random_losses", "other_losses",
		"airtime_samples", "airtime_p50_ms", "airtime_p95_ms",
		"rssi_samples", "rssi_p50_dbm", "rssi_p95_dbm", "snr_samples", "snr_p50_db", "snr_p95_db",
	}
	keys := make([]telemetryRadioKey, 0, len(report.radio))
	for key := range report.radio {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, compareTelemetryRadioKeys)
	return writeTelemetryCSV(ctx, output, header, func(write func([]string) error) error {
		for _, key := range keys {
			radio := report.radio[key]
			row := []string{
				telemetryText(key.protocol), telemetryPositiveInt(key.frequency), telemetryPositiveInt(int64(key.sf)),
				telemetryInt(radio.transmissions), telemetryInt(radio.receptions), telemetryInt(radio.transportFrames),
				telemetryInt(radio.collisions), telemetryInt(radio.lowSNR), telemetryInt(radio.noGateway),
				telemetryInt(radio.randomLoss), telemetryInt(radio.otherLoss),
			}
			for _, values := range []telemetryHistogram{radio.airtime, radio.rssi, radio.snr} {
				if values.count == 0 {
					row = append(row, "0", "", "")
				} else {
					p50, p95 := values.percentiles()
					row = append(row, telemetryInt(values.count), telemetryFloat(p50), telemetryFloat(p95))
				}
			}
			if err := write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

func (report *simulationTelemetryReport) writeDevices(ctx context.Context, output io.Writer, objects []contracts.SimulationLogObject) error {
	header := []string{"device_id", "name", "dev_eui", "uplinks_started", "uplinks_succeeded", "uplinks_dropped",
		"uplinks_without_outcome", "packet_success_rate_pct", "retransmissions_observed", "acks_observed", "join_requests_observed", "join_accepts_observed"}
	return writeTelemetryCSV(ctx, output, header, func(write func([]string) error) error {
		for _, object := range telemetryObjects(objects, contracts.SimulationLogObjectKindDevice, report.devices) {
			device := report.devices[object.Id]
			if device == nil {
				device = &telemetryDevice{}
			}
			traffic := device.traffic
			row := make([]string, len(header))
			copy(row, []string{telemetryText(object.Id), telemetryText(object.Name), telemetryText(object.Identifier)})
			if report.completeUplinkLog {
				copy(row[3:], []string{
					telemetryInt(traffic.started), telemetryInt(traffic.succeeded), telemetryInt(traffic.dropped),
					telemetryInt(max(0, traffic.started-traffic.succeeded-traffic.dropped)), telemetryRate(traffic.succeeded, traffic.started),
				})
			}
			copy(row[8:], []string{
				telemetryInt(traffic.retransmissions), telemetryInt(traffic.acknowledgements),
				telemetryInt(device.joinRequests), telemetryInt(device.joinAccepts),
			})
			if err := write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

func (report *simulationTelemetryReport) writeGateways(ctx context.Context, output io.Writer, objects []contracts.SimulationLogObject) error {
	header := []string{"gateway_id", "name", "gateway_eui", "gateway_type", "radio_receptions_observed", "transport_ingress_observed",
		"transport_egress_observed", "disconnections_observed", "reconnections_observed", "network_errors_observed"}
	return writeTelemetryCSV(ctx, output, header, func(write func([]string) error) error {
		for _, object := range telemetryObjects(objects, contracts.SimulationLogObjectKindGateway, report.gateways) {
			gateway := report.gateways[object.Id]
			if gateway == nil {
				gateway = &telemetryGateway{}
			}
			row := []string{
				telemetryText(object.Id), telemetryText(object.Name), telemetryText(object.Identifier), "",
				telemetryInt(gateway.received), telemetryInt(gateway.ingress), telemetryInt(gateway.egress),
				telemetryInt(gateway.disconnected), telemetryInt(gateway.reconnects), telemetryInt(gateway.errors),
			}
			if object.GatewayType != nil {
				row[3] = string(*object.GatewayType)
			}
			if err := write(row); err != nil {
				return err
			}
		}
		return nil
	})
}

func telemetryObjects[V any](objects []contracts.SimulationLogObject, kind contracts.SimulationLogObjectKind, observed map[string]*V) []contracts.SimulationLogObject {
	result := make([]contracts.SimulationLogObject, 0)
	known := make(map[string]bool)
	for _, object := range objects {
		if object.Kind == kind {
			result = append(result, object)
			known[object.Id] = true
		}
	}
	for id := range observed {
		if !known[id] {
			result = append(result, contracts.SimulationLogObject{Id: id, Kind: kind})
		}
	}
	slices.SortFunc(result, func(a, b contracts.SimulationLogObject) int { return strings.Compare(a.Id, b.Id) })
	return result
}

func writeTelemetryCSV(ctx context.Context, output io.Writer, header []string, rows func(func([]string) error) error) error {
	writer := csv.NewWriter(output)
	write := func(row []string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return writer.Write(row)
	}
	if err := write(header); err != nil {
		return err
	}
	if err := rows(write); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

func telemetryInt(value int64) string     { return strconv.FormatInt(value, 10) }
func telemetryFloat(value float64) string { return strconv.FormatFloat(value, 'f', -1, 64) }

func telemetryOptionalInt(value *int64) string {
	if value == nil {
		return ""
	}
	return telemetryInt(*value)
}

func telemetryPositiveInt(value int64) string {
	if value <= 0 {
		return ""
	}
	return telemetryInt(value)
}

func telemetryRate(succeeded, total int64) string {
	if total == 0 {
		return ""
	}
	return telemetryFloat(100 * float64(succeeded) / float64(total))
}

func telemetryText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	first, _ := utf8.DecodeRuneInString(trimmed)
	if strings.ContainsRune("=+-@＝＋－＠", first) ||
		strings.HasPrefix(value, "\t") || strings.HasPrefix(value, "\r") || strings.HasPrefix(value, "\n") {
		return "'" + value
	}
	return value
}

const telemetryReadme = `LWN Simulator telemetry

CSV files use UTF-8, commas, and a decimal point. Packet success rate is a
percentage: successful logical uplinks divided by started logical uplinks.
A retry or fragment does not create a new logical uplink. A packet may be
received by several gateways. Radio losses describe individual attempts,
whereas dropped uplinks describe final outcomes. Blank means unavailable
or undefined, including a rate with zero uplinks.

summary.csv reports the latest persisted metrics sample. For a running
simulation, events can be newer than this sample. The sample timestamp and
sequence describe the metrics cutoff. Older runs may contain only a stored
nonzero success rate, with an unknown denominator.

timeline.csv groups observed events into 10-second intervals of simulation
time. A row represents [start, end). Empty intervals are omitted. Device
and connected-gateway counts apply at the recorded sample timestamp.
Uplink outcomes by interval and by device require telemetry from the start
of the run. Fields ending in _observed count the matching recorded events.

radio.csv groups observations by radio protocol, frequency, and spreading
factor. Airtime samples come from completed transmissions. RSSI and SNR
samples come from recorded gateway receptions and transport ingress.
Transport frames include direction-independent input. Percentiles are nearest-rank
over samples rounded to one decimal place.

devices.csv and gateways.csv group activity by object. A blank object
property means that the information was unavailable in the run history.
`

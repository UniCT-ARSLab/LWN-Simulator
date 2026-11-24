package services

import (
	"cmp"
	"math"
	"slices"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

const telemetryIntervalMilliseconds int64 = 10_000

type telemetryTraffic struct {
	started, succeeded, dropped, retransmissions, acknowledgements int64
}

func (traffic *telemetryTraffic) observe(event contracts.SimulationEvent) {
	if event.UplinkOutcome != nil {
		switch *event.UplinkOutcome {
		case contracts.UplinkStarted:
			traffic.started++
		case contracts.UplinkAccepted:
			traffic.started++
			traffic.succeeded++
		case contracts.UplinkSucceeded:
			traffic.succeeded++
		case contracts.UplinkDropped:
			traffic.dropped++
		}
	}
	if event.Type == contracts.DeviceUplinkTransmitted && event.RetryAttempt != nil && *event.RetryAttempt > 1 {
		traffic.retransmissions++
	}
	if event.Type == contracts.UplinkACKReceived {
		traffic.acknowledgements++
	}
}

type telemetryInterval struct {
	traffic            telemetryTraffic
	collisions, losses int64
	sample             *contracts.SimulationTelemetry
}

type telemetryDevice struct {
	traffic                   telemetryTraffic
	joinRequests, joinAccepts int64
}

type telemetryGateway struct {
	received, ingress, egress, disconnected, reconnects, errors int64
}

type telemetryRadioKey struct {
	protocol  string
	frequency int64
	sf        int32
}

type telemetryHistogram struct {
	frequencies map[float64]int64
	count       int64
}

func (histogram *telemetryHistogram) observe(value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	if histogram.frequencies == nil {
		histogram.frequencies = make(map[float64]int64)
	}
	value = math.Round(value*10) / 10
	histogram.frequencies[value]++
	histogram.count++
}

func (histogram telemetryHistogram) percentiles() (p50, p95 float64) {
	values := make([]float64, 0, len(histogram.frequencies))
	for value := range histogram.frequencies {
		values = append(values, value)
	}
	slices.Sort(values)
	var cumulative int64
	rank50 := (histogram.count + 1) / 2
	rank95 := (95*histogram.count + 99) / 100
	for _, value := range values {
		cumulative += histogram.frequencies[value]
		if cumulative >= rank50 && cumulative-histogram.frequencies[value] < rank50 {
			p50 = value
		}
		if cumulative >= rank95 {
			return p50, value
		}
	}
	return p50, p95
}

type telemetryRadio struct {
	transmissions, receptions, transportFrames, collisions, lowSNR, noGateway, randomLoss, otherLoss int64
	airtime, rssi, snr                                                                               telemetryHistogram
}

type simulationTelemetryReport struct {
	intervals         map[int64]*telemetryInterval
	devices           map[string]*telemetryDevice
	gateways          map[string]*telemetryGateway
	radio             map[telemetryRadioKey]*telemetryRadio
	completeUplinkLog bool
	lastSequence      int64
	latest            *contracts.SimulationTelemetry
}

func newSimulationTelemetryReport() *simulationTelemetryReport {
	return &simulationTelemetryReport{
		intervals: make(map[int64]*telemetryInterval),
		devices:   make(map[string]*telemetryDevice),
		gateways:  make(map[string]*telemetryGateway),
		radio:     make(map[telemetryRadioKey]*telemetryRadio),
	}
}

func latestTelemetry(current, incoming *contracts.SimulationTelemetry) *contracts.SimulationTelemetry {
	if incoming != nil && (current == nil || incoming.Sequence >= current.Sequence) {
		return incoming
	}
	return current
}

func (report *simulationTelemetryReport) observe(event contracts.SimulationEvent) {
	if report.lastSequence == 0 && event.Sequence == 1 && event.Type == contracts.SimulationStarted && event.Telemetry != nil {
		report.completeUplinkLog = true
	} else if event.Sequence != report.lastSequence+1 {
		report.completeUplinkLog = false
	}
	report.lastSequence = max(report.lastSequence, event.Sequence)
	report.latest = latestTelemetry(report.latest, event.Telemetry)

	isTraffic := event.UplinkOutcome != nil || event.Type == contracts.UplinkACKReceived ||
		(event.Type == contracts.DeviceUplinkTransmitted && event.RetryAttempt != nil && *event.RetryAttempt > 1)
	if isTraffic || event.Telemetry != nil || event.Type == contracts.RadioCollisionDetected || event.Type == contracts.RadioPacketLost {
		bucket := max(0, event.TimestampMilliseconds) / telemetryIntervalMilliseconds * telemetryIntervalMilliseconds
		interval := telemetryEntry(report.intervals, bucket)
		interval.traffic.observe(event)
		interval.sample = latestTelemetry(interval.sample, event.Telemetry)
		switch event.Type {
		case contracts.RadioCollisionDetected:
			interval.collisions++
		case contracts.RadioPacketLost:
			interval.losses++
		}
	}

	if event.DeviceID != nil && (isTraffic || isJoinRequest(event) || event.Type == contracts.JoinAcceptReceived) {
		device := telemetryEntry(report.devices, *event.DeviceID)
		device.traffic.observe(event)
		if isJoinRequest(event) {
			device.joinRequests++
		}
		if event.Type == contracts.JoinAcceptReceived {
			device.joinAccepts++
		}
	}
	if event.GatewayID != nil {
		switch event.Type {
		case contracts.GatewayPacketReceived, contracts.JoinRequestReceived:
			telemetryEntry(report.gateways, *event.GatewayID).received++
		case contracts.GatewayPacketIngress:
			telemetryEntry(report.gateways, *event.GatewayID).ingress++
		case contracts.GatewayPacketEgress:
			telemetryEntry(report.gateways, *event.GatewayID).egress++
		case contracts.GatewayDisconnected:
			telemetryEntry(report.gateways, *event.GatewayID).disconnected++
		case contracts.GatewayReconnecting:
			telemetryEntry(report.gateways, *event.GatewayID).reconnects++
		case contracts.GatewayNetworkError:
			telemetryEntry(report.gateways, *event.GatewayID).errors++
		}
	}
	report.observeRadio(event)
}

func isJoinRequest(event contracts.SimulationEvent) bool {
	return event.Type == contracts.DeviceJoinRequestTransmitted ||
		(event.Type == contracts.JoinRequestReceived && event.RadioFrame == nil)
}

func (report *simulationTelemetryReport) observeRadio(event contracts.SimulationEvent) {
	switch event.Type {
	case contracts.RadioTransmissionCompleted, contracts.RadioPacketLost, contracts.GatewayPacketReceived, contracts.JoinRequestReceived, contracts.GatewayPacketIngress:
	default:
		return
	}
	if event.RadioFrame == nil && event.RadioProtocol == nil && event.ChannelFrequency == nil && event.SpreadingFactor == nil {
		return
	}
	key := telemetryRadioKey{protocol: "unknown"}
	if event.RadioFrame != nil {
		key = telemetryRadioKey{string(event.RadioFrame.Protocol), event.RadioFrame.Frequency, event.RadioFrame.SpreadingFactor}
	} else {
		if event.RadioProtocol != nil {
			key.protocol = string(*event.RadioProtocol)
		}
		if event.ChannelFrequency != nil {
			key.frequency = *event.ChannelFrequency
		}
		if event.SpreadingFactor != nil {
			key.sf = *event.SpreadingFactor
		}
	}
	if key.protocol == "" {
		key.protocol = "unknown"
	}
	radio := telemetryEntry(report.radio, key)
	switch event.Type {
	case contracts.RadioTransmissionCompleted:
		radio.transmissions++
		if event.AirtimeMilliseconds != nil {
			radio.airtime.observe(float64(*event.AirtimeMilliseconds))
		}
	case contracts.GatewayPacketReceived, contracts.JoinRequestReceived, contracts.GatewayPacketIngress:
		if event.Type == contracts.GatewayPacketIngress {
			radio.transportFrames++
		} else {
			radio.receptions++
		}
		if event.RSSI != nil {
			radio.rssi.observe(*event.RSSI)
		}
		if event.SNR != nil {
			radio.snr.observe(*event.SNR)
		}
	case contracts.RadioPacketLost:
		var reason contracts.SimulationRadioLossReason
		if event.RadioLossReason != nil {
			reason = *event.RadioLossReason
		}
		switch reason {
		case contracts.Collision:
			radio.collisions++
		case contracts.LowSNR:
			radio.lowSNR++
		case contracts.NoGateway:
			radio.noGateway++
		case contracts.RandomLoss:
			radio.randomLoss++
		default:
			radio.otherLoss++
		}
	}
}

func telemetryEntry[K comparable, V any](entries map[K]*V, key K) *V {
	entry := entries[key]
	if entry == nil {
		entry = new(V)
		entries[key] = entry
	}
	return entry
}

func compareTelemetryRadioKeys(a, b telemetryRadioKey) int {
	if order := cmp.Compare(a.protocol, b.protocol); order != 0 {
		return order
	}
	if order := cmp.Compare(a.frequency, b.frequency); order != 0 {
		return order
	}
	return cmp.Compare(a.sf, b.sf)
}

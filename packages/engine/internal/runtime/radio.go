package runtime

import (
	"encoding/base64"
	"math"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/geometry"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

const (
	radioTransmitPowerDBm = 14.0
	radioNoiseFloorDBm    = -120.0
	radioCodingRate       = 1
	radioBandwidthHz      = 125_000
)

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func (e *Engine) startRadioTransmissionLocked(
	device contracts.Device,
	scheduled types.ScheduledEvent,
	packetID string,
	frameCounter int64,
	confirmed bool,
) []contracts.SimulationEvent {
	session := e.sessions[device.ID]
	dataRate := currentDeviceDataRate(session, device)
	if scheduled.DataRate > 0 {
		dataRate = regional.ClampDataRate(device.LocationConfig.Region, scheduled.DataRate)
	}
	payloadSize := payloadSizeForFragment(device, dataRate, scheduled.FragmentIndex, session)
	channel := selectRadioChannelAtDataRate(device, session, frameCounter, dataRate)
	gateways := e.coveredGateways(device)
	transmitPower := radioTransmitPowerDBm
	if session != nil {
		transmitPower = float64(session.CurrentTxPower)
	}
	rssi, snr := calculateSignal(device, gateways, channel.Frequency, transmitPower)
	payload := e.buildUplinkPayloadLocked(device, session, frameCounter, confirmed, scheduled.FragmentIndex, dataRate)
	airtimeSize := payloadSize
	if deviceRadioProtocol(device) == contracts.LoRaWAN && len(payload) >= 8 && (payload[0]>>5 == lorawan.MTypeUnconfirmedDataUp || payload[0]>>5 == lorawan.MTypeConfirmedDataUp) {
		airtimeSize += int(payload[5] & 15)
	}
	airtime := calculateAirtimeForPayload(airtimeSize, channel.SpreadingFactor, channel.Bandwidth)
	transmission := &types.RadioTransmission{
		RadioFrame: types.RadioFrame{
			Protocol:          deviceRadioProtocol(device),
			Payload:           append([]byte(nil), payload...),
			ChannelFrequency:  channel.Frequency,
			Bandwidth:         channel.Bandwidth,
			SpreadingFactor:   channel.SpreadingFactor,
			DataRate:          dataRate,
			TransmissionPower: transmitPower,
		},
		PacketID:      packetID,
		DeviceID:      device.ID,
		FrameCounter:  frameCounter,
		Attempt:       scheduled.Attempt,
		Confirmed:     confirmed,
		StartAt:       scheduled.At,
		EndAt:         scheduled.At + airtime,
		FPort:         device.FrameConfig.FPort,
		PayloadSize:   payloadSize,
		FragmentIndex: scheduled.FragmentIndex,
		FragmentCount: scheduled.FragmentCount,
		FPendingPoll:  scheduled.FPendingPoll,
		Airtime:       airtime,
		RSSI:          rssi,
		SNR:           snr,
		GatewayIDs:    gatewayIDs(gateways),
	}
	for _, active := range e.radio {
		if active.EndAt > transmission.StartAt &&
			active.ChannelFrequency == transmission.ChannelFrequency &&
			active.Bandwidth == transmission.Bandwidth &&
			active.SpreadingFactor == transmission.SpreadingFactor &&
			sharesGateway(active.GatewayIDs, transmission.GatewayIDs) {
			active.Collision = true
			transmission.Collision = true
		}
	}
	e.radio[packetID] = transmission

	if session := e.sessions[device.ID]; session != nil {
		session.CurrentDataRate = dataRate
		session.CurrentSpreadingFactor = channel.SpreadingFactor
		session.LastRSSI = rssi
		session.LastSNR = snr
		session.LastAirtime = airtime
		session.LastChannel = channel.Frequency
		session.LastPayloadSize = payloadSize
		session.LastFPort = device.FrameConfig.FPort
	}
	e.metrics.TotalTransmissions++
	e.metrics.AverageRSSI = runningAverage(e.metrics.AverageRSSI, e.metrics.TotalTransmissions, rssi)
	e.metrics.AverageSNR = runningAverage(e.metrics.AverageSNR, e.metrics.TotalTransmissions, snr)
	e.metrics.AverageAirtimeMilliseconds = runningAverage(e.metrics.AverageAirtimeMilliseconds, e.metrics.TotalTransmissions, float64(airtime.Milliseconds()))

	events := []contracts.SimulationEvent{e.newRadioEventLocked(
		contracts.RadioTransmissionStarted,
		"radio transmission started",
		device.ID,
		"",
		scheduled.ID,
		packetID,
		transmission,
		false,
		"",
	)}
	completion := types.ScheduledEvent{
		ID: uuid.NewString(), At: transmission.EndAt, Type: contracts.RadioTransmissionCompleted,
		Message: "radio transmission completed", DeviceID: device.ID,
		Kind: types.ScheduledEventRadioComplete, Attempt: scheduled.Attempt,
		PacketID: packetID, FrameCounter: frameCounter, Confirmed: confirmed,
	}
	if err := e.scheduler.Schedule(completion); err != nil {
		delete(e.radio, packetID)
		events = append(events, e.newRadioEventLocked(
			contracts.RadioPacketLost, "radio completion could not be scheduled", device.ID, "", scheduled.ID,
			packetID, transmission, true, contracts.RandomLoss,
		))
		return events
	}
	events = append(events, e.newEventLocked(contracts.SchedulerEventScheduled, "radio transmission completion scheduled", device.ID, "", completion.ID))
	return events
}

func (e *Engine) processRadioTransmissionCompletedLocked(scheduled types.ScheduledEvent) []contracts.SimulationEvent {
	transmission, ok := e.radio[scheduled.PacketID]
	if !ok {
		e.metrics.FrameCounterErrors++
		return []contracts.SimulationEvent{e.newEventLocked(contracts.FrameCounterRejected, "radio transmission is no longer active", scheduled.DeviceID, "", scheduled.ID)}
	}
	delete(e.radio, scheduled.PacketID)

	events := []contracts.SimulationEvent{e.newRadioEventLocked(
		contracts.RadioTransmissionCompleted,
		"radio transmission completed",
		transmission.DeviceID, "", scheduled.ID, transmission.PacketID, transmission, transmission.Collision, "",
	)}
	reason, lost := e.radioLossReason(transmission)
	if lost {
		e.metrics.RadioPacketLosses++
		if reason == contracts.Collision {
			e.metrics.Collisions++
			events = append(events, e.newRadioEventLocked(
				contracts.RadioCollisionDetected, "radio collision detected", transmission.DeviceID, "", scheduled.ID,
				transmission.PacketID, transmission, true, reason,
			))
		}
		if reason == contracts.LowSNR {
			e.metrics.LowSNRLosses++
		}
		events = append(events, e.newRadioEventLocked(
			contracts.RadioPacketLost, "radio packet lost", transmission.DeviceID, "", scheduled.ID,
			transmission.PacketID, transmission, true, reason,
		))
		return append(events, e.completeRadioLossLocked(transmission, scheduled.ID)...)
	}

	for _, gatewayID := range transmission.GatewayIDs {
		e.metrics.TotalPacketsReceived++
		events = append(events, e.newRadioEventLocked(
			gatewayEventType(transmission), "virtual gateway received radio packet", transmission.DeviceID,
			gatewayID, scheduled.ID, transmission.PacketID, transmission, false, "",
		))
		if len(transmission.Payload) > 0 {
			if gateway, ok := e.registry.Gateway(gatewayID); ok && gateway.Type == contracts.Virtual {
				if _, connected := e.gatewayAdapters[gatewayID]; connected {
					e.gatewayPackets = append(e.gatewayPackets, types.GatewayPacket{
						GatewayID:       gatewayID,
						RSSI:            transmission.RSSI,
						SNR:             transmission.SNR,
						DeviceID:        transmission.DeviceID,
						Kind:            types.GatewayPacketUplink,
						Protocol:        transmission.Protocol,
						Payload:         append([]byte(nil), transmission.Payload...),
						Frequency:       transmission.ChannelFrequency,
						Bandwidth:       transmission.Bandwidth,
						SpreadingFactor: transmission.SpreadingFactor,
						DataRate:        dataRateString(transmission.SpreadingFactor, transmission.Bandwidth),
					})
				}
			}
		}
	}
	return append(events, e.completeRadioSuccessLocked(transmission, scheduled.ID)...)
}

func (e *Engine) completeRadioLossLocked(transmission *types.RadioTransmission, scheduledID string) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(transmission.DeviceID)
	if !ok {
		return nil
	}
	if session.PendingJoinRequest != nil && session.PendingJoinRequest.PacketID == transmission.PacketID {
		session.PendingJoinRequest.GatewayIDs = nil
		return []contracts.SimulationEvent{e.scheduleWindowLocked(device, radioWindowSource(transmission), contracts.RX1, transmission.PacketID, -1, false)}
	}
	if !transmission.Confirmed {
		if pending := session.PendingUplink; pending != nil && pending.PacketID == transmission.PacketID {
			if pending.Attempt < session.UnconfirmedRepetitions {
				return e.scheduleUnconfirmedRepetitionLocked(device, pending, transmission.EndAt)
			}
			if pending.AnySuccessful {
				session.PendingUplink = nil
				e.applyADRLocked(device, session)
				e.metrics.SuccessfulUplinks++
				e.metrics.PacketSuccessRate = e.packetSuccessRate()
				return append([]contracts.SimulationEvent{e.newUplinkOutcomeEventLocked(contracts.MetricsUpdated, "simulation metrics updated", transmission.DeviceID, "", scheduledID, transmission.PacketID, contracts.UplinkSucceeded)}, e.scheduleNextUplinkAfterTransmissionLocked(device, transmission)...)
			}
			session.PendingUplink = nil
		}
		e.metrics.DroppedUplinks++
		e.metrics.PacketSuccessRate = e.packetSuccessRate()
		events := []contracts.SimulationEvent{
			e.newUplinkOutcomeEventLocked(contracts.PacketDropped, "uplink dropped by radio model", transmission.DeviceID, "", scheduledID, transmission.PacketID, contracts.UplinkDropped),
			e.newPacketEventLocked(contracts.MetricsUpdated, "simulation metrics updated", transmission.DeviceID, "", scheduledID, transmission.PacketID),
		}
		return append(events, e.scheduleNextUplinkAfterTransmissionLocked(device, transmission)...)
	}
	if session.PendingUplink == nil || session.PendingUplink.PacketID != transmission.PacketID {
		return nil
	}
	session.PendingUplink.GatewayIDs = nil
	return []contracts.SimulationEvent{e.scheduleWindowLocked(device, radioWindowSource(transmission), contracts.RX1, transmission.PacketID, transmission.FrameCounter, true)}
}

func (e *Engine) completeRadioSuccessLocked(transmission *types.RadioTransmission, scheduledID string) []contracts.SimulationEvent {
	device, session, ok := e.deviceSessionLocked(transmission.DeviceID)
	if !ok {
		return nil
	}
	if session.PendingJoinRequest != nil && session.PendingJoinRequest.PacketID == transmission.PacketID {
		session.PendingJoinRequest.GatewayIDs = append([]string(nil), transmission.GatewayIDs...)
		return []contracts.SimulationEvent{e.scheduleWindowLocked(device, radioWindowSource(transmission), contracts.RX1, transmission.PacketID, -1, false)}
	}
	if session.PendingUplink == nil || session.PendingUplink.PacketID != transmission.PacketID {
		if !transmission.Confirmed {
			e.applyADRLocked(device, session)
			e.metrics.SuccessfulUplinks++
			e.metrics.PacketSuccessRate = e.packetSuccessRate()
			events := []contracts.SimulationEvent{e.newUplinkOutcomeEventLocked(contracts.MetricsUpdated, "simulation metrics updated", transmission.DeviceID, "", scheduledID, transmission.PacketID, contracts.UplinkSucceeded)}
			if len(session.PendingClassADownlinks) > 0 {
				events = append(events, e.scheduleWindowLocked(device, radioWindowSource(transmission), contracts.RX1, transmission.PacketID, transmission.FrameCounter, false))
			}
			return append(events, e.scheduleNextUplinkAfterTransmissionLocked(device, transmission)...)
		}
		return nil
	}
	session.PendingUplink.SentAt = time.Now()
	session.PendingUplink.GatewayIDs = append([]string(nil), transmission.GatewayIDs...)
	if transmission.Confirmed {
		return []contracts.SimulationEvent{e.scheduleWindowLocked(device, radioWindowSource(transmission), contracts.RX1, transmission.PacketID, transmission.FrameCounter, true)}
	}
	session.PendingUplink.AnySuccessful = true
	if session.PendingUplink.Attempt < session.UnconfirmedRepetitions {
		return e.scheduleUnconfirmedRepetitionLocked(device, session.PendingUplink, transmission.EndAt)
	}
	if session.PendingUplink.FragmentIndex+1 < session.PendingUplink.FragmentCount {
		session.PendingUplink.FragmentIndex++
		session.PendingUplink.AnySuccessful = false
		session.PendingUplink.GatewayIDs = nil
		next := types.ScheduledEvent{
			ID: uuid.NewString(), At: transmission.EndAt, Type: contracts.DeviceUplinkTransmitted,
			Message: "next application payload fragment scheduled", DeviceID: device.ID,
			Kind: types.ScheduledEventDeviceUplink, Attempt: 1, PacketID: transmission.PacketID,
			FrameCounter: transmission.FrameCounter, Confirmed: false,
			FragmentIndex: session.PendingUplink.FragmentIndex, FragmentCount: session.PendingUplink.FragmentCount,
		}
		if err := e.scheduler.Schedule(next); err != nil {
			session.PendingUplink = nil
			return []contracts.SimulationEvent{e.newEventLocked(contracts.PacketDropped, "next payload fragment could not be scheduled", device.ID, "", next.ID)}
		}
		return []contracts.SimulationEvent{e.newEventLocked(contracts.DeviceUplinkScheduled, "next application payload fragment scheduled", device.ID, "", next.ID)}
	}
	session.PendingUplink = nil
	e.applyADRLocked(device, session)
	e.metrics.SuccessfulUplinks++
	e.metrics.PacketSuccessRate = e.packetSuccessRate()
	events := []contracts.SimulationEvent{e.newUplinkOutcomeEventLocked(contracts.MetricsUpdated, "simulation metrics updated", transmission.DeviceID, "", scheduledID, transmission.PacketID, contracts.UplinkSucceeded)}
	return append(events, e.scheduleNextUplinkAfterTransmissionLocked(device, transmission)...)
}

func (e *Engine) scheduleUnconfirmedRepetitionLocked(device contracts.Device, pending *types.PendingUplink, at time.Duration) []contracts.SimulationEvent {
	pending.Attempt++
	pending.GatewayIDs = nil
	repetition := types.ScheduledEvent{
		ID: uuid.NewString(), At: at, Type: contracts.DeviceUplinkTransmitted,
		Message: "unconfirmed uplink repetition scheduled", DeviceID: device.ID,
		Kind: types.ScheduledEventDeviceUplink, Attempt: pending.Attempt,
		PacketID: pending.PacketID, FrameCounter: pending.FrameCounter,
		FragmentIndex: pending.FragmentIndex, FragmentCount: pending.FragmentCount,
	}
	if err := e.scheduler.Schedule(repetition); err != nil {
		session := e.sessions[device.ID]
		if session != nil {
			session.PendingUplink = nil
		}
		e.metrics.DroppedUplinks++
		e.metrics.PacketSuccessRate = e.packetSuccessRate()
		return []contracts.SimulationEvent{e.newUplinkOutcomeEventLocked(contracts.PacketDropped, "unconfirmed uplink repetition could not be scheduled", device.ID, "", repetition.ID, pending.PacketID, contracts.UplinkDropped)}
	}
	return []contracts.SimulationEvent{e.newProtocolEventLocked(
		contracts.UplinkRetryScheduled, "unconfirmed uplink repetition scheduled", device.ID,
		"", repetition.ID, pending.PacketID, pending.FrameCounter, pending.Attempt, false, nil,
	)}
}

func (e *Engine) scheduleNextUplinkAfterTransmissionLocked(device contracts.Device, transmission *types.RadioTransmission) []contracts.SimulationEvent {
	if transmission.FPendingPoll {
		return nil
	}
	nextAt := transmission.StartAt + durationSecondsFloat(device.PayloadConfig.UplinkInterval)
	if session := e.sessions[device.ID]; session != nil && session.MaximumDutyCycle > 0 && session.MaximumDutyCycle < 1 {
		dutyCycleAt := transmission.EndAt + time.Duration(float64(transmission.Airtime)*(1/session.MaximumDutyCycle-1))
		if dutyCycleAt > nextAt {
			nextAt = dutyCycleAt
		}
	}
	if nextAt < transmission.EndAt {
		nextAt = transmission.EndAt
	}
	return e.scheduleUplinkAtLocked(device, nextAt)
}

func (e *Engine) newRadioEventLocked(eventType contracts.SimulationEventType, message, deviceID, gatewayID, scheduledID, packetID string, transmission *types.RadioTransmission, collision bool, reason contracts.SimulationRadioLossReason) contracts.SimulationEvent {
	event := e.newProtocolEventLocked(eventType, message, deviceID, gatewayID, scheduledID, packetID, transmission.FrameCounter, transmission.Attempt, transmission.Confirmed, nil)
	channelFrequency := transmission.ChannelFrequency
	bandwidth := transmission.Bandwidth
	spreadingFactor := int32(transmission.SpreadingFactor)
	airtimeMilliseconds := transmission.Airtime.Milliseconds()
	rssi := transmission.RSSI
	snr := transmission.SNR
	dataRate := transmission.DataRate
	fPort := transmission.FPort
	payloadSize := int64(transmission.PayloadSize)
	event.ChannelFrequency = &channelFrequency
	event.Bandwidth = &bandwidth
	event.SpreadingFactor = &spreadingFactor
	event.DataRate = &dataRate
	event.FPort = &fPort
	event.PayloadSize = &payloadSize
	event.AirtimeMilliseconds = &airtimeMilliseconds
	event.RSSI = &rssi
	event.SNR = &snr
	event.Collision = &collision
	event.RadioFrame = &contracts.RadioFrame{
		Protocol:              transmission.Protocol,
		Payload:               append([]byte(nil), transmission.Payload...),
		Frequency:             transmission.ChannelFrequency,
		Bandwidth:             transmission.Bandwidth,
		SpreadingFactor:       int32(transmission.SpreadingFactor),
		CodingRate:            "4/5",
		TransmissionPower:     int32(transmission.TransmissionPower),
		SenderID:              transmission.DeviceID,
		GatewayID:             optionalString(gatewayID),
		TimestampMilliseconds: transmission.StartAt.Milliseconds(),
		AirtimeMilliseconds:   &airtimeMilliseconds,
	}
	device, ok := e.registry.Device(deviceID)
	if ok {
		decorateDecodedRadioPayload(&event, device, transmission, e.sessions[deviceID])
	}
	if reason != "" {
		reasonValue := reason
		event.RadioLossReason = &reasonValue
	}
	e.eventLog[len(e.eventLog)-1] = event
	return event
}

func (e *Engine) radioLossReason(transmission *types.RadioTransmission) (contracts.SimulationRadioLossReason, bool) {
	if len(transmission.GatewayIDs) == 0 {
		return contracts.NoGateway, true
	}
	if transmission.Collision {
		return contracts.Collision, true
	}
	minimumSNR := minimumRequiredSNR(transmission.SpreadingFactor)
	if transmission.SNR < minimumSNR {
		return contracts.LowSNR, true
	}
	if transmission.SNR < minimumSNR+6 {
		probability := math.Min(0.8, (minimumSNR+6-transmission.SNR)/12)
		if e.rng.Float64() < probability {
			return contracts.RandomLoss, true
		}
	}
	return "", false
}

func sharesGateway(left, right []string) bool {
	for _, leftID := range left {
		for _, rightID := range right {
			if leftID == rightID {
				return true
			}
		}
	}
	return false
}

func gatewayEventType(transmission *types.RadioTransmission) contracts.SimulationEventType {
	if transmission.Protocol == contracts.RawLoRa {
		return contracts.GatewayPacketReceived
	}
	if transmission.Confirmed || transmission.FrameCounter > 0 {
		return contracts.GatewayPacketReceived
	}
	return contracts.JoinRequestReceived
}

func radioWindowSource(transmission *types.RadioTransmission) types.ScheduledEvent {
	return types.ScheduledEvent{
		At: transmission.EndAt, DeviceID: transmission.DeviceID, PacketID: transmission.PacketID,
		FrameCounter: transmission.FrameCounter, Attempt: transmission.Attempt, Confirmed: transmission.Confirmed,
		ChannelFrequency: transmission.ChannelFrequency, WindowBaseAt: transmission.EndAt,
	}
}

func selectRadioChannel(device contracts.Device, frameCounter int64) types.RadioChannel {
	return selectRadioChannelAtDataRate(device, nil, frameCounter, defaultUplinkDataRate(device.LocationConfig.Region))
}

func selectRadioChannelAtDataRate(device contracts.Device, session *types.DeviceSession, frameCounter int64, dataRate int) types.RadioChannel {
	if session != nil && (len(session.NetworkChannels) > 0 || len(session.DisabledChannels) > 0) {
		var channels []types.RadioChannel
		profile := regional.Profile(device.LocationConfig.Region, dataRate)
		table := externalChannelTable(device, session)
		for i := 0; i < 256; i++ {
			ch, ok := table[i]
			if ok && !session.DisabledChannels[i] && dataRate >= ch.MinimumDataRate && dataRate <= ch.MaximumDataRate {
				channels = append(channels, types.RadioChannel{Frequency: ch.Frequency, DataRate: dataRate, Bandwidth: profile.Bandwidth, SpreadingFactor: profile.SpreadingFactor})
			}
		}
		if len(channels) > 0 {
			return channels[int(maxInt64(frameCounter, 0))%len(channels)]
		}
	}
	channels := regionalChannels(device.LocationConfig.Region, dataRate)
	if session != nil {
		for _, channel := range session.AdditionalChannels {
			if channel.DataRate == dataRate {
				channels = append(channels, channel)
			}
		}
	}
	index := int(frameCounter)
	if index < 0 {
		index = 0
	}
	return channels[index%len(channels)]
}

func regionalChannels(region contracts.DeviceRegion, dataRate int) []types.RadioChannel {
	return regional.UplinkChannels(region, dataRate)
}

func defaultUplinkDataRate(region contracts.DeviceRegion) int {
	return regional.MustPlan(region).DefaultDataRate
}

func spreadingFactorForDataRate(region contracts.DeviceRegion, dataRate int) int {
	return regional.Profile(region, dataRate).SpreadingFactor
}

func currentDeviceDataRate(session *types.DeviceSession, device contracts.Device) int {
	if session != nil && regional.SupportsUplinkDataRate(device.LocationConfig.Region, session.CurrentDataRate) {
		return session.CurrentDataRate
	}
	if configured := device.AdvancedConfig.UplinkDataRate; configured != nil {
		return regional.ClampDataRate(device.LocationConfig.Region, *configured)
	}
	return defaultUplinkDataRate(device.LocationConfig.Region)
}

func effectivePayloadSize(device contracts.Device, dataRate int) int {
	return payloadSizeForFragment(device, dataRate, 0)
}

func payloadFragmentCount(device contracts.Device, dataRate int, sessions ...*types.DeviceSession) int {
	payloadSize := len(payloadBytes(device))
	maximum := applicationPayloadCapacity(device, dataRate, sessions...)
	if payloadSize <= maximum || device.PayloadConfig.OversizedPayloadBehavior != contracts.Fragment {
		return 1
	}
	return (payloadSize + maximum - 1) / maximum
}

func payloadSizeForFragment(device contracts.Device, dataRate, fragmentIndex int, sessions ...*types.DeviceSession) int {
	payloadSize := len(payloadBytes(device))
	maximum := applicationPayloadCapacity(device, dataRate, sessions...)
	if payloadSize <= maximum {
		return payloadSize
	}
	if device.PayloadConfig.OversizedPayloadBehavior != contracts.Fragment {
		return maximum
	}
	start := fragmentIndex * maximum
	if start >= payloadSize {
		return 0
	}
	remaining := payloadSize - start
	if remaining > maximum {
		return maximum
	}
	return remaining
}

func payloadBytes(device contracts.Device) []byte {
	if device.PayloadConfig.Base64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(device.PayloadConfig.Payload)
		if err == nil {
			return decoded
		}
		return nil
	}
	return []byte(device.PayloadConfig.Payload)
}

func maximumPayloadSize(region contracts.DeviceRegion, dataRate int) int {
	return regional.Profile(region, dataRate).MaximumPayload
}

func calculateAirtime(device contracts.Device, sf int, bandwidth int64) time.Duration {
	return calculateAirtimeForPayload(len(payloadBytes(device)), sf, bandwidth)
}

func calculateAirtimeForPayload(payloadSize, sf int, bandwidth int64) time.Duration {
	payloadSize += 13
	symbolDuration := math.Pow(2, float64(sf)) / float64(bandwidth)
	lowDataRateOptimization := 0
	if bandwidth == 125_000 && sf >= 11 {
		lowDataRateOptimization = 1
	}
	denominator := 4 * float64(sf-2*lowDataRateOptimization)
	payloadSymbols := 8.0
	if denominator > 0 {
		payloadSymbols += math.Max(math.Ceil((8*float64(payloadSize)-4*float64(sf)+28+16)/(denominator))*float64(radioCodingRate+4), 0)
	}
	seconds := (8 + 4.25 + payloadSymbols) * symbolDuration
	return time.Duration(math.Max(1, math.Ceil(seconds*float64(time.Second))))
}

func calculateSignal(device contracts.Device, gateways []contracts.Gateway, frequency int64, transmitPower float64) (float64, float64) {
	if len(gateways) == 0 {
		return -120, -30
	}
	minimumDistance := math.MaxFloat64
	latitude, longitude := locationCoordinates(device)
	for _, gateway := range gateways {
		gatewayLatitude, gatewayLongitude := gatewayCoordinates(gateway)
		distance := geometry.DistanceMeters(latitude, longitude, gatewayLatitude, gatewayLongitude)
		verticalDistance := float64(pointerValue(device.LocationConfig.Altitude)) - float64(pointerValue(gateway.Altitude))
		distance = math.Sqrt(distance*distance + verticalDistance*verticalDistance)
		if distance < minimumDistance {
			minimumDistance = distance
		}
	}
	distanceKilometers := math.Max(minimumDistance/1000, 0.001)
	pathLoss := 32.44 + 20*math.Log10(float64(frequency)/1_000_000) + 20*math.Log10(distanceKilometers)
	rssi := transmitPower - pathLoss
	return rssi, rssi - radioNoiseFloorDBm
}

func minimumRequiredSNR(spreadingFactor int) float64 {
	thresholds := map[int]float64{7: -7.5, 8: -10, 9: -12.5, 10: -15, 11: -17.5, 12: -20}
	if threshold, ok := thresholds[spreadingFactor]; ok {
		return threshold
	}
	return -20
}

func runningAverage(previous float64, count int64, value float64) float64 {
	if count <= 1 {
		return value
	}
	return previous + (value-previous)/float64(count)
}

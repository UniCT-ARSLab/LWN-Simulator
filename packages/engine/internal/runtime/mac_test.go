package runtime

import (
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestMACCommandsUpdateDeviceRuntimeState(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	session := newDeviceSession(device)
	dataRate, txPower, repetitions := 2, 4, 3
	exponent := 3
	rx2Rate, offset := 3, 1
	frequency := int64(869_525_000)
	delay := 4 * time.Second

	commands := []types.MACCommand{
		{Type: types.MACLinkADRReq, DataRate: &dataRate, TxPower: &txPower, NbTrans: &repetitions},
		{Type: types.MACDutyCycleReq, MaxDutyCycleExponent: &exponent},
		{Type: types.MACRXParamSetupReq, DataRate: &rx2Rate, RX1DataRateOffset: &offset, Frequency: &frequency},
		{Type: types.MACRXTimingSetupReq, Delay: &delay},
	}
	for _, command := range commands {
		if err := validateMACCommand(device.LocationConfig.Region, command); err != nil {
			t.Fatalf("validate command %s: %v", command.Type, err)
		}
	}
	applyMACCommands(device, session, commands)

	if session.CurrentDataRate != 2 || session.CurrentSpreadingFactor != 10 || session.CurrentTxPower != 4 || session.UnconfirmedRepetitions != 3 {
		t.Fatalf("LinkADRReq was not applied: %+v", session)
	}
	if session.MaximumDutyCycle != 0.125 || session.RX2DataRate != 3 || session.RX2Frequency != frequency || session.RX1DataRateOffset != 1 || session.ReceiveDelay != delay {
		t.Fatalf("MAC runtime settings were not applied: %+v", session)
	}
}

func TestEverySupportedMACCommandValidatesAndAppliesItsParameters(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000206")
	configureABP(&device)
	session := newDeviceSession(device)
	margin, dataRate, power, repetitions := 10, 2, 5, 3
	exponent, offset := 4, 1
	channelIndex, minimumRate, maximumRate := 3, 1, 4
	frequency := int64(868_700_000)
	delay := 6 * time.Second
	uplinkDwell, downlinkDwell := true, true
	maximumEIRP, periodicity := 12, 4
	deviceTime := time.Unix(1_700_000_000, 0)

	commands := []types.MACCommand{
		{Type: types.MACLinkCheckAns, Margin: &margin},
		{Type: types.MACLinkADRReq, DataRate: &dataRate, TxPower: &power, NbTrans: &repetitions},
		{Type: types.MACDutyCycleReq, MaxDutyCycleExponent: &exponent},
		{Type: types.MACRXParamSetupReq, DataRate: &dataRate, Frequency: &frequency, RX1DataRateOffset: &offset},
		{Type: types.MACDevStatusReq},
		{Type: types.MACNewChannelReq, ChannelIndex: &channelIndex, Frequency: &frequency, MinimumDataRate: &minimumRate, MaximumDataRate: &maximumRate},
		{Type: types.MACRXTimingSetupReq, Delay: &delay},
		{Type: types.MACTXParamSetupReq, UplinkDwellTime: &uplinkDwell, DownlinkDwellTime: &downlinkDwell, MaximumEIRP: &maximumEIRP},
		{Type: types.MACDLChannelReq, ChannelIndex: &channelIndex, Frequency: &frequency},
		{Type: types.MACDeviceTimeAns, DeviceTime: &deviceTime},
		{Type: types.MACPingSlotInfoAns, PingSlotPeriodicity: &periodicity},
		{Type: types.MACPingSlotChannelReq, Frequency: &frequency, DataRate: &dataRate},
		{Type: types.MACBeaconFreqReq, Frequency: &frequency},
	}
	for _, command := range commands {
		if err := validateMACCommand(device.LocationConfig.Region, command); err != nil {
			t.Fatalf("valid %s command was rejected: %v", command.Type, err)
		}
	}
	applyMACCommands(device, session, commands)
	if session.CurrentDataRate != dataRate || session.CurrentTxPower != power || session.UnconfirmedRepetitions != repetitions {
		t.Fatalf("LinkADRReq parameters were not applied: %+v", session)
	}
	if session.MaximumDutyCycle != 1.0/16 || session.RX1DataRateOffset != offset || session.RX2DataRate != dataRate || session.RX2Frequency != frequency || session.ReceiveDelay != delay {
		t.Fatalf("duty-cycle or receive parameters were not applied: %+v", session)
	}
	if !session.UplinkDwellTime || !session.DownlinkDwellTime || session.MaximumEIRP != maximumEIRP {
		t.Fatalf("TX parameters were not applied: %+v", session)
	}
	if session.PingSlotPeriodicity != periodicity || session.PingSlotFrequency != frequency || session.PingSlotDataRate != dataRate || session.BeaconFrequency != frequency {
		t.Fatalf("Class B parameters were not applied: %+v", session)
	}
	if len(session.AdditionalChannels) != 1 || session.AdditionalChannels[0].Frequency != frequency || session.AdditionalChannels[0].DataRate != minimumRate {
		t.Fatalf("new channel parameters were not applied: %+v", session.AdditionalChannels)
	}
	if got := classBPingSlotPeriod(session); got != 16*time.Second {
		t.Fatalf("ping-slot periodicity produced %s", got)
	}
}

func TestMACCommandValidationRejectsMissingAndOutOfRangeParameters(t *testing.T) {
	invalid := []types.MACCommand{
		{Type: types.MACLinkCheckAns, Margin: intPtr(255)},
		{Type: types.MACLinkADRReq, DataRate: intPtr(99)},
		{Type: types.MACLinkADRReq, TxPower: intPtr(16)},
		{Type: types.MACLinkADRReq, NbTrans: intPtr(0)},
		{Type: types.MACDutyCycleReq},
		{Type: types.MACRXParamSetupReq},
		{Type: types.MACRXParamSetupReq, DataRate: intPtr(3)},
		{Type: types.MACNewChannelReq},
		{Type: types.MACNewChannelReq, ChannelIndex: intPtr(1), Frequency: pointerTo(int64(868_100_001)), MinimumDataRate: intPtr(0), MaximumDataRate: intPtr(5)},
		{Type: types.MACRXTimingSetupReq, Delay: pointerTo(1500 * time.Millisecond)},
		{Type: types.MACTXParamSetupReq, MaximumEIRP: intPtr(16)},
		{Type: types.MACDLChannelReq},
		{Type: types.MACDeviceTimeAns},
		{Type: types.MACPingSlotChannelReq, Frequency: pointerTo(int64(869_525_000))},
		{Type: types.MACBeaconFreqReq},
		{Type: types.MACCommandType("unsupported")},
	}
	for _, command := range invalid {
		if err := validateMACCommand(contracts.EU868, command); err == nil {
			t.Fatalf("invalid %s command was accepted: %+v", command.Type, command)
		}
	}
	if got := classBPingSlotPeriod(nil); got != types.ClassBPingSlotPeriod {
		t.Fatalf("nil session should use default ping period: %s", got)
	}
	session := &types.DeviceSession{PingSlotPeriodicity: 9}
	if got := classBPingSlotPeriod(session); got != 128*time.Second {
		t.Fatalf("periodicity should clamp to seven: %s", got)
	}
}

func TestQueueMACCommandUsesClassCDownlinkPath(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.Class = contracts.ClassC
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")
	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	dataRate, repetitions := 2, 3
	if _, err := engine.QueueMACCommand(device.ID, types.MACCommand{Type: types.MACLinkADRReq, DataRate: &dataRate, NbTrans: &repetitions}); err != nil {
		t.Fatalf("queue MAC command: %v", err)
	}
	waitForEventType(t, engine.Events(), contracts.DeviceDownlinkScheduled)
	dataRate = 5
	repetitions = 1
	waitForEventType(t, engine.Events(), contracts.DeviceDownlinkTransmitted)

	engine.mu.RLock()
	session := *engine.sessions[device.ID]
	engine.mu.RUnlock()
	if session.CurrentDataRate != 2 || session.UnconfirmedRepetitions != 3 {
		t.Fatalf("queued MAC command was not applied: %+v", session)
	}
}

func TestLinkADRRepetitionsReuseFrameCounterAndCompleteOnce(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{Clock: clock, EventBuffer: 256})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	engine.sessions[device.ID].UnconfirmedRepetitions = 3
	startEngine(t, engine)
	defer stopEngine(t, engine)

	if err := clock.Advance(10 * time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	waitForEventCounts(t, engine.Events(), map[contracts.SimulationEventType]int{
		contracts.DeviceUplinkTransmitted: 3,
		contracts.GatewayPacketReceived:   3,
	})

	snapshot := engine.Snapshot()
	if snapshot.Metrics.TotalUplinks != 1 || snapshot.Metrics.Retransmissions != 2 || snapshot.Metrics.SuccessfulUplinks != 1 {
		t.Fatalf("unexpected repetition metrics: %+v", snapshot.Metrics)
	}
	if snapshot.Devices[0].FrameCounterUp != 1 {
		t.Fatalf("repetitions must reuse one frame counter: %+v", snapshot.Devices[0])
	}
}

func TestFPendingSchedulesOneClassAPollWithoutDuplicatingPeriodicTraffic(t *testing.T) {
	clock := types.NewManualClock(0)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000001")
	configureABP(&device)
	device.PayloadConfig.UplinkInterval = 60
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")
	engine := newTestEngine(t, clock, device, gateway)
	defer stopEngine(t, engine)

	if _, err := engine.QueueDownlink(types.Downlink{DeviceID: device.ID, Payload: []byte("more"), FPending: true, DataRate: -1}); err != nil {
		t.Fatalf("queue FPending downlink: %v", err)
	}
	if err := clock.Advance(10 * time.Second); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	waitForEventCounts(t, engine.Events(), map[contracts.SimulationEventType]int{
		contracts.DeviceDownlinkTransmitted: 1,
		contracts.DeviceUplinkTransmitted:   2,
	})

	engine.mu.RLock()
	periodicUplinks := 0
	for _, scheduled := range engine.scheduler.Events() {
		if scheduled.Kind == types.ScheduledEventDeviceUplink && !scheduled.FPendingPoll {
			periodicUplinks++
		}
	}
	engine.mu.RUnlock()
	if periodicUplinks != 1 {
		t.Fatalf("expected one periodic uplink chain after FPending, got %d", periodicUplinks)
	}
}

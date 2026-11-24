package runtime

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestPhaseFiveEngineTracksRealGatewayLifecycleAndPackets(t *testing.T) {
	adapter := newFakeGatewayAdapter()
	factory := &fakeGatewayFactory{adapter: adapter}
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01")
	gateway.Type = contracts.Real
	gateway.KeepAlive = nil
	gateway.GatewayEUI = "A840410001000101"
	gateway.GatewayIPv4 = engineStringPtr("127.0.0.1")
	gateway.GatewayPort = engineInt32Ptr(1700)

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		nil,
		[]contracts.Gateway{gateway},
		types.Options{EventBuffer: 256, GatewayAdapterFactory: factory},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected, Heartbeat: true})
	waitForEventType(t, engine.Events(), contracts.GatewayHeartbeat)

	payload := []byte("downlink")
	if err := engine.SendGatewayPacket(context.Background(), gateway.ID, payload); err != nil {
		t.Fatalf("send gateway packet: %v", err)
	}
	select {
	case packet := <-adapter.sends:
		if string(packet.Payload) != string(payload) {
			t.Fatalf("unexpected egress payload: %q", packet.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for fake gateway egress")
	}
	waitForEventType(t, engine.Events(), contracts.GatewayPacketEgress)

	adapter.packets <- types.GatewayPacket{GatewayID: gateway.ID, Payload: []byte("uplink"), RSSI: -80, SNR: 4.5}
	ingress := waitForEventType(t, engine.Events(), contracts.GatewayPacketIngress)
	if ingress.RSSI == nil || *ingress.RSSI != -80 || ingress.SNR == nil || *ingress.SNR != 4.5 {
		t.Fatalf("real gateway radio metadata was not propagated: %+v", ingress)
	}
	adapter.emit(types.GatewayAdapterEvent{
		GatewayID: gateway.ID,
		State:     contracts.Error,
		Error:     "heartbeat timeout",
		Timeout:   true,
	})
	waitForEventType(t, engine.Events(), contracts.GatewayNetworkError)

	snapshot := engine.Snapshot()
	if snapshot.Gateways[0].GatewayState != contracts.Error || snapshot.Gateways[0].LastNetworkError != "heartbeat timeout" {
		t.Fatalf("unexpected real gateway runtime: %+v", snapshot.Gateways[0])
	}
	if snapshot.Gateways[0].IngressPackets != 1 || snapshot.Gateways[0].EgressPackets != 1 {
		t.Fatalf("unexpected gateway packet counters: %+v", snapshot.Gateways[0])
	}
	if snapshot.Metrics.GatewayConnections != 1 || snapshot.Metrics.GatewayHeartbeats != 1 ||
		snapshot.Metrics.GatewayIngressPackets != 1 || snapshot.Metrics.GatewayEgressPackets != 1 || snapshot.Metrics.GatewayTimeouts != 1 {
		t.Fatalf("unexpected gateway metrics: %+v", snapshot.Metrics)
	}
}

func TestPhaseFiveEngineRequeuesDownlinkAfterGatewayFailure(t *testing.T) {
	adapter := newFakeGatewayAdapter()
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c45")
	gateway.Type = contracts.Real
	gateway.KeepAlive = nil
	gateway.GatewayEUI = "A840410001000145"
	gateway.GatewayIPv4 = engineStringPtr("127.0.0.1")
	gateway.GatewayPort = engineInt32Ptr(1745)

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		nil,
		[]contracts.Gateway{gateway},
		types.Options{EventBuffer: 256, GatewayAdapterFactory: &fakeGatewayFactory{adapter: adapter}},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	packet := types.GatewayPacket{
		GatewayID: gateway.ID,
		Payload:   []byte("retry after reconnect"),
	}
	if err := engine.SendGatewayPacket(context.Background(), gateway.ID, packet.Payload); err != nil {
		t.Fatalf("send gateway packet: %v", err)
	}
	select {
	case sent := <-adapter.sends:
		if string(sent.Payload) != string(packet.Payload) {
			t.Fatalf("unexpected initial downlink: %q", sent.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for initial downlink")
	}

	adapter.emit(types.GatewayAdapterEvent{
		GatewayID: gateway.ID,
		State:     contracts.Error,
		Error:     "gateway heartbeat timeout",
		Timeout:   true,
		Packet:    &packet,
	})
	waitForEventType(t, engine.Events(), contracts.GatewayNetworkError)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Reconnecting})
	waitForEventType(t, engine.Events(), contracts.GatewayReconnecting)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	select {
	case resent := <-adapter.sends:
		if string(resent.Payload) != string(packet.Payload) {
			t.Fatalf("unexpected requeued downlink: %q", resent.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for downlink resend after reconnect")
	}
}

func TestPhaseFiveRealUplinkUpdatesDeviceAndSchedulesConfirmedACK(t *testing.T) {
	adapter := newFakeGatewayAdapter()
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000041")
	configureABP(&device)
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c41")
	gateway.Type = contracts.Real
	gateway.KeepAlive = nil
	gateway.GatewayEUI = "A840410001000141"
	gateway.GatewayIPv4 = engineStringPtr("127.0.0.1")
	gateway.GatewayPort = engineInt32Ptr(1741)

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device},
		[]contracts.Gateway{gateway},
		types.Options{EventBuffer: 256, GatewayAdapterFactory: &fakeGatewayFactory{adapter: adapter}},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	nwkSKey, _ := lorawan.DecodeHex(device.ABPConfig.NwkSKey)
	appSKey, _ := lorawan.DecodeHex(device.ABPConfig.AppSKey)
	fPort := byte(device.FrameConfig.FPort)
	payload, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr:   0x26011bda,
		FCnt:      1,
		FPort:     &fPort,
		Payload:   []byte("uplink"),
		Confirmed: true,
		NwkSKey:   nwkSKey,
		AppSKey:   appSKey,
	})
	if err != nil {
		t.Fatalf("build real uplink: %v", err)
	}
	adapter.packets <- types.GatewayPacket{
		GatewayID: gateway.ID,
		Payload:   payload,
		Frequency: 868100000,
		Bandwidth: 125000,
		DataRate:  "SF7BW125",
	}
	waitForEventTypeWithLog(t, engine, contracts.UplinkACKReceived)
	if snapshot := engine.Snapshot(); snapshot.Devices[0].FrameCounterUp < 1 || snapshot.Metrics.SuccessfulUplinks != 1 {
		t.Fatalf("real uplink did not update runtime state: %+v", snapshot)
	}
	select {
	case ack := <-adapter.sends:
		decoded, parseErr := lorawan.Parse(ack.Payload)
		if parseErr != nil || decoded.MType != lorawan.MTypeUnconfirmedDataDown {
			t.Fatalf("unexpected real ACK frame: %+v", ack)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for real gateway ACK")
	}
}

func TestPhaseFourExternalGatewayDownlinkIsValidatedAndApplied(t *testing.T) {
	adapter := newFakeGatewayAdapter()
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000044")
	configureABP(&device)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c44")
	gateway.Type = contracts.Real
	gateway.KeepAlive = nil
	gateway.GatewayEUI = "A840410001000144"
	gateway.GatewayIPv4 = engineStringPtr("127.0.0.1")
	gateway.GatewayPort = engineInt32Ptr(1744)

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device}, []contracts.Gateway{gateway},
		types.Options{EventBuffer: 256, GatewayAdapterFactory: &fakeGatewayFactory{adapter: adapter}},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	nwkSKey, _ := lorawan.DecodeHex(device.ABPConfig.NwkSKey)
	appSKey, _ := lorawan.DecodeHex(device.ABPConfig.AppSKey)
	fPort := byte(1)
	payload, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr: 0x26011BDA, FCnt: 1, FPort: &fPort, Payload: []byte("command"),
		Direction: 1, NwkSKey: nwkSKey, AppSKey: appSKey,
	})
	if err != nil {
		t.Fatalf("build external downlink: %v", err)
	}
	adapter.packets <- types.GatewayPacket{
		GatewayID: gateway.ID, Kind: types.GatewayPacketDownlink, Payload: payload,
		Frequency: 869525000, Bandwidth: 125000, SpreadingFactor: 7, DataRate: "SF7BW125",
	}
	downlinkEvent := waitForEventTypeWithLog(t, engine, contracts.DeviceDownlinkTransmitted)
	if downlinkEvent.DeviceID == nil || *downlinkEvent.DeviceID != device.ID || downlinkEvent.PayloadSize == nil || *downlinkEvent.PayloadSize != 7 {
		t.Fatalf("unexpected external downlink event: %+v", downlinkEvent)
	}
	if engine.Snapshot().Devices[0].FrameCounterDown != 1 {
		t.Fatalf("external downlink did not update frame counter: %+v", engine.Snapshot().Devices[0])
	}
}

func TestRealGatewaysCanIngestTheSameDeviceThroughMultipleAdapters(t *testing.T) {
	first := newFakeGatewayAdapter()
	second := newFakeGatewayAdapter()
	firstGateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c51")
	secondGateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c52")
	for index, gateway := range []*contracts.Gateway{&firstGateway, &secondGateway} {
		gateway.Type = contracts.Real
		gateway.KeepAlive = nil
		gateway.GatewayEUI = "A84041000100015" + strconv.Itoa(index+1)
		gateway.GatewayIPv4 = engineStringPtr("127.0.0.1")
		port := int32(1751 + index)
		gateway.GatewayPort = &port
	}
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000051")
	configureABP(&device)
	factory := &multiFakeGatewayFactory{adapters: map[string]*fakeGatewayAdapter{
		firstGateway.ID: first, secondGateway.ID: second,
	}}
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{firstGateway, secondGateway}, types.Options{
		Clock: types.NewManualClock(0), EventBuffer: 256, GatewayAdapterFactory: factory,
	})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	first.emit(types.GatewayAdapterEvent{GatewayID: firstGateway.ID, State: contracts.Connected})
	second.emit(types.GatewayAdapterEvent{GatewayID: secondGateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)
	baseline := engine.Snapshot()
	deadline := time.After(time.Second)
	for baseline.Metrics.TotalUplinks < 1 {
		select {
		case <-deadline:
			t.Fatalf("initial device uplink was not processed: %+v", baseline)
		default:
			time.Sleep(time.Millisecond)
			baseline = engine.Snapshot()
		}
	}

	nwkSKey, _ := lorawan.DecodeHex(device.ABPConfig.NwkSKey)
	appSKey, _ := lorawan.DecodeHex(device.ABPConfig.AppSKey)
	fPort := byte(device.FrameConfig.FPort)
	build := func(counter uint32) []byte {
		frame, buildErr := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
			DevAddr: 0x26011bda, FCnt: counter, FPort: &fPort, Payload: []byte("gateway uplink"),
			NwkSKey: nwkSKey, AppSKey: appSKey,
		})
		if buildErr != nil {
			t.Fatalf("build uplink %d: %v", counter, buildErr)
		}
		return frame
	}
	firstCounter := uint32(baseline.Devices[0].FrameCounterUp + 1)
	secondCounter := firstCounter + 1
	first.packets <- types.GatewayPacket{GatewayID: firstGateway.ID, Payload: build(firstCounter), Bandwidth: 125_000, SpreadingFactor: 7}
	waitForEventType(t, engine.Events(), contracts.GatewayPacketIngress)
	second.packets <- types.GatewayPacket{GatewayID: secondGateway.ID, Payload: build(secondCounter), Bandwidth: 125_000, SpreadingFactor: 7}
	deadline = time.After(time.Second)
	var snapshot contracts.SimulationSnapshot
	for {
		snapshot = engine.Snapshot()
		if snapshot.Metrics.GatewayIngressPackets == baseline.Metrics.GatewayIngressPackets+2 &&
			snapshot.Devices[0].FrameCounterUp == int64(secondCounter) &&
			snapshot.Metrics.SuccessfulUplinks == baseline.Metrics.SuccessfulUplinks+2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("multiple real gateway packets were not ingested: %+v", snapshot)
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if snapshot.Metrics.GatewayIngressPackets != baseline.Metrics.GatewayIngressPackets+2 || snapshot.Devices[0].FrameCounterUp != int64(secondCounter) ||
		snapshot.Gateways[0].IngressPackets != 1 || snapshot.Gateways[1].IngressPackets != 1 {
		t.Fatalf("multiple real gateway uplinks were not processed: metrics=%+v device=%+v gateways=%+v", snapshot.Metrics, snapshot.Devices[0], snapshot.Gateways)
	}
}

func TestRealClassADownlinkCarriesACKFPendingAndMACCommands(t *testing.T) {
	adapter := newFakeGatewayAdapter()
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000043")
	configureABP(&device)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c43")
	gateway.Type = contracts.Real
	gateway.KeepAlive = nil
	gateway.GatewayEUI = "A840410001000143"
	gateway.GatewayIPv4 = engineStringPtr("127.0.0.1")
	gateway.GatewayPort = engineInt32Ptr(1743)
	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device},
		[]contracts.Gateway{gateway},
		types.Options{EventBuffer: 256, GatewayAdapterFactory: &fakeGatewayFactory{adapter: adapter}},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	dataRate, repetitions := 2, 3
	if _, err := engine.QueueDownlink(types.Downlink{
		DeviceID: device.ID, Payload: []byte("command"), Confirmed: true, DataRate: -1,
		MACCommands: []types.MACCommand{{Type: types.MACLinkADRReq, DataRate: &dataRate, NbTrans: &repetitions}},
	}); err != nil {
		t.Fatalf("queue first downlink: %v", err)
	}
	if _, err := engine.QueueDownlink(types.Downlink{DeviceID: device.ID, Payload: []byte("second"), DataRate: -1}); err != nil {
		t.Fatalf("queue second downlink: %v", err)
	}

	nwkSKey, _ := lorawan.DecodeHex(device.ABPConfig.NwkSKey)
	appSKey, _ := lorawan.DecodeHex(device.ABPConfig.AppSKey)
	fPort := byte(device.FrameConfig.FPort)
	uplink, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr: 0x26011bda, FCnt: 1, FPort: &fPort, Payload: []byte("uplink"),
		Confirmed: true, NwkSKey: nwkSKey, AppSKey: appSKey,
	})
	if err != nil {
		t.Fatalf("build uplink: %v", err)
	}
	adapter.packets <- types.GatewayPacket{
		GatewayID: gateway.ID, Payload: uplink, Frequency: 868_100_000,
		Bandwidth: 125_000, SpreadingFactor: 7, DataRate: "SF7BW125",
	}
	waitForEventTypeWithLog(t, engine, contracts.DeviceDownlinkTransmitted)
	select {
	case sent := <-adapter.sends:
		packet, parseErr := lorawan.Parse(sent.Payload)
		if parseErr != nil {
			t.Fatalf("parse downlink: %v", parseErr)
		}
		if !packet.ACK || !packet.Confirmed || !packet.FPending || len(packet.FOpts) != 5 || packet.FOpts[0] != 0x03 {
			t.Fatalf("unexpected Class A downlink controls: %+v FOpts=%x", packet, packet.FOpts)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for real Class A downlink")
	}
	engine.mu.RLock()
	session := *engine.sessions[device.ID]
	engine.mu.RUnlock()
	if session.CurrentDataRate != dataRate || session.UnconfirmedRepetitions != repetitions {
		t.Fatalf("MAC command was not applied: %+v", session)
	}
}

func TestPhaseFiveRealOTAAJoinQueuesEncryptedJoinAccept(t *testing.T) {
	adapter := newFakeGatewayAdapter()
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000042")
	device.RX1Config.Delay = intPtr(1000)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c42")
	gateway.Type = contracts.Real
	gateway.KeepAlive = nil
	gateway.GatewayEUI = "A840410001000142"
	gateway.GatewayIPv4 = engineStringPtr("127.0.0.1")
	gateway.GatewayPort = engineInt32Ptr(1742)

	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device},
		[]contracts.Gateway{gateway},
		types.Options{EventBuffer: 256, GatewayAdapterFactory: &fakeGatewayFactory{adapter: adapter}},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	waitForEventType(t, engine.Events(), contracts.GatewayConnecting)
	adapter.emit(types.GatewayAdapterEvent{GatewayID: gateway.ID, State: contracts.Connected})
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	joinEUI, _ := strconv.ParseUint(device.OOTAConfig.JoinEUI, 16, 64)
	devEUI, _ := strconv.ParseUint(device.DevEUI, 16, 64)
	appKey, _ := lorawan.DecodeHex(device.OOTAConfig.AppKey)
	joinRequest, err := lorawan.BuildJoinRequest(lorawan.JoinRequestOptions{JoinEUI: joinEUI, DevEUI: devEUI, DevNonce: 7, AppKey: appKey})
	if err != nil {
		t.Fatalf("build real join request: %v", err)
	}
	adapter.packets <- types.GatewayPacket{GatewayID: gateway.ID, Payload: joinRequest, Frequency: 868100000, Bandwidth: 125000, DataRate: "SF7BW125"}
	waitForEventType(t, engine.Events(), contracts.JoinAcceptReceived)

	select {
	case packet := <-adapter.sends:
		if packet.Kind != types.GatewayPacketDownlink || len(packet.Payload) != 17 || packet.Payload[0] != lorawan.MTypeJoinAccept<<5 {
			t.Fatalf("unexpected real Join-Accept packet: %+v", packet)
		}
		if packet.TransmitAt.IsZero() {
			t.Fatal("Join-Accept packet is not scheduled for RX1")
		}
		joinAccept, decodeErr := lorawan.DecodeJoinAccept(packet.Payload, appKey)
		if decodeErr != nil || joinAccept.RXDelay != 1 {
			t.Fatalf("Join-Accept did not encode the 1000 ms RX1 delay as one second: %+v error=%v", joinAccept, decodeErr)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for real Join-Accept")
	}
}

func TestVirtualGatewayBridgeJoinAcceptCompletesExternalOTAAJoin(t *testing.T) {
	adapter := newVirtualUplinkTestAdapter()
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000046")
	device.RX1Config.Delay = intPtr(1000)
	device.RX2Config.Delay = intPtr(1000)
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c46")
	clock := types.NewManualClock(0)
	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		[]contracts.Device{device},
		[]contracts.Gateway{gateway},
		types.Options{Clock: clock, EventBuffer: 256, GatewayAdapterFactory: &virtualUplinkTestFactory{adapter: adapter}},
	)
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	var joinAt time.Duration
	foundJoin := false
	for _, scheduled := range engine.Checkpoint().Scheduled {
		if scheduled.Kind == types.ScheduledEventJoinRequest {
			joinAt = scheduled.At
			foundJoin = true
			break
		}
	}
	if !foundJoin {
		t.Fatal("engine did not schedule an OTAA join request")
	}
	if err := clock.Advance(joinAt + 500*time.Millisecond); err != nil {
		t.Fatalf("advance clock to the gateway uplink: %v", err)
	}

	var joinRequest types.GatewayPacket
	select {
	case joinRequest = <-adapter.uplinks:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the virtual gateway join request")
	}
	decodedRequest, err := lorawan.Parse(joinRequest.Payload)
	if err != nil || decodedRequest.MType != lorawan.MTypeJoinRequest {
		t.Fatalf("unexpected bridge join request: packet=%+v error=%v", decodedRequest, err)
	}
	if err := clock.Advance(time.Second); err != nil {
		t.Fatalf("advance clock through the external RX1 window: %v", err)
	}
	waitForEventType(t, engine.Events(), contracts.RX1WindowOpened)
	if engine.Snapshot().Devices[0].Joined {
		t.Fatal("device joined without receiving a Join-Accept from the external network server")
	}
	if err := clock.Advance(11 * time.Second); err != nil {
		t.Fatalf("advance clock to the next external join attempt: %v", err)
	}
	var retryRequest types.GatewayPacket
	select {
	case retryRequest = <-adapter.uplinks:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the retried virtual gateway join request")
	}
	decodedRetry, err := lorawan.Parse(retryRequest.Payload)
	if err != nil || decodedRetry.MType != lorawan.MTypeJoinRequest {
		t.Fatalf("unexpected retried bridge join request: packet=%+v error=%v", decodedRetry, err)
	}
	if decodedRetry.DevNonce == decodedRequest.DevNonce {
		t.Fatalf("retried Join-Request reused DevNonce %d", decodedRetry.DevNonce)
	}
	decodedRequest = decodedRetry
	appKey, err := lorawan.DecodeHex(device.OOTAConfig.AppKey)
	if err != nil {
		t.Fatalf("decode AppKey: %v", err)
	}
	joinAccept, err := lorawan.BuildJoinAccept(lorawan.JoinAcceptOptions{
		AppNonce:    0x010203,
		NetID:       0x000001,
		DevAddr:     0x26011BDA,
		RX2DataRate: 5,
		AppKey:      appKey,
	})
	if err != nil {
		t.Fatalf("build external Join-Accept: %v", err)
	}
	requestTime := engine.Checkpoint().Sessions[device.ID].PendingJoinRequest.RequestedAt
	adapter.packets <- types.GatewayPacket{
		GatewayID:  gateway.ID,
		Kind:       types.GatewayPacketDownlink,
		Payload:    joinAccept,
		TransmitAt: requestTime.Add(-time.Second),
	}
	waitForEventType(t, engine.Events(), contracts.DeviceDownlinkDropped)
	if engine.Snapshot().Devices[0].Joined {
		t.Fatal("a delayed Join-Accept from an older attempt completed the current join")
	}
	adapter.packets <- types.GatewayPacket{
		GatewayID:  gateway.ID,
		Kind:       types.GatewayPacketDownlink,
		Payload:    joinAccept,
		TransmitAt: requestTime.Add(5 * time.Second),
	}
	joinEvent := waitForEventType(t, engine.Events(), contracts.JoinAcceptReceived)
	if !strings.Contains(joinEvent.Message, "external network server") {
		t.Fatalf("join was not completed by the external network server: %+v", joinEvent)
	}

	checkpoint := engine.Checkpoint()
	session := checkpoint.Sessions[device.ID]
	if !session.Joined || session.PendingJoinRequest != nil || session.DeviceAddress != "26011BDA" {
		t.Fatalf("external Join-Accept did not complete the session: %+v", session)
	}
	if !engine.Snapshot().Devices[0].Joined {
		t.Fatal("external Join-Accept did not update the device status to joined")
	}
	expectedNwk, expectedApp, err := lorawan.DeriveSessionKeys(appKey, 0x010203, 0x000001, decodedRequest.DevNonce)
	if err != nil {
		t.Fatalf("derive expected session keys: %v", err)
	}
	if string(session.NwkSKey) != string(expectedNwk) || string(session.AppSKey) != string(expectedApp) {
		t.Fatal("external Join-Accept did not install the network-server session keys")
	}
	if err := clock.Advance(11 * time.Second); err != nil {
		t.Fatalf("advance clock to the first post-join uplink: %v", err)
	}
	select {
	case uplink := <-adapter.uplinks:
		decodedUplink, parseErr := lorawan.Parse(uplink.Payload)
		if parseErr != nil || decodedUplink.MType != lorawan.MTypeUnconfirmedDataUp || decodedUplink.DevAddr != 0x26011BDA {
			t.Fatalf("unexpected first post-join uplink: packet=%+v decoded=%+v error=%v", uplink, decodedUplink, parseErr)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the first post-join uplink")
	}
	fPort := byte(1)
	downlink, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr:   0x26011BDA,
		FCnt:      0,
		FPort:     &fPort,
		Payload:   []byte("server command"),
		Direction: 1,
		NwkSKey:   expectedNwk,
		AppSKey:   expectedApp,
	})
	if err != nil {
		t.Fatalf("build first external downlink: %v", err)
	}
	adapter.packets <- types.GatewayPacket{
		GatewayID: gateway.ID,
		Kind:      types.GatewayPacketDownlink,
		Payload:   downlink,
	}
	downlinkEvent := waitForEventType(t, engine.Events(), contracts.DeviceDownlinkTransmitted)
	if downlinkEvent.DeviceID == nil || *downlinkEvent.DeviceID != device.ID {
		t.Fatalf("first external downlink was not applied to the joined device: %+v", downlinkEvent)
	}
	adapter.packets <- types.GatewayPacket{
		GatewayID: gateway.ID,
		Kind:      types.GatewayPacketDownlink,
		Payload:   downlink,
	}
	waitForEventType(t, engine.Events(), contracts.FrameCounterRejected)
	if got := engine.Snapshot().Devices[0].FrameCounterDown; got != 0 {
		t.Fatalf("duplicate external downlink changed the frame counter: %d", got)
	}
}

func TestExternalJoinAcceptRejectsInvalidMICAndKeepsDeviceJoining(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000047")
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c47")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	session := engine.sessions[device.ID]
	session.PendingJoinRequest = &types.PendingJoinRequest{PacketID: "join-request", GatewayIDs: []string{gateway.ID}}

	joinAccept, err := lorawan.BuildJoinAccept(lorawan.JoinAcceptOptions{
		AppNonce: 0x010203,
		NetID:    0x000001,
		DevAddr:  0x26011BDA,
		AppKey:   mustAppKey(device),
	})
	if err != nil {
		t.Fatalf("build Join-Accept: %v", err)
	}
	joinAccept[len(joinAccept)-1] ^= 0xff
	decoded, err := lorawan.Parse(joinAccept)
	if err != nil {
		t.Fatalf("parse invalid Join-Accept: %v", err)
	}

	engine.mu.Lock()
	events := engine.processExternalJoinAcceptLocked(types.GatewayPacket{
		GatewayID: gateway.ID,
		Kind:      types.GatewayPacketDownlink,
		Payload:   joinAccept,
	}, decoded)
	engine.mu.Unlock()

	if len(events) != 1 || events[0].Type != contracts.DeviceDownlinkDropped {
		t.Fatalf("invalid Join-Accept was not rejected: %+v", events)
	}
	if session.Joined || session.PendingJoinRequest == nil {
		t.Fatalf("invalid Join-Accept changed the pending join state: %+v", session)
	}
}

func TestExternalDownlinkACKCompletesConfirmedUplink(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000049")
	configureABP(&device)
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c49")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	session := engine.sessions[device.ID]
	session.PendingUplink = &types.PendingUplink{
		PacketID: "confirmed-uplink", FrameCounter: 1, Attempt: 1, Confirmed: true,
		TransmissionAt: time.Second, FragmentCount: 1, GatewayIDs: []string{gateway.ID},
	}
	engine.metrics.TotalUplinks = 1
	downlink, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr: 0x26011BDA, FCnt: 0, ACK: true, Direction: 1,
		NwkSKey: session.NwkSKey, AppSKey: session.AppSKey,
	})
	if err != nil {
		t.Fatalf("build external ACK: %v", err)
	}
	decoded, err := lorawan.Parse(downlink)
	if err != nil {
		t.Fatalf("parse external ACK: %v", err)
	}

	engine.mu.Lock()
	events := engine.processExternalDownlinkLocked(types.GatewayPacket{
		GatewayID: gateway.ID, Kind: types.GatewayPacketDownlink, Payload: downlink,
	}, decoded)
	engine.mu.Unlock()

	if !containsEventType(events, contracts.UplinkACKReceived) || !containsEventType(events, contracts.MetricsUpdated) {
		t.Fatalf("external ACK did not complete the confirmed uplink: %+v", events)
	}
	if session.PendingUplink != nil {
		t.Fatalf("external ACK left the confirmed uplink pending: %+v", session.PendingUplink)
	}
	if engine.metrics.AcknowledgedUplinks != 1 || engine.metrics.SuccessfulUplinks != 1 || engine.metrics.PacketSuccessRate != 1 {
		t.Fatalf("external ACK did not update metrics: %+v", engine.metrics)
	}
}

func TestExternalJoinAcceptFromUnexpectedGatewayIsRejected(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000048")
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c48")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	session := engine.sessions[device.ID]
	session.PendingJoinRequest = &types.PendingJoinRequest{PacketID: "join-request", GatewayIDs: []string{gateway.ID}}
	joinAccept, err := lorawan.BuildJoinAccept(lorawan.JoinAcceptOptions{
		AppNonce: 0x010203,
		NetID:    0x000001,
		DevAddr:  0x26011BDA,
		AppKey:   mustAppKey(device),
	})
	if err != nil {
		t.Fatalf("build Join-Accept: %v", err)
	}
	decoded, err := lorawan.Parse(joinAccept)
	if err != nil {
		t.Fatalf("parse Join-Accept: %v", err)
	}

	engine.mu.Lock()
	events := engine.processExternalJoinAcceptLocked(types.GatewayPacket{
		GatewayID: "unexpected-gateway",
		Kind:      types.GatewayPacketDownlink,
		Payload:   joinAccept,
	}, decoded)
	engine.mu.Unlock()

	if len(events) != 1 || events[0].Type != contracts.DeviceDownlinkDropped {
		t.Fatalf("Join-Accept from an unexpected gateway was not rejected: %+v", events)
	}
	if session.Joined || session.PendingJoinRequest == nil {
		t.Fatalf("unexpected gateway changed the pending join state: %+v", session)
	}
}

func TestStaleExternalJoinReceiveWindowIsIgnoredAfterJoin(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000049")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, nil, types.Options{})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	session := engine.sessions[device.ID]
	session.Joined = true
	session.PendingJoinRequest = nil

	engine.mu.Lock()
	events := engine.processReceiveWindowLocked(types.ScheduledEvent{
		DeviceID:     device.ID,
		Kind:         types.ScheduledEventRX1Window,
		Window:       contracts.RX1,
		FrameCounter: -1,
	})
	engine.mu.Unlock()

	if len(events) != 0 {
		t.Fatalf("stale join receive window produced events after the session joined: %+v", events)
	}
}

type fakeGatewayFactory struct {
	adapter *fakeGatewayAdapter
}

func (factory *fakeGatewayFactory) NewGatewayAdapter(contracts.Gateway) (types.GatewayAdapter, error) {
	return factory.adapter, nil
}

type multiFakeGatewayFactory struct {
	adapters map[string]*fakeGatewayAdapter
}

func (factory *multiFakeGatewayFactory) NewGatewayAdapter(gateway contracts.Gateway) (types.GatewayAdapter, error) {
	adapter, ok := factory.adapters[gateway.ID]
	if !ok {
		return nil, strconv.ErrSyntax
	}
	return adapter, nil
}

type fakeGatewayAdapter struct {
	events  chan types.GatewayAdapterEvent
	packets chan types.GatewayPacket
	sends   chan types.GatewayPacket
	closed  chan struct{}
}

func newFakeGatewayAdapter() *fakeGatewayAdapter {
	return &fakeGatewayAdapter{
		events:  make(chan types.GatewayAdapterEvent, 32),
		packets: make(chan types.GatewayPacket, 32),
		sends:   make(chan types.GatewayPacket, 32),
		closed:  make(chan struct{}),
	}
}

func (adapter *fakeGatewayAdapter) Start(context.Context) error { return nil }

func (adapter *fakeGatewayAdapter) Send(_ context.Context, packet types.GatewayPacket) error {
	adapter.sends <- packet
	return nil
}

func (adapter *fakeGatewayAdapter) Packets() <-chan types.GatewayPacket {
	return adapter.packets
}

func (adapter *fakeGatewayAdapter) Events() <-chan types.GatewayAdapterEvent {
	return adapter.events
}

func (adapter *fakeGatewayAdapter) Close() error {
	select {
	case <-adapter.closed:
	default:
		close(adapter.closed)
		close(adapter.events)
		close(adapter.packets)
	}
	return nil
}

func (adapter *fakeGatewayAdapter) emit(event types.GatewayAdapterEvent) {
	adapter.events <- event
}

func engineStringPtr(value string) *string { return &value }

func engineInt32Ptr(value int32) *int32 { return &value }

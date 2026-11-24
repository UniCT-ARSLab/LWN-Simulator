package gateway

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestUDPAdapterImplementsSemtechIngressAndEgress(t *testing.T) {
	gatewaySocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway: %v", err)
	}
	defer gatewaySocket.Close()

	gateway := testRealGateway(gatewaySocket.LocalAddr().(*net.UDPAddr).Port)
	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: time.Second})
	adapter, err := factory.NewGatewayAdapter(gateway)
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer adapter.Close()

	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start adapter: %v", err)
	}
	factory.mu.Lock()
	serverAddress := factory.runtime.conn.LocalAddr().(*net.UDPAddr)
	factory.mu.Unlock()

	token := [2]byte{0x12, 0x34}
	eui, _ := hexBytes(gateway.GatewayEUI)
	pullData := append(packetHeader(token, pullDataType), eui...)
	if _, err := gatewaySocket.WriteToUDP(pullData, serverAddress); err != nil {
		t.Fatalf("send PULL_DATA: %v", err)
	}
	readSemtechPacket(t, gatewaySocket, pullAckType)
	waitAdapterEvent(t, adapter.Events(), contracts.Connected)
	waitHeartbeat(t, adapter.Events())

	counter := uint32(491697203)
	body, err := json.Marshal(pushDataPayload{RXPK: []rxPacket{{
		Timestamp: &counter,
		Data:      base64.StdEncoding.EncodeToString([]byte("uplink")), Frequency: 868.3, DataRate: "SF7BW125",
		Modulation: "LORA", CodingRate: "4/5", RSSI: -72, SNR: 5.5, Size: 6,
	}}})
	if err != nil {
		t.Fatalf("encode PUSH_DATA: %v", err)
	}
	pushData := append(packetHeader([2]byte{0x22, 0x33}, pushDataType), eui...)
	pushData = append(pushData, body...)
	if _, err := gatewaySocket.WriteToUDP(pushData, serverAddress); err != nil {
		t.Fatalf("send PUSH_DATA: %v", err)
	}
	readSemtechPacket(t, gatewaySocket, pushAckType)
	var receivedPacket types.GatewayPacket
	select {
	case packet := <-adapter.Packets():
		receivedPacket = packet
		if packet.ConcentratorTimestamp == nil || *packet.ConcentratorTimestamp != counter {
			t.Fatal("received concentrator timestamp was lost")
		}
		if string(packet.Payload) != "uplink" || packet.Frequency != 868300000 || packet.RSSI != -72 || packet.SNR != 5.5 {
			t.Fatalf("unexpected ingress packet: %+v", packet)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for ingress packet")
	}

	downlinkBody := []byte(`{"txpk":{"imme":true,"freq":868.1,"modu":"LORA","datr":"SF7BW125","size":8,"data":"ZG93bmxpbms="}}`)
	pullResponse := append(packetHeader([2]byte{0x44, 0x55}, pullResponseType), downlinkBody...)
	if _, err := gatewaySocket.WriteToUDP(pullResponse, serverAddress); err != nil {
		t.Fatalf("send PULL_RESP: %v", err)
	}
	ack := readSemtechPacket(t, gatewaySocket, txAckType)
	if len(ack) < 12 || !equalBytes(ack[4:12], eui) {
		t.Fatalf("TX_ACK did not contain the configured gateway EUI: %x", ack)
	}
	var ackBody txAckPayload
	if err := json.Unmarshal(ack[12:], &ackBody); err != nil || ackBody.TXPKAck.Error != "NONE" {
		t.Fatalf("unexpected PULL_RESP acknowledgement: %s (%v)", ack[12:], err)
	}
	select {
	case packet := <-adapter.Packets():
		if packet.Kind != types.GatewayPacketDownlink || string(packet.Payload) != "downlink" || packet.Frequency != 868100000 {
			t.Fatalf("unexpected downlink packet: %+v", packet)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for downlink packet")
	}

	if err := adapter.Send(ctx, types.GatewayPacket{Payload: []byte("downlink")}); err != nil {
		t.Fatalf("send PULL_RESP: %v", err)
	}
	response := readSemtechPacket(t, gatewaySocket, pullResponseType)
	var responseBody struct {
		TXPK txPacket `json:"txpk"`
	}
	if err := json.Unmarshal(response[4:], &responseBody); err != nil {
		t.Fatalf("decode PULL_RESP: %v", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(responseBody.TXPK.Data)
	if err != nil || string(decoded) != "downlink" {
		t.Fatalf("unexpected egress payload: %+v", responseBody.TXPK)
	}

	target := time.Now().Add(time.Second).UTC().Truncate(time.Millisecond)
	if err := adapter.Send(ctx, types.GatewayPacket{Payload: []byte("timed-downlink"), TransmitAt: target}); err != nil {
		t.Fatalf("send timed PULL_RESP: %v", err)
	}
	timedResponse := readSemtechPacket(t, gatewaySocket, pullResponseType)
	var timedBody struct {
		TXPK txPacket `json:"txpk"`
	}
	if err := json.Unmarshal(timedResponse[4:], &timedBody); err != nil {
		t.Fatalf("decode timed PULL_RESP: %v", err)
	}
	wantTimestamp := counter + uint32(target.Sub(receivedPacket.ReceivedAt).Microseconds())
	if timedBody.TXPK.Immediate || timedBody.TXPK.Time != "" || timedBody.TXPK.Timestamp == nil || *timedBody.TXPK.Timestamp != wantTimestamp {
		t.Fatalf("timed transmission did not use the concentrator clock: %+v want=%d", timedBody.TXPK, wantTimestamp)
	}
	txAckBody := []byte(`{"txpk_ack":{"error":"TOO_LATE"}}`)
	// Semtech TX_ACK includes the gateway EUI before its optional JSON body.
	txAck := append([]byte{protocolVersion, timedResponse[1], timedResponse[2], txAckType}, eui...)
	txAck = append(txAck, txAckBody...)
	if _, err := gatewaySocket.WriteToUDP(txAck, serverAddress); err != nil {
		t.Fatalf("send TX_ACK: %v", err)
	}
	deadline := time.After(time.Second)
	for {
		select {
		case event := <-adapter.Events():
			if event.State != contracts.Error {
				continue
			}
			if event.Packet == nil || string(event.Packet.Payload) != "timed-downlink" {
				t.Fatalf("expected correlated TX_ACK error, got %+v", event)
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for correlated TX_ACK error")
		}
	}
}

func TestUDPAdapterAcceptsSemtechGatewayAcknowledgements(t *testing.T) {
	gatewaySocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway: %v", err)
	}
	defer gatewaySocket.Close()

	gateway := testRealGateway(gatewaySocket.LocalAddr().(*net.UDPAddr).Port)
	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: time.Second})
	adapter, err := factory.NewGatewayAdapter(gateway)
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer adapter.Close()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start adapter: %v", err)
	}
	rawAdapter := adapter.(*udpAdapter)
	factory.mu.Lock()
	serverAddress := factory.runtime.conn.LocalAddr().(*net.UDPAddr)
	factory.mu.Unlock()
	eui, _ := hexBytes(gateway.GatewayEUI)

	pullToken := [2]byte{0x20, 0x21}
	if _, err := gatewaySocket.WriteToUDP(append(packetHeader(pullToken, pullAckType), nil...), serverAddress); err != nil {
		t.Fatalf("send PULL_ACK: %v", err)
	}
	waitHeartbeat(t, adapter.Events())

	pushToken := [2]byte{0x22, 0x23}
	pushAck := append(packetHeader(pushToken, pushAckType), nil...)
	if _, err := gatewaySocket.WriteToUDP(pushAck, serverAddress); err != nil {
		t.Fatalf("send PUSH_ACK: %v", err)
	}
	waitHeartbeat(t, adapter.Events())

	emptyTXAck := append(packetHeader([2]byte{0x24, 0x25}, txAckType), eui...)
	if err := rawAdapter.handleDatagram(ctx, udpDatagram{data: emptyTXAck, remote: gatewaySocket.LocalAddr().(*net.UDPAddr)}); err != nil {
		t.Fatalf("empty TX_ACK should be accepted: %v", err)
	}
}

func TestUDPAdapterRejectsUnsupportedOrInconsistentRXPK(t *testing.T) {
	factory := NewUDPFactory(UDPOptions{})
	adapter, err := factory.NewGatewayAdapter(testRealGateway(1700))
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	defer adapter.Close()
	rawAdapter := adapter.(*udpAdapter)
	ctx := context.Background()

	unsupported, err := json.Marshal(pushDataPayload{RXPK: []rxPacket{{
		Data: base64.StdEncoding.EncodeToString([]byte("uplink")), Frequency: 868.1,
		DataRate: "SF7BW125", Modulation: "FSK", Size: 6,
	}}})
	if err != nil {
		t.Fatalf("encode unsupported RXPK: %v", err)
	}
	if err := rawAdapter.handlePushData(ctx, unsupported); err == nil {
		t.Fatal("unsupported modulation should be rejected")
	}

	mismatch, err := json.Marshal(pushDataPayload{RXPK: []rxPacket{{
		Data: base64.StdEncoding.EncodeToString([]byte("uplink")), Frequency: 868.1,
		DataRate: "SF7BW125", Modulation: "LORA", Size: 99,
	}}})
	if err != nil {
		t.Fatalf("encode inconsistent RXPK: %v", err)
	}
	if err := rawAdapter.handlePushData(ctx, mismatch); err == nil {
		t.Fatal("inconsistent RXPK size should be rejected")
	}

	batch := []byte(`{"rxpk":[null,{"modu":"FSK","datr":50000,"data":"AA=="},
		{"stat":-1,"modu":"LORA","data":"AA=="},{"stat":1,"modu":"LORA","datr":"SF7BW125","data":"AQI="}]}`)
	if err := rawAdapter.handlePushData(ctx, batch); err == nil {
		t.Fatal("invalid packets in the batch should be reported")
	}
	select {
	case packet := <-adapter.Packets():
		if !equalBytes(packet.Payload, []byte{1, 2}) {
			t.Fatalf("unexpected packet accepted from mixed batch: %x", packet.Payload)
		}
	default:
		t.Fatal("the valid LoRa packet was lost")
	}
	select {
	case packet := <-adapter.Packets():
		t.Fatalf("another packet was accepted from the mixed batch: %x", packet.Payload)
	default:
	}
}

func TestUDPAdapterRejectsTXAckWithMissingGatewayEUI(t *testing.T) {
	gatewaySocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway: %v", err)
	}
	defer gatewaySocket.Close()

	gateway := testRealGateway(gatewaySocket.LocalAddr().(*net.UDPAddr).Port)
	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: time.Second})
	adapter, err := factory.NewGatewayAdapter(gateway)
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer adapter.Close()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start adapter: %v", err)
	}
	udpAdapter := adapter.(*udpAdapter)

	malformed := append(packetHeader([2]byte{0x30, 0x31}, txAckType), []byte(`{"txpk_ack":{"error":"NONE"}}`)...)
	if err := udpAdapter.handleDatagram(ctx, udpDatagram{data: malformed, remote: gatewaySocket.LocalAddr().(*net.UDPAddr)}); err == nil {
		t.Fatal("TX_ACK without the gateway EUI should be rejected")
	}
}

func TestUDPAdapterReportsTimeoutAndReconnectState(t *testing.T) {
	gatewaySocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway: %v", err)
	}
	defer gatewaySocket.Close()

	gateway := testRealGateway(gatewaySocket.LocalAddr().(*net.UDPAddr).Port)
	factory := NewUDPFactory(UDPOptions{
		LocalAddress:     "127.0.0.1:0",
		Timeout:          50 * time.Millisecond,
		ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax:     20 * time.Millisecond,
	})
	adapter, err := factory.NewGatewayAdapter(gateway)
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer adapter.Close()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start adapter: %v", err)
	}
	factory.mu.Lock()
	serverAddress := factory.runtime.conn.LocalAddr().(*net.UDPAddr)
	factory.mu.Unlock()
	eui, _ := hexBytes(gateway.GatewayEUI)
	if _, err := gatewaySocket.WriteToUDP(append(packetHeader([2]byte{1, 2}, pullDataType), eui...), serverAddress); err != nil {
		t.Fatalf("send PULL_DATA: %v", err)
	}
	readSemtechPacket(t, gatewaySocket, pullAckType)
	waitAdapterEvent(t, adapter.Events(), contracts.Connected)
	waitHeartbeat(t, adapter.Events())
	waitAdapterEvent(t, adapter.Events(), contracts.Error)
	waitAdapterEvent(t, adapter.Events(), contracts.Disconnected)
	waitAdapterEvent(t, adapter.Events(), contracts.Reconnecting)
}

func TestUDPAdapterRecoversAfterHeartbeatTimeout(t *testing.T) {
	gatewaySocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway: %v", err)
	}
	defer gatewaySocket.Close()

	gateway := testRealGateway(gatewaySocket.LocalAddr().(*net.UDPAddr).Port)
	factory := NewUDPFactory(UDPOptions{
		LocalAddress:     "127.0.0.1:0",
		Timeout:          50 * time.Millisecond,
		ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax:     20 * time.Millisecond,
	})
	adapter, err := factory.NewGatewayAdapter(gateway)
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer adapter.Close()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start adapter: %v", err)
	}
	eui, _ := hexBytes(gateway.GatewayEUI)
	sendPullData := func(address *net.UDPAddr, token [2]byte) {
		t.Helper()
		if _, writeErr := gatewaySocket.WriteToUDP(append(packetHeader(token, pullDataType), eui...), address); writeErr != nil {
			t.Fatalf("send PULL_DATA: %v", writeErr)
		}
	}
	factory.mu.Lock()
	serverAddress := cloneUDPAddr(factory.runtime.conn.LocalAddr().(*net.UDPAddr))
	factory.mu.Unlock()
	sendPullData(serverAddress, [2]byte{0x01, 0x02})
	readSemtechPacket(t, gatewaySocket, pullAckType)
	waitAdapterEvent(t, adapter.Events(), contracts.Connected)
	waitHeartbeat(t, adapter.Events())
	waitAdapterEvent(t, adapter.Events(), contracts.Error)
	waitAdapterEvent(t, adapter.Events(), contracts.Disconnected)
	waitAdapterEvent(t, adapter.Events(), contracts.Reconnecting)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		factory.mu.Lock()
		if factory.runtime != nil {
			serverAddress = cloneUDPAddr(factory.runtime.conn.LocalAddr().(*net.UDPAddr))
		}
		factory.mu.Unlock()
		if serverAddress != nil {
			sendPullData(serverAddress, [2]byte{0x03, 0x04})
			if packet, readErr := readSemtechPacketWithDeadline(gatewaySocket, pullAckType, 50*time.Millisecond); readErr == nil && len(packet) == 4 {
				waitAdapterEvent(t, adapter.Events(), contracts.Connected)
				waitHeartbeat(t, adapter.Events())
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("gateway adapter did not recover after heartbeat timeout")
}

func TestUDPFactoryRoutesMultipleRealGatewaysThroughSharedListener(t *testing.T) {
	firstSocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen first fake gateway: %v", err)
	}
	defer firstSocket.Close()
	secondSocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen second fake gateway: %v", err)
	}
	defer secondSocket.Close()

	firstGateway := testRealGateway(firstSocket.LocalAddr().(*net.UDPAddr).Port)
	firstGateway.ID = "6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c61"
	firstGateway.GatewayEUI = "A840410001000161"
	secondGateway := testRealGateway(secondSocket.LocalAddr().(*net.UDPAddr).Port)
	secondGateway.ID = "6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c62"
	secondGateway.GatewayEUI = "A840410001000162"
	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: time.Second})
	firstAdapter, err := factory.NewGatewayAdapter(firstGateway)
	if err != nil {
		t.Fatalf("create first adapter: %v", err)
	}
	secondAdapter, err := factory.NewGatewayAdapter(secondGateway)
	if err != nil {
		t.Fatalf("create second adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer firstAdapter.Close()
	defer secondAdapter.Close()
	if err := firstAdapter.Start(ctx); err != nil {
		t.Fatalf("start first adapter: %v", err)
	}
	if err := secondAdapter.Start(ctx); err != nil {
		t.Fatalf("start second adapter: %v", err)
	}
	factory.mu.Lock()
	serverAddress := cloneUDPAddr(factory.runtime.conn.LocalAddr().(*net.UDPAddr))
	factory.mu.Unlock()

	firstEUI, _ := hexBytes(firstGateway.GatewayEUI)
	secondEUI, _ := hexBytes(secondGateway.GatewayEUI)
	if _, err := firstSocket.WriteToUDP(append(packetHeader([2]byte{0x10, 0x11}, pullDataType), firstEUI...), serverAddress); err != nil {
		t.Fatalf("send first PULL_DATA: %v", err)
	}
	if _, err := secondSocket.WriteToUDP(append(packetHeader([2]byte{0x12, 0x13}, pullDataType), secondEUI...), serverAddress); err != nil {
		t.Fatalf("send second PULL_DATA: %v", err)
	}
	readSemtechPacket(t, firstSocket, pullAckType)
	readSemtechPacket(t, secondSocket, pullAckType)
	waitAdapterEvent(t, firstAdapter.Events(), contracts.Connected)
	waitAdapterEvent(t, secondAdapter.Events(), contracts.Connected)

	sendUplink := func(socket *net.UDPConn, eui []byte, token [2]byte, payload string) {
		t.Helper()
		body, marshalErr := json.Marshal(pushDataPayload{RXPK: []rxPacket{{
			Data: base64.StdEncoding.EncodeToString([]byte(payload)), Frequency: 868.1,
			DataRate: "SF7BW125", Modulation: "LORA", Size: len(payload),
		}}})
		if marshalErr != nil {
			t.Fatalf("encode %s PUSH_DATA: %v", payload, marshalErr)
		}
		packet := append(packetHeader(token, pushDataType), eui...)
		packet = append(packet, body...)
		if _, writeErr := socket.WriteToUDP(packet, serverAddress); writeErr != nil {
			t.Fatalf("send %s PUSH_DATA: %v", payload, writeErr)
		}
	}
	sendUplink(firstSocket, firstEUI, [2]byte{0x14, 0x15}, "first")
	sendUplink(secondSocket, secondEUI, [2]byte{0x16, 0x17}, "second")
	readSemtechPacket(t, firstSocket, pushAckType)
	readSemtechPacket(t, secondSocket, pushAckType)
	select {
	case packet := <-firstAdapter.Packets():
		if packet.GatewayID != firstGateway.ID || string(packet.Payload) != "first" {
			t.Fatalf("first gateway received the wrong packet: %+v", packet)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first gateway packet")
	}
	select {
	case packet := <-secondAdapter.Packets():
		if packet.GatewayID != secondGateway.ID || string(packet.Payload) != "second" {
			t.Fatalf("second gateway received the wrong packet: %+v", packet)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second gateway packet")
	}
}

func TestUDPAdapterRequeuesDownlinkWhenGatewayTimesOut(t *testing.T) {
	gatewaySocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway: %v", err)
	}
	defer gatewaySocket.Close()

	gateway := testRealGateway(gatewaySocket.LocalAddr().(*net.UDPAddr).Port)
	factory := NewUDPFactory(UDPOptions{
		LocalAddress:     "127.0.0.1:0",
		Timeout:          50 * time.Millisecond,
		ReconnectInitial: 10 * time.Millisecond,
		ReconnectMax:     20 * time.Millisecond,
	})
	adapter, err := factory.NewGatewayAdapter(gateway)
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer adapter.Close()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start adapter: %v", err)
	}
	factory.mu.Lock()
	serverAddress := factory.runtime.conn.LocalAddr().(*net.UDPAddr)
	factory.mu.Unlock()
	eui, _ := hexBytes(gateway.GatewayEUI)
	if _, err := gatewaySocket.WriteToUDP(append(packetHeader([2]byte{0x01, 0x02}, pullDataType), eui...), serverAddress); err != nil {
		t.Fatalf("send PULL_DATA: %v", err)
	}
	readSemtechPacket(t, gatewaySocket, pullAckType)
	waitAdapterEvent(t, adapter.Events(), contracts.Connected)
	waitHeartbeat(t, adapter.Events())

	if err := adapter.Send(ctx, types.GatewayPacket{GatewayID: gateway.ID, Payload: []byte("retry-me")}); err != nil {
		t.Fatalf("send downlink: %v", err)
	}
	readSemtechPacket(t, gatewaySocket, pullResponseType)

	deadline := time.After(time.Second)
	for {
		select {
		case event := <-adapter.Events():
			if event.State != contracts.Error {
				continue
			}
			if event.Packet == nil || string(event.Packet.Payload) != "retry-me" {
				t.Fatalf("expected timed-out downlink to be requeued, got %+v", event)
			}
			if !event.Timeout {
				t.Fatal("expected a timeout flag for a downlink lost with the gateway connection")
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for requeued downlink event")
		}
	}
}

func TestSemtechDataRateParsing(t *testing.T) {
	if got := parseSpreadingFactor("SF7BW125"); got != 7 {
		t.Fatalf("expected SF7, got %d", got)
	}
	if got := parseBandwidth("SF7BW125"); got != 125000 {
		t.Fatalf("expected 125kHz bandwidth, got %d", got)
	}
	if got := parseSpreadingFactor("FSK"); got != 0 {
		t.Fatalf("expected invalid spreading factor to return zero, got %d", got)
	}
}

func TestUDPFactoryConfiguresGatewayBridgeBeforeStart(t *testing.T) {
	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:1700"})

	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: true,
		Address: "127.0.0.1",
		Port:    1700,
	}); err != nil {
		t.Fatalf("configure gateway bridge: %v", err)
	}
	if got := factory.LocalAddress(); got != "127.0.0.1:1700" {
		t.Fatalf("bridge configuration changed the local listen address: %q", got)
	}
	if !factory.SupportsVirtualGateways() {
		t.Fatal("expected virtual gateway transport to be enabled")
	}

	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: true,
		Address: "not/an/address",
		Port:    1700,
	}); err == nil {
		t.Fatal("invalid bridge address should be rejected")
	}
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: false}); err != nil {
		t.Fatalf("disable gateway bridge: %v", err)
	}
	if factory.SupportsVirtualGateways() {
		t.Fatal("disabling the bridge should disable virtual gateway transport")
	}
}

func TestGatewayAndBridgeEndpointParameterMatrix(t *testing.T) {
	for _, test := range []struct {
		name    string
		config  contracts.GatewayBridgeConfig
		wantErr bool
	}{
		{"URL address", contracts.GatewayBridgeConfig{Enabled: true, Address: "udp://127.0.0.1", Port: 1700}, false},
		{"hostname", contracts.GatewayBridgeConfig{Enabled: true, Address: "localhost", Port: 1700}, false},
		{"missing address", contracts.GatewayBridgeConfig{Enabled: true, Port: 1700}, true},
		{"unspecified address", contracts.GatewayBridgeConfig{Enabled: true, Address: "0.0.0.0", Port: 1700}, true},
		{"whitespace", contracts.GatewayBridgeConfig{Enabled: true, Address: "bad host", Port: 1700}, true},
		{"port below range", contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: 0}, true},
		{"port above range", contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: 65536}, true},
	} {
		t.Run("bridge "+test.name, func(t *testing.T) {
			_, err := gatewayBridgeEndpoint(test.config)
			if test.wantErr && err == nil {
				t.Fatal("invalid bridge endpoint was accepted")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("valid bridge endpoint was rejected: %v", err)
			}
		})
	}

	factory := NewUDPFactory(UDPOptions{})
	real := testRealGateway(1700)
	for _, test := range []struct {
		name   string
		mutate func(*contracts.Gateway)
	}{
		{"short EUI", func(gateway *contracts.Gateway) { gateway.GatewayEUI = "abcd" }},
		{"non-hex EUI", func(gateway *contracts.Gateway) { gateway.GatewayEUI = "ZZZZZZZZZZZZZZZZ" }},
		{"missing IPv4", func(gateway *contracts.Gateway) { gateway.GatewayIPv4 = nil }},
		{"non-IPv4 endpoint", func(gateway *contracts.Gateway) { gateway.GatewayIPv4 = gatewayStringPtr("::1") }},
		{"missing port", func(gateway *contracts.Gateway) { gateway.GatewayPort = nil }},
		{"invalid port", func(gateway *contracts.Gateway) { gateway.GatewayPort = gatewayInt32Ptr(0) }},
		{"unsupported type", func(gateway *contracts.Gateway) { gateway.Type = contracts.GatewayType("unsupported") }},
	} {
		t.Run("gateway "+test.name, func(t *testing.T) {
			gateway := real
			test.mutate(&gateway)
			if _, err := factory.NewGatewayAdapter(gateway); err == nil {
				t.Fatal("invalid gateway endpoint was accepted")
			}
		})
	}
	virtual := real
	virtual.Type = contracts.Virtual
	virtual.GatewayIPv4 = nil
	virtual.GatewayPort = nil
	if _, err := factory.NewGatewayAdapter(virtual); !errors.Is(err, ErrGatewayNotConnected) {
		t.Fatalf("virtual gateway without bridge returned the wrong error: %v", err)
	}
}

func TestUDPFactoryCloseStopsAllAdapters(t *testing.T) {
	gatewaySocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway: %v", err)
	}
	defer gatewaySocket.Close()
	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0"})
	adapter, err := factory.NewGatewayAdapter(testRealGateway(gatewaySocket.LocalAddr().(*net.UDPAddr).Port))
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	if err := adapter.Start(t.Context()); err != nil {
		t.Fatalf("start adapter: %v", err)
	}
	if err := factory.Close(); err != nil {
		t.Fatalf("close factory: %v", err)
	}
	if factory.runtime != nil {
		t.Fatal("factory retained a closed runtime")
	}
}

func TestUDPVirtualGatewaySendsKeepaliveAndUplinkToBridge(t *testing.T) {
	bridgeSocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway bridge: %v", err)
	}
	defer bridgeSocket.Close()

	factory := NewUDPFactory(UDPOptions{
		HeartbeatInterval: 50 * time.Millisecond,
		Timeout:           500 * time.Millisecond,
	})
	bridgeAddress := bridgeSocket.LocalAddr().(*net.UDPAddr)
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: true, Address: "127.0.0.1", Port: int32(bridgeAddress.Port),
	}); err != nil {
		t.Fatalf("configure gateway bridge: %v", err)
	}
	virtualGateway := contracts.Gateway{
		ID: "virtual-gateway-test", Active: true, Name: "Virtual Test Gateway",
		Type: contracts.Virtual, GatewayEUI: "A840410001000102",
	}
	adapter, err := factory.NewGatewayAdapter(virtualGateway)
	if err != nil {
		t.Fatalf("create virtual adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer adapter.Close()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start virtual adapter: %v", err)
	}

	keepalive := readSemtechPacket(t, bridgeSocket, pullDataType)
	if len(keepalive) < 12 {
		t.Fatalf("PULL_DATA did not contain gateway EUI: %x", keepalive)
	}
	if _, err := bridgeSocket.WriteToUDP(append(packetHeader([2]byte{keepalive[1], keepalive[2]}, pullAckType), nil...), mustUDPAddr(adapter)); err != nil {
		t.Fatalf("send PULL_ACK: %v", err)
	}
	waitAdapterEvent(t, adapter.Events(), contracts.Connected)

	uplink := types.GatewayPacket{
		GatewayID: "virtual-gateway-test", Payload: []byte("uplink"),
		Frequency: 868300000, Bandwidth: 125000, SpreadingFactor: 7,
	}
	if err := adapter.(types.GatewayUplinkAdapter).SendUplink(ctx, uplink); err != nil {
		t.Fatalf("send virtual uplink: %v", err)
	}
	packet := readSemtechPacket(t, bridgeSocket, pushDataType)
	if len(packet) < 12 {
		t.Fatalf("PUSH_DATA did not contain gateway EUI: %x", packet)
	}
	var body pushDataPayload
	if err := json.Unmarshal(packet[12:], &body); err != nil {
		t.Fatalf("decode PUSH_DATA: %v", err)
	}
	if len(body.RXPK) != 1 || body.RXPK[0].Timestamp == nil || *body.RXPK[0].Timestamp == 0 || body.RXPK[0].Channel != 0 || body.RXPK[0].RFChannel != 0 || body.RXPK[0].Status != 1 || body.RXPK[0].DataRate != "SF7BW125" || body.RXPK[0].Size != len(uplink.Payload) {
		t.Fatalf("unexpected virtual uplink metadata: %+v", body.RXPK)
	}
	decoded, err := base64.StdEncoding.DecodeString(body.RXPK[0].Data)
	if err != nil || string(decoded) != "uplink" {
		t.Fatalf("unexpected virtual uplink payload: %q (%v)", decoded, err)
	}

	for _, signal := range []struct {
		rssi     float64
		snr      float64
		wireRSSI int
	}{
		{-103.7, -6.25, -104},
		{-75.2, 12.5, -75},
		{-90, 0, -90},
		{0, 0, 0},
	} {
		uplink.RSSI, uplink.SNR = signal.rssi, signal.snr
		if err := adapter.(types.GatewayUplinkAdapter).SendUplink(ctx, uplink); err != nil {
			t.Fatalf("send signal metadata: %v", err)
		}
		packet := readSemtechPacket(t, bridgeSocket, pushDataType)
		var wire struct {
			RXPK []map[string]any `json:"rxpk"`
		}
		if err := json.Unmarshal(packet[12:], &wire); err != nil {
			t.Fatal(err)
		}
		if len(wire.RXPK) != 1 || wire.RXPK[0]["rssi"] != float64(signal.wireRSSI) || wire.RXPK[0]["lsnr"] != signal.snr {
			t.Fatalf("signal metadata lost or replaced for %+v: %+v", signal, wire.RXPK)
		}
	}

	downlinkBody := []byte(`{"txpk":{"imme":true,"freq":869.525,"modu":"LORA","datr":"SF9BW125","size":8,"data":"ZG93bmxpbms="}}`)
	downlink := append(packetHeader([2]byte{0x34, 0x56}, pullResponseType), downlinkBody...)
	if _, err := bridgeSocket.WriteToUDP(downlink, mustUDPAddr(adapter)); err != nil {
		t.Fatalf("send bridge downlink to virtual gateway: %v", err)
	}
	readSemtechPacket(t, bridgeSocket, txAckType)
	select {
	case received := <-adapter.Packets():
		if received.Kind != types.GatewayPacketDownlink || string(received.Payload) != "downlink" || received.Frequency != 869525000 {
			t.Fatalf("unexpected virtual bridge downlink: %+v", received)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for virtual bridge downlink")
	}
}

func mustUDPAddr(adapter types.GatewayAdapter) *net.UDPAddr {
	return adapter.(*udpAdapter).conn.LocalAddr().(*net.UDPAddr)
}

func TestUDPFactoryIgnoresDisabledGatewayBridge(t *testing.T) {
	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:1700"})
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: false,
		Address: "not-an-address",
		Port:    0,
	}); err != nil {
		t.Fatalf("disabled bridge configuration should be ignored: %v", err)
	}
	if got := factory.LocalAddress(); got != "127.0.0.1:1700" {
		t.Fatalf("disabled configuration changed listen address to %q", got)
	}
}

func TestUDPFactoryRejectsBridgeReconfigurationWhileRunning(t *testing.T) {
	gatewaySocket, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen fake gateway: %v", err)
	}
	defer gatewaySocket.Close()

	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0"})
	adapter, err := factory.NewGatewayAdapter(testRealGateway(gatewaySocket.LocalAddr().(*net.UDPAddr).Port))
	if err != nil {
		t.Fatalf("create adapter: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer adapter.Close()
	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("start adapter: %v", err)
	}

	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: true,
		Address: "127.0.0.1",
		Port:    1701,
	}); err == nil {
		t.Fatal("running factory should reject bridge reconfiguration")
	}
}

func testRealGateway(port int) contracts.Gateway {
	return contracts.Gateway{
		ID: "6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c01", Active: true, Name: "Real Test Gateway",
		Type: contracts.Real, GatewayEUI: "A840410001000101",
		GatewayIPv4: gatewayStringPtr("127.0.0.1"), GatewayPort: gatewayInt32Ptr(int32(port)),
		Latitude: gatewayFloat32Ptr(37.5), Longitude: gatewayFloat32Ptr(15.1), Altitude: gatewayFloat32Ptr(100),
	}
}

func gatewayStringPtr(value string) *string { return &value }

func gatewayInt32Ptr(value int32) *int32 { return &value }

func gatewayFloat32Ptr(value float32) *float32 { return &value }

func hexBytes(value string) ([]byte, error) {
	return hex.DecodeString(value)
}

func readSemtechPacket(t *testing.T, socket *net.UDPConn, expectedType byte) []byte {
	t.Helper()
	if err := socket.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatalf("set fake gateway deadline: %v", err)
	}
	buffer := make([]byte, 4096)
	for {
		size, _, err := socket.ReadFromUDP(buffer)
		if err != nil {
			t.Fatalf("read Semtech packet: %v", err)
		}
		packet := buffer[:size]
		if len(packet) >= 12 && packet[3] == pushDataType {
			var body pushDataPayload
			if json.Unmarshal(packet[12:], &body) == nil && body.Stat != nil && len(body.RXPK) == 0 {
				continue
			}
		}
		if len(packet) < 4 || packet[0] != protocolVersion || packet[3] != expectedType {
			t.Fatalf("unexpected Semtech packet: %x", packet)
		}
		return packet
	}
}

func readSemtechPacketWithDeadline(socket *net.UDPConn, expectedType byte, timeout time.Duration) ([]byte, error) {
	if err := socket.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	buffer := make([]byte, 4096)
	size, _, err := socket.ReadFromUDP(buffer)
	if err != nil {
		return nil, err
	}
	packet := buffer[:size]
	if len(packet) < 4 || packet[0] != protocolVersion || packet[3] != expectedType {
		return nil, errors.New("unexpected Semtech packet")
	}
	return packet, nil
}

func waitAdapterEvent(t *testing.T, events <-chan types.GatewayAdapterEvent, state contracts.GatewayConnectionState) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if event.State == state {
				return
			}
		case <-deadline:
			t.Fatalf("timed out waiting for gateway state %q", state)
		}
	}
}

func waitHeartbeat(t *testing.T, events <-chan types.GatewayAdapterEvent) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		select {
		case event := <-events:
			if event.Heartbeat {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for gateway heartbeat")
		}
	}
}

func TestVirtualGatewayDecodesConcentratorDownlinkTime(t *testing.T) {
	for _, delay := range []time.Duration{-5 * time.Second, 5 * time.Second, 6 * time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			adapter := &udpAdapter{virtual: true}
			target := time.Now().Add(delay)
			timestamp := uint32(target.UnixMicro())
			body, err := json.Marshal(struct {
				TXPK txPacket `json:"txpk"`
			}{TXPK: txPacket{Timestamp: &timestamp, Data: "AQ==", Size: 1, Modulation: "LORA"}})
			if err != nil {
				t.Fatal(err)
			}
			packet, _, err := adapter.decodePullResponse(body)
			if err != nil {
				t.Fatal(err)
			}
			if difference := packet.TransmitAt.Sub(target); difference < -time.Microsecond || difference > time.Microsecond {
				t.Fatalf("wrong concentrator time: got %s, want %s", packet.TransmitAt, target)
			}
			if packet.ReceivedAt.IsZero() {
				t.Fatal("missing reception timestamp")
			}
		})
	}
}

func TestGatewayConcentratorTimestampDoesNotOverrideImmediateOrPhysicalClock(t *testing.T) {
	for _, virtual := range []bool{false, true} {
		adapter := &udpAdapter{virtual: virtual}
		packet, _, err := adapter.decodePullResponse([]byte(`{"txpk":{"tmst":42,"imme":true,"data":"AQ=="}}`))
		if err != nil || !packet.TransmitAt.IsZero() {
			t.Fatalf("immediate packet gained a transmission deadline: %+v %v", packet, err)
		}
	}
	adapter := &udpAdapter{}
	packet, _, err := adapter.decodePullResponse([]byte(`{"txpk":{"tmst":42,"data":"AQ=="}}`))
	if err != nil || !packet.TransmitAt.IsZero() {
		t.Fatalf("physical concentrator clock was interpreted as local wall time: %+v %v", packet, err)
	}
}

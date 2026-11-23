package gateway

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math"
	"net"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestUDPBridgeReportsGatewayStatistics(t *testing.T) {
	peer := listenStatisticsTestPeer(t)
	otherPeer := listenStatisticsTestPeer(t)
	factory := NewUDPFactory(UDPOptions{HeartbeatInterval: 5 * time.Second, Timeout: 10 * time.Second})
	t.Cleanup(func() { factory.Close() })
	port := peer.LocalAddr().(*net.UDPAddr).Port
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: int32(port)}); err != nil {
		t.Fatal(err)
	}
	config := testRealGateway(port)
	config.Type = contracts.Virtual
	interval := int32(1)
	latitude, longitude, altitude := float32(37.5), float32(15.1), float32(100.6)
	config.KeepAlive = &interval
	config.Latitude, config.Longitude, config.Altitude = &latitude, &longitude, &altitude
	adapter, err := factory.NewGatewayAdapter(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { adapter.Close() })
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := adapter.Start(ctx); err != nil {
		t.Fatal(err)
	}

	send := func(socket *net.UDPConn, datagram []byte) {
		t.Helper()
		if _, err := socket.WriteToUDP(datagram, mustUDPAddr(adapter)); err != nil {
			t.Fatal(err)
		}
	}
	pull := readSemtechPacket(t, peer, pullDataType)
	send(peer, packetHeader([2]byte{pull[1], pull[2]}, pullAckType))
	waitAdapterEvent(t, adapter.Events(), contracts.Connected)
	firstToken, first := readStatisticsReport(t, peer, config.GatewayEUI)
	if first.RadioReceived != 0 || first.Downlinks != 0 || first.Transmitted != 0 || first.AckPercentage != 0 ||
		first.Latitude == nil || *first.Latitude != latitude || first.Longitude == nil || *first.Longitude != longitude ||
		first.Altitude == nil || *first.Altitude != 101 {
		t.Fatalf("unexpected initial report: %+v", first)
	}
	send(peer, packetHeader(firstToken, pushAckType))
	waitHeartbeat(t, adapter.Events())
	packet := types.GatewayPacket{Payload: []byte{1, 2}, Frequency: 868100000, SpreadingFactor: 7, Bandwidth: 125000}
	for index := range 2 {
		if err := adapter.(types.GatewayUplinkAdapter).SendUplink(ctx, packet); err != nil {
			t.Fatal(err)
		}
		uplink := readSemtechPacket(t, peer, pushDataType)
		token := [2]byte{uplink[1], uplink[2]}
		if index == 0 {
			send(peer, packetHeader(token, pushAckType))
			send(peer, packetHeader(token, pushAckType))
		} else {
			send(otherPeer, packetHeader(token, pushAckType))
		}
	}
	latitude = 38
	adapter.(types.GatewayPositionAdapter).SetGatewayPosition(&latitude, &longitude, &altitude)
	downlink := append(packetHeader([2]byte{1, 2}, pullResponseType),
		[]byte(`{"txpk":{"imme":true,"freq":868.1,"modu":"LORA","datr":"SF7BW125","size":2,"data":"AQI="}}`)...)
	send(peer, downlink)
	readSemtechPacket(t, peer, txAckType)
	select {
	case <-adapter.Packets():
	case <-time.After(time.Second):
		t.Fatal("immediate downlink was not received")
	}
	_, report := readStatisticsReport(t, peer, config.GatewayEUI)
	if report.RadioReceived != 2 || report.ValidCRC != 2 || report.RadioForwarded != 2 ||
		report.Downlinks != 1 || report.Transmitted != 1 || math.Abs(report.AckPercentage-200.0/3) > 0.001 ||
		report.Latitude == nil || *report.Latitude != latitude {
		t.Fatalf("unexpected active report: %+v", report)
	}
	_, idle := readStatisticsReport(t, peer, config.GatewayEUI)
	if idle.RadioReceived != 0 || idle.RadioForwarded != 0 || idle.Downlinks != 0 || idle.Transmitted != 0 {
		t.Fatalf("counters should restart each reporting interval: %+v", idle)
	}
}

func listenStatisticsTestPeer(t *testing.T) *net.UDPConn {
	t.Helper()
	peer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { peer.Close() })
	return peer
}

func readStatisticsReport(t *testing.T, peer *net.UDPConn, eui string) ([2]byte, gatewayStatistics) {
	t.Helper()
	if err := peer.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	wantEUI, err := hex.DecodeString(eui)
	if err != nil {
		t.Fatal(err)
	}
	var buffer [2048]byte
	for {
		size, _, err := peer.ReadFromUDP(buffer[:])
		if err != nil {
			t.Fatal(err)
		}
		if size < 12 || buffer[3] != pushDataType {
			continue
		}
		var message pushDataPayload
		if err := json.Unmarshal(buffer[12:size], &message); err != nil {
			t.Fatal(err)
		}
		if message.Stat == nil {
			continue
		}
		if buffer[0] != protocolVersion || !equalBytes(buffer[4:12], wantEUI) || len(message.RXPK) != 0 {
			t.Fatalf("invalid report envelope: %x", buffer[:size])
		}
		if _, err := time.Parse("2006-01-02 15:04:05 GMT", message.Stat.Time); err != nil {
			t.Fatalf("invalid UTC report timestamp: %v", err)
		}
		return [2]byte{buffer[1], buffer[2]}, *message.Stat
	}
}

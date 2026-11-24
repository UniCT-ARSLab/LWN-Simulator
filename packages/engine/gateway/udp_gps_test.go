package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestSemtechGPSTimeConversion(t *testing.T) {
	for _, example := range []struct {
		utc string
		gps uint64
	}{
		{"1980-01-06T00:00:00Z", 0},
		{"2016-12-31T23:59:59.999Z", 1167264016999},
		{"2017-01-01T00:00:00Z", 1167264018000},
		{"2020-10-06T13:25:02.523Z", 1286025920523},
	} {
		utc, err := time.Parse(time.RFC3339Nano, example.utc)
		if err != nil {
			t.Fatal(err)
		}
		decoded, valid := timeFromGPSMilliseconds(example.gps)
		if got := gpsMilliseconds(utc); got != example.gps || !valid || !decoded.Equal(utc) {
			t.Fatalf("%s: GPS time %d, decoded %s (valid: %v)", example.utc, got, decoded, valid)
		}
	}
	for _, invalid := range []uint64{1167264017000, math.MaxUint64} {
		if _, valid := timeFromGPSMilliseconds(invalid); valid {
			t.Fatalf("accepted GPS value %d with no representable UTC timestamp", invalid)
		}
	}
}

func TestRealGatewayDoesNotSubstituteHostClockForGPS(t *testing.T) {
	adapter := &udpAdapter{}
	_, _, err := adapter.decodePullResponse([]byte(`{"txpk":{"tmms":1234567890,"data":"AQ=="}}`))
	if !errors.Is(err, errGatewayGPSUnlocked) {
		t.Fatalf("expected GPS_UNLOCKED, got %v", err)
	}
}

func TestUDPVirtualGatewaySchedulesGPSDownlink(t *testing.T) {
	peer := listenStatisticsTestPeer(t)
	port := peer.LocalAddr().(*net.UDPAddr).Port
	factory := NewUDPFactory(UDPOptions{PacketBuffer: 2})
	t.Cleanup(func() { factory.Close() })
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: int32(port)}); err != nil {
		t.Fatal(err)
	}
	config := testRealGateway(port)
	config.Type = contracts.Virtual
	adapter, err := factory.NewGatewayAdapter(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { adapter.Close() })
	if err := adapter.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	readSemtechPacket(t, peer, pullDataType)
	readStatisticsReport(t, peer, config.GatewayEUI)
	if err := adapter.(types.GatewayUplinkAdapter).SendUplink(context.Background(), types.GatewayPacket{Payload: []byte{1}}); err != nil {
		t.Fatal(err)
	}
	uplink := readSemtechPacket(t, peer, pushDataType)
	var body pushDataPayload
	if err := json.Unmarshal(uplink[12:], &body); err != nil {
		t.Fatal(err)
	}
	rx := body.RXPK[0]
	if rx.GPSMilliseconds == nil || rx.Timestamp == nil {
		t.Fatal("virtual uplink is missing GPS or concentrator time")
	}
	utc, valid := timeFromGPSMilliseconds(*rx.GPSMilliseconds)
	if offset := int32(*rx.Timestamp - uint32(utc.UnixMicro())); !valid || offset < 0 || offset >= 1000 {
		t.Fatalf("GPS and concentrator times do not describe the same instant: %d µs", offset)
	}

	submit := func(token byte, when time.Time, payload, want string) {
		t.Helper()
		body := fmt.Appendf(packetHeader([2]byte{token, 0}, pullResponseType),
			`{"txpk":{"tmms":%d,"modu":"LORA","freq":868.1,"datr":"SF7BW125","size":1,"data":%q}}`, gpsMilliseconds(when), payload)
		if _, err := peer.WriteToUDP(body, mustUDPAddr(adapter)); err != nil {
			t.Fatal(err)
		}
		ack := readSemtechPacket(t, peer, txAckType)
		var response txAckPayload
		if err := json.Unmarshal(ack[12:], &response); err != nil || ack[1] != token || response.TXPKAck.Error != want {
			t.Fatalf("wanted TX_ACK %s for token %d, got %s (%v)", want, token, ack[12:], err)
		}
	}
	now := time.Now().Truncate(time.Millisecond)
	first, second := now.Add(500*time.Millisecond), now.Add(time.Second)
	submit(1, second, "Ag==", "NONE")
	submit(2, first, "AQ==", "NONE")
	submit(3, now.Add(3*time.Second), "Aw==", "COLLISION_PACKET")
	submit(4, now.Add(-time.Second), "BA==", "TOO_LATE")
	submit(5, now.Add(time.Hour), "BQ==", "TOO_EARLY")
	for index, deadline := range []time.Time{first, second} {
		select {
		case packet := <-adapter.Packets():
			if !packet.TransmitAt.Equal(deadline) || time.Now().Before(deadline) || !equalBytes(packet.Payload, []byte{byte(index + 1)}) {
				t.Fatalf("early or incorrect GPS downlink: %+v", packet)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("GPS downlink was not released")
		}
	}
}

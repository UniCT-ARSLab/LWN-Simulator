package gateway

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestPhysicalGatewaySchedulesUsingOwnWrappingCounter(t *testing.T) {
	for _, counter := range []uint32{0, 491697203, ^uint32(0) - 500000} {
		peer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			t.Fatal(err)
		}
		factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: time.Second})
		transport, err := factory.NewGatewayAdapter(testRealGateway(peer.LocalAddr().(*net.UDPAddr).Port))
		if err != nil {
			t.Fatal(err)
		}
		adapter := transport.(*udpAdapter)
		ctx, cancel := context.WithCancel(t.Context())
		if err := adapter.Start(ctx); err != nil {
			t.Fatal(err)
		}
		anchor := time.Now()
		adapter.mu.Lock()
		adapter.clockTimestamp = counter
		adapter.clockReceivedAt = anchor
		adapter.mu.Unlock()
		if err := adapter.Send(ctx, types.GatewayPacket{Payload: []byte("data"), TransmitAt: anchor.Add(2 * time.Second)}); err != nil {
			t.Fatal(err)
		}
		response := readSemtechPacket(t, peer, pullResponseType)
		var body struct {
			TXPK txPacket `json:"txpk"`
		}
		if err := json.Unmarshal(response[4:], &body); err != nil {
			t.Fatal(err)
		}
		expected := counter + 2000000
		if body.TXPK.Timestamp == nil || *body.TXPK.Timestamp != expected || body.TXPK.Time != "" || body.TXPK.Immediate {
			t.Fatalf("wrong concentrator clock: %+v expected=%d", body.TXPK, expected)
		}
		cancel()
		adapter.Close()
		peer.Close()
	}
}

func TestPhysicalGatewayRejectsUnknownOrStaleClock(t *testing.T) {
	for _, receivedAt := range []time.Time{{}, time.Now().Add(-time.Hour)} {
		adapter := &udpAdapter{clockReceivedAt: receivedAt}
		err := adapter.Send(t.Context(), types.GatewayPacket{Payload: []byte("data"), TransmitAt: time.Now().Add(time.Second)})
		if err == nil || !strings.Contains(err.Error(), "concentrator") {
			t.Fatalf("unusable clock accepted: %v", err)
		}
	}
}

func TestGatewayRejectsTooEarlyAndTooLateTransmissions(t *testing.T) {
	for _, reason := range []string{"TOO_EARLY", "TOO_LATE"} {
		adapter := &udpAdapter{pending: map[string]types.GatewayPacket{"0102": {Payload: []byte("downlink")}}}
		body := []byte(`{"txpk_ack":{"error":"` + reason + `"}}`)
		packet, err := adapter.handleTXAck([]byte{1, 2}, body)
		if err == nil || !strings.Contains(err.Error(), reason) || packet == nil || string(packet.Payload) != "downlink" {
			t.Fatalf("gateway timing rejection was lost: %v %+v", err, packet)
		}
	}
}

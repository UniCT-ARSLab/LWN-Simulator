package runtime

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"testing"
	"time"
)

func TestGatewayReplyTimingUsesReceptionAndWraps(t *testing.T) {
	for _, counter := range []uint32{0, 491697203, ^uint32(0) - 100} {
		for _, delay := range []time.Duration{time.Second, 2 * time.Second, 5 * time.Second, 6 * time.Second} {
			receivedAt := time.Now().Add(-200 * time.Millisecond)
			at, stamp := gatewayReplyTiming(types.GatewayPacket{ReceivedAt: receivedAt, ConcentratorTimestamp: &counter}, delay)
			if stamp == nil || *stamp != counter+uint32(delay.Microseconds()) || !at.Equal(receivedAt.Add(delay)) {
				t.Fatalf("processing latency changed radio schedule: at=%v stamp=%v", at, stamp)
			}
		}
	}
}

func TestRealJoinAcceptPreservesUplinkModulationAndJoinDelay(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000991")
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7991")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{})
	if err != nil {
		t.Fatal(err)
	}
	session := engine.sessions[device.ID]
	session.DeviceAddress = "26011BDA"
	counter := uint32(491697203)
	packet := types.GatewayPacket{GatewayID: gateway.ID, Frequency: 868100000, Bandwidth: 125000, SpreadingFactor: 12, DataRate: "SF12BW125", ConcentratorTimestamp: &counter, ReceivedAt: time.Now().Add(-100 * time.Millisecond)}
	if !engine.queueRealJoinAcceptLocked(packet, device, session, mustAppKey(device)) {
		t.Fatal("join reply not queued")
	}
	reply := engine.gatewayPackets[0]
	if reply.ConcentratorTimestamp == nil || *reply.ConcentratorTimestamp != counter+5000000 || reply.SpreadingFactor != 12 || reply.DataRate != "SF12BW125" || !reply.TransmitAt.Equal(packet.ReceivedAt.Add(5*time.Second)) {
		t.Fatalf("invalid physical Join-Accept timing or modulation: %+v", reply)
	}
}

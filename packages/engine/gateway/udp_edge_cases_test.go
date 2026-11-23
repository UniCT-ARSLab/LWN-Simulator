package gateway

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestUDPSharedPeerRoutesTXAckByGatewayEUI(t *testing.T) {
	peer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: 5 * time.Second})
	defer factory.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	const count = 5
	adapters := make([]types.GatewayAdapter, count)
	gateways := make([]contracts.Gateway, count)
	for index := range adapters {
		config := testRealGateway(peer.LocalAddr().(*net.UDPAddr).Port)
		config.ID = fmt.Sprintf("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c%02d", index+70)
		config.GatewayEUI = fmt.Sprintf("A8404100010001%02d", index+70)
		gateways[index] = config
		adapters[index], err = factory.NewGatewayAdapter(config)
		if err != nil {
			t.Fatal(err)
		}
		if err := adapters[index].Start(ctx); err != nil {
			t.Fatal(err)
		}
	}
	factory.mu.Lock()
	server := cloneUDPAddr(factory.runtime.conn.LocalAddr().(*net.UDPAddr))
	factory.mu.Unlock()
	for index, adapter := range adapters {
		eui, _ := hex.DecodeString(gateways[index].GatewayEUI)
		if _, err := peer.WriteToUDP(append(packetHeader([2]byte{byte(index), 1}, pullDataType), eui...), server); err != nil {
			t.Fatal(err)
		}
		readSemtechPacket(t, peer, pullAckType)
		waitAdapterEvent(t, adapter.Events(), contracts.Connected)
		waitHeartbeat(t, adapter.Events())
	}
	for index, adapter := range adapters {
		payload := []byte{0xde, 0xad, byte(index)}
		if err := adapter.Send(ctx, types.GatewayPacket{
			GatewayID: gateways[index].ID, DeviceID: fmt.Sprintf("device-%d", index),
			Protocol: contracts.RawLoRa, Kind: types.GatewayPacketDownlink, Payload: payload,
		}); err != nil {
			t.Fatal(err)
		}
		response := readSemtechPacket(t, peer, pullResponseType)
		eui, _ := hex.DecodeString(gateways[index].GatewayEUI)
		ack := append(packetHeader([2]byte{response[1], response[2]}, txAckType), eui...)
		ack = append(ack, []byte(`{"txpk_ack":{"error":"TOO_LATE"}}`)...)
		if _, err := peer.WriteToUDP(ack, server); err != nil {
			t.Fatal(err)
		}
		select {
		case event := <-adapter.Events():
			if event.Packet == nil || event.Packet.GatewayID != gateways[index].ID || event.Packet.Protocol != contracts.RawLoRa || string(event.Packet.Payload) != string(payload) {
				t.Fatalf("gateway %d lost or received the wrong pending packet: %+v", index, event)
			}
		case <-time.After(time.Second):
			t.Fatalf("TX_ACK was not routed to gateway %d by EUI", index)
		}
	}
}

func TestUDPBridgeRejectsMalformedDownlinksAndContinuesReceiving(t *testing.T) {
	peer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	factory := NewUDPFactory(UDPOptions{HeartbeatInterval: time.Second, Timeout: 5 * time.Second})
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: int32(peer.LocalAddr().(*net.UDPAddr).Port)}); err != nil {
		t.Fatal(err)
	}
	config := testRealGateway(1700)
	config.Type = contracts.Virtual
	adapter, err := factory.NewGatewayAdapter(config)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := adapter.Start(ctx); err != nil {
		t.Fatal(err)
	}
	keepalive := readSemtechPacket(t, peer, pullDataType)
	remote := mustUDPAddr(adapter)
	if _, err := peer.WriteToUDP(packetHeader([2]byte{keepalive[1], keepalive[2]}, pullAckType), remote); err != nil {
		t.Fatal(err)
	}
	waitAdapterEvent(t, adapter.Events(), contracts.Connected)
	waitHeartbeat(t, adapter.Events())
	for index, test := range []struct{ body, want string }{
		{`{`, "INVALID_DATA"},
		{`{"txpk":{"modu":"FSK","data":"3q0=","size":2}}`, "INVALID_DATA"},
		{`{"txpk":{"modu":"LORA","data":"???","size":2}}`, "INVALID_DATA"},
		{`{"txpk":{"modu":"LORA","data":"3q0=","size":9}}`, "INVALID_DATA"},
		{`{"txpk":{"modu":"LORA","data":""}}`, "INVALID_DATA"},
		{`{"txpk":{"tmms":1234567890,"modu":"LORA","data":"3q0=","size":2}}`, "TOO_LATE"},
	} {
		request := append(packetHeader([2]byte{byte(index), 2}, pullResponseType), test.body...)
		if _, err := peer.WriteToUDP(request, remote); err != nil {
			t.Fatal(err)
		}
		ack := readSemtechPacket(t, peer, txAckType)
		var decoded txAckPayload
		if err := json.Unmarshal(ack[12:], &decoded); err != nil || decoded.TXPKAck.Error != test.want {
			t.Fatalf("invalid downlink %d was acknowledged as valid: %s (%v)", index, ack, err)
		}
		waitAdapterEvent(t, adapter.Events(), contracts.Error)
	}
	valid := []byte{0, 0xde, 0xad, 0xff}
	body, _ := json.Marshal(map[string]any{"txpk": map[string]any{
		"imme": true, "modu": "LORA", "freq": 868.1, "datr": "SF7BW125", "size": len(valid), "data": base64.StdEncoding.EncodeToString(valid),
	}})
	if _, err := peer.WriteToUDP(append(packetHeader([2]byte{99, 2}, pullResponseType), body...), remote); err != nil {
		t.Fatal(err)
	}
	ack := readSemtechPacket(t, peer, txAckType)
	var decoded txAckPayload
	if err := json.Unmarshal(ack[12:], &decoded); err != nil || decoded.TXPKAck.Error != "NONE" {
		t.Fatalf("valid downlink was rejected after malformed traffic: %s (%v)", ack, err)
	}
	select {
	case packet := <-adapter.Packets():
		if string(packet.Payload) != string(valid) {
			t.Fatalf("unexpected decoded packet: %x", packet.Payload)
		}
	case <-time.After(time.Second):
		t.Fatal("adapter did not recover after malformed downlinks")
	}
}

func TestUDPRawPacketSurvivesTimeoutAndReconnect(t *testing.T) {
	for _, bridge := range []bool{false, true} {
		t.Run(fmt.Sprintf("bridge=%v", bridge), func(t *testing.T) {
			peer, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
			if err != nil {
				t.Fatal(err)
			}
			defer peer.Close()
			factory := NewUDPFactory(UDPOptions{LocalAddress: "127.0.0.1:0", HeartbeatInterval: 20 * time.Millisecond, Timeout: 120 * time.Millisecond, ReconnectInitial: 10 * time.Millisecond, ReconnectMax: 20 * time.Millisecond})
			defer factory.Close()
			config := testRealGateway(peer.LocalAddr().(*net.UDPAddr).Port)
			if bridge {
				config.Type = contracts.Virtual
				if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: *config.GatewayPort}); err != nil {
					t.Fatal(err)
				}
			}
			adapter, err := factory.NewGatewayAdapter(config)
			if err != nil {
				t.Fatal(err)
			}
			defer adapter.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := adapter.Start(ctx); err != nil {
				t.Fatal(err)
			}
			var remote *net.UDPAddr
			eui, _ := hex.DecodeString(config.GatewayEUI)
			heartbeat := func() {
				t.Helper()
				var frame []byte
				if bridge {
					rawAdapter := adapter.(*udpAdapter)
					rawAdapter.mu.Lock()
					token := rawAdapter.pullToken
					rawAdapter.mu.Unlock()
					frame = packetHeader(token, pullAckType)
				} else {
					factory.mu.Lock()
					remote = cloneUDPAddr(factory.runtime.conn.LocalAddr().(*net.UDPAddr))
					factory.mu.Unlock()
					frame = append(packetHeader([2]byte{7, 8}, pullDataType), eui...)
				}
				if _, err := peer.WriteToUDP(frame, remote); err != nil {
					t.Fatal(err)
				}
			}
			if bridge {
				readSemtechPacket(t, peer, pullDataType)
				remote = mustUDPAddr(adapter)
			}
			heartbeat()
			waitAdapterEvent(t, adapter.Events(), contracts.Connected)
			waitHeartbeat(t, adapter.Events())
			packet := types.GatewayPacket{GatewayID: config.ID, DeviceID: "raw-device", Protocol: contracts.RawLoRa, Kind: types.GatewayPacketDownlink,
				Payload: []byte{0xde, 0xad, 0, 0xff}, Frequency: 868300000, Bandwidth: 125000, SpreadingFactor: 9, DataRate: "SF9BW125", Power: 14}
			if err := adapter.Send(ctx, packet); err != nil {
				t.Fatal(err)
			}
			var pending *types.GatewayPacket
			deadline := time.After(2 * time.Second)
			for pending == nil {
				select {
				case event := <-adapter.Events():
					if event.Timeout && event.Packet != nil {
						pending = event.Packet
					}
				case <-deadline:
					t.Fatal("missing raw packet after transport timeout")
				}
			}
			if pending.Protocol != packet.Protocol || pending.DeviceID != packet.DeviceID || pending.Frequency != packet.Frequency || pending.DataRate != packet.DataRate || string(pending.Payload) != string(packet.Payload) {
				t.Fatalf("timeout lost raw packet metadata: %+v", pending)
			}
			waitAdapterEvent(t, adapter.Events(), contracts.Reconnecting)
			heartbeat()
			waitAdapterEvent(t, adapter.Events(), contracts.Connected)
			waitHeartbeat(t, adapter.Events())
			if err := adapter.Send(ctx, *pending); err != nil {
				t.Fatalf("raw retry failed after reconnection: %v", err)
			}
		})
	}
}

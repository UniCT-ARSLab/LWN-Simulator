package runtime

import (
	"encoding/hex"
	"net"
	"strconv"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/gateway"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestEngineMixedUDPDownlinksRejectInvalidMICAndReplay(t *testing.T) {
	for _, bridge := range []bool{false, true} {
		name := "real-gateway"
		if bridge {
			name = "virtual-gateway-bridge"
		}
		t.Run(name, func(t *testing.T) {
			peer := listenIntegrationUDP(t)
			peerAddress := peer.LocalAddr().(*net.UDPAddr)
			port := reserveIntegrationUDPPort(t)
			factory := gateway.NewUDPFactory(gateway.UDPOptions{
				LocalAddress:      net.JoinHostPort("127.0.0.1", strconv.Itoa(port)),
				HeartbeatInterval: time.Second, Timeout: 5 * time.Second,
			})
			device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000708")
			configureABP(&device)
			device.RadioReceiveMode = receiveModePointer(contracts.ReceiveAuto)
			config := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c98")
			if bridge {
				if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: int32(peerAddress.Port)}); err != nil {
					t.Fatal(err)
				}
			} else {
				config.Type = contracts.Real
				config.KeepAlive = nil
				config.GatewayIPv4 = stringPointer("127.0.0.1")
				config.GatewayPort = int32Pointer(peerAddress.Port)
			}
			engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{config}, types.Options{
				Clock: types.NewManualClock(0), EventBuffer: 512, GatewayAdapterFactory: factory,
			})
			if err != nil {
				t.Fatal(err)
			}
			startEngine(t, engine)
			defer stopEngine(t, engine)
			var remote *net.UDPAddr
			if bridge {
				keepalive := readIntegrationPacket(t, peer, semtechPullData)
				remote = keepalive.remote
				writeIntegrationPacket(t, peer, remote, semtechPullAck, keepalive.token, nil)
			} else {
				remote = &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port}
				eui, _ := hex.DecodeString(config.GatewayEUI)
				writeIntegrationPacket(t, peer, remote, semtechPullData, [2]byte{1, 1}, eui)
				readIntegrationPacket(t, peer, semtechPullAck)
			}
			waitForEventType(t, engine.Events(), contracts.GatewayConnected)
			valid := buildIntegrationABPDownlink(t, device)
			invalid := append([]byte(nil), valid...)
			invalid[len(invalid)-1] ^= 0xff
			writeIntegrationPullResponse(t, peer, remote, [2]byte{2, 1}, invalid)
			readIntegrationPacket(t, peer, semtechTxAck)
			waitForEventType(t, engine.Events(), contracts.DeviceDownlinkDropped)
			if engine.Snapshot().Devices[0].FrameCounterDown != 0 {
				t.Fatal("invalid MIC mutated the device counter")
			}
			writeIntegrationPullResponse(t, peer, remote, [2]byte{3, 1}, valid)
			readIntegrationPacket(t, peer, semtechTxAck)
			waitForEventType(t, engine.Events(), contracts.DeviceDownlinkTransmitted)
			counter := engine.Snapshot().Devices[0].FrameCounterDown
			writeIntegrationPullResponse(t, peer, remote, [2]byte{4, 1}, valid)
			readIntegrationPacket(t, peer, semtechTxAck)
			waitForEventType(t, engine.Events(), contracts.FrameCounterRejected)
			if engine.Snapshot().Devices[0].FrameCounterDown != counter {
				t.Fatal("replayed downlink mutated the counter")
			}
			payload := []byte{0xde, 0xad, 0xbe, 0xef}
			writeIntegrationPullResponse(t, peer, remote, [2]byte{5, 1}, payload)
			readIntegrationPacket(t, peer, semtechTxAck)
			event := waitForEventType(t, engine.Events(), contracts.GatewayPacketIngress)
			if event.RadioFrame == nil || event.RadioFrame.Protocol != contracts.RawLoRa || string(event.RadioFrame.Payload) != string(payload) {
				t.Fatalf("raw response after LoRaWAN failures was not preserved: %+v", event)
			}
		})
	}
}

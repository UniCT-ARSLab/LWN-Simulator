package runtime

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/gateway"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

const (
	semtechVersion  byte = 2
	semtechPushData byte = 0x00
	semtechPushAck  byte = 0x01
	semtechPullData byte = 0x02
	semtechPullResp byte = 0x03
	semtechPullAck  byte = 0x04
	semtechTxAck    byte = 0x05
)

type semtechRXPacket struct {
	Timestamp  *uint32 `json:"tmst,omitempty"`
	Data       string  `json:"data"`
	Frequency  float64 `json:"freq"`
	DataRate   string  `json:"datr"`
	Modulation string  `json:"modu"`
	Size       int     `json:"size"`
}

type semtechPushDataPayload struct {
	RXPK []semtechRXPacket `json:"rxpk"`
}

type semtechTXPacket struct {
	Timestamp      *uint32 `json:"tmst,omitempty"`
	Time           string  `json:"time,omitempty"`
	InvertPolarity bool    `json:"ipol"`
	Immediate      bool    `json:"imme"`
	Frequency      float64 `json:"freq"`
	Modulation     string  `json:"modu"`
	DataRate       string  `json:"datr"`
	CodingRate     string  `json:"codr"`
	Size           int     `json:"size"`
	Data           string  `json:"data"`
}

type semtechPullResponse struct {
	TXPK semtechTXPacket `json:"txpk"`
}

func TestEngineVirtualGatewayBridgeOTAAOverSemtechUDP(t *testing.T) {
	bridgeSocket := listenIntegrationUDP(t)
	bridgeAddress := bridgeSocket.LocalAddr().(*net.UDPAddr)
	factory := gateway.NewUDPFactory(gateway.UDPOptions{
		LocalAddress:      "127.0.0.1:0",
		HeartbeatInterval: time.Second,
		Timeout:           3 * time.Second,
	})
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{
		Address: "127.0.0.1",
		Enabled: true,
		Port:    int32(bridgeAddress.Port),
	}); err != nil {
		t.Fatalf("configure UDP bridge: %v", err)
	}

	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000091")
	device.PayloadConfig.UplinkInterval = 1
	device.PayloadConfig.MType = contracts.ConfirmedDataUp
	device.RX1Config.Delay = intPtr(1000)
	device.RX2Config.Delay = intPtr(2000)
	gatewayConfig := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c91")
	clock := types.NewManualClock(0)
	engine, err := New(
		contracts.SimulationConfig{Speed: 10},
		[]contracts.Device{device},
		[]contracts.Gateway{gatewayConfig},
		types.Options{Clock: clock, EventBuffer: 256, GatewayAdapterFactory: factory},
	)
	if err != nil {
		t.Fatalf("create bridge engine: %v", err)
	}
	if engine.Snapshot().State.Speed != 1 {
		t.Fatal("external UDP bridge must use a real-time clock")
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	keepalive := readIntegrationPacket(t, bridgeSocket, semtechPullData)
	bridgePeer := keepalive.remote
	writeIntegrationPacket(t, bridgeSocket, bridgePeer, semtechPullAck, keepalive.token, nil)
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	if err := clock.Advance(500 * time.Millisecond); err != nil {
		t.Fatalf("advance bridge simulation to join request: %v", err)
	}
	joinPacket := readIntegrationPacket(t, bridgeSocket, semtechPushData)
	joinRequest := decodeIntegrationUplink(t, joinPacket.body, gatewayConfig.GatewayEUI)
	decodedJoinRequest, err := lorawan.Parse(joinRequest)
	if err != nil || decodedJoinRequest.MType != lorawan.MTypeJoinRequest {
		t.Fatalf("bridge did not receive a valid OTAA Join Request: packet=%x error=%v", joinRequest, err)
	}

	joinAcceptOptions := lorawan.JoinAcceptOptions{
		AppNonce:    0x010203,
		NetID:       0x000001,
		DevAddr:     0x26011BDA,
		RX2DataRate: 5,
		RXDelay:     1,
		AppKey:      mustAppKey(device),
		CFList: []byte{
			0x18, 0x4f, 0x84, 0xe8, 0x56, 0x84, 0xb8, 0x5e,
			0x84, 0x88, 0x66, 0x84, 0x00, 0x00, 0x00, 0x00,
		},
	}
	joinAccept, err := lorawan.BuildJoinAccept(joinAcceptOptions)
	if err != nil {
		t.Fatalf("build bridge Join-Accept: %v", err)
	}
	if len(joinAccept) != 33 {
		t.Fatalf("bridge integration test requires a 33-byte Join-Accept, got %d", len(joinAccept))
	}
	writeIntegrationPullResponse(t, bridgeSocket, bridgePeer, [2]byte{0x31, 0x32}, joinAccept)
	readIntegrationPacket(t, bridgeSocket, semtechTxAck)
	waitForEventType(t, engine.Events(), contracts.JoinAcceptReceived)
	if snapshot := engine.Snapshot(); len(snapshot.Devices) != 1 || !snapshot.Devices[0].Joined {
		t.Fatalf("device did not enter the joined state after the external Join-Accept: %+v", snapshot.Devices)
	}

	if err := clock.Advance(2 * time.Second); err != nil {
		t.Fatalf("advance bridge simulation to first uplink: %v", err)
	}
	dataPacket := readIntegrationPacket(t, bridgeSocket, semtechPushData)
	dataUplink := decodeIntegrationUplink(t, dataPacket.body, gatewayConfig.GatewayEUI)
	decodedDataUplink, err := lorawan.Parse(dataUplink)
	if err != nil || decodedDataUplink.MType != lorawan.MTypeConfirmedDataUp || decodedDataUplink.DevAddr != joinAcceptOptions.DevAddr || decodedDataUplink.FCnt != 1 {
		t.Fatalf("bridge did not receive the first confirmed post-join uplink: packet=%x decoded=%+v error=%v", dataUplink, decodedDataUplink, err)
	}
	nwkSKey, appSKey, err := lorawan.DeriveSessionKeys(
		joinAcceptOptions.AppKey,
		joinAcceptOptions.AppNonce,
		joinAcceptOptions.NetID,
		decodedJoinRequest.DevNonce,
	)
	if err != nil {
		t.Fatalf("derive bridge session keys: %v", err)
	}
	if !lorawan.VerifyDataMIC(decodedDataUplink, nwkSKey, 0) {
		t.Fatal("post-join uplink MIC does not match the negotiated network session key")
	}
	decryptedPayload, err := lorawan.DecryptPayload(decodedDataUplink, appSKey, 0)
	if err != nil || string(decryptedPayload) != device.PayloadConfig.Payload {
		t.Fatalf("post-join payload was not transported intact: payload=%q error=%v", decryptedPayload, err)
	}

	downlinkACK, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr:   joinAcceptOptions.DevAddr,
		FCnt:      0,
		Direction: 1,
		ACK:       true,
		NwkSKey:   nwkSKey,
		AppSKey:   appSKey,
	})
	if err != nil {
		t.Fatalf("build bridge downlink ACK: %v", err)
	}
	writeIntegrationPullResponse(t, bridgeSocket, bridgePeer, [2]byte{0x33, 0x34}, downlinkACK)
	readIntegrationPacket(t, bridgeSocket, semtechTxAck)
	waitForEventType(t, engine.Events(), contracts.UplinkACKReceived)
	metrics := engine.Snapshot().Metrics
	if metrics.TotalUplinks != 1 || metrics.SuccessfulUplinks != 1 || metrics.AcknowledgedUplinks != 1 || metrics.PacketSuccessRate != 1 {
		t.Fatalf("external ACK did not complete the confirmed uplink: %+v", metrics)
	}
}

func TestEngineRealGatewayOTAAOverSemtechUDP(t *testing.T) {
	for _, counter := range []uint32{0, 491697203, ^uint32(0) - 1000} {
		t.Run(strconv.FormatUint(uint64(counter), 10), func(t *testing.T) {
			gatewaySocket := listenIntegrationUDP(t)
			gatewayAddress := gatewaySocket.LocalAddr().(*net.UDPAddr)
			enginePort := reserveIntegrationUDPPort(t)
			factory := gateway.NewUDPFactory(gateway.UDPOptions{
				LocalAddress:      net.JoinHostPort("127.0.0.1", strconv.Itoa(enginePort)),
				HeartbeatInterval: time.Second,
				Timeout:           3 * time.Second,
			})

			device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000092")
			gatewayConfig := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c92")
			gatewayConfig.Type = contracts.Real
			gatewayConfig.KeepAlive = nil
			gatewayConfig.GatewayIPv4 = stringPointer("127.0.0.1")
			gatewayConfig.GatewayPort = int32Pointer(gatewayAddress.Port)
			clock := types.NewManualClock(0)
			engine, err := New(
				contracts.SimulationConfig{Speed: 1},
				[]contracts.Device{device},
				[]contracts.Gateway{gatewayConfig},
				types.Options{Clock: clock, EventBuffer: 256, GatewayAdapterFactory: factory},
			)
			if err != nil {
				t.Fatalf("create real gateway engine: %v", err)
			}
			startEngine(t, engine)
			defer stopEngine(t, engine)

			engineAddress := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: enginePort}
			eui, err := hex.DecodeString(gatewayConfig.GatewayEUI)
			if err != nil {
				t.Fatalf("decode gateway EUI: %v", err)
			}
			writeIntegrationPacket(t, gatewaySocket, engineAddress, semtechPullData, [2]byte{0x41, 0x42}, eui)
			readIntegrationPacket(t, gatewaySocket, semtechPullAck)
			waitForEventType(t, engine.Events(), contracts.GatewayConnected)

			joinRequest, err := lorawan.BuildJoinRequest(lorawan.JoinRequestOptions{
				JoinEUI:  parseIntegrationEUI(t, device.OOTAConfig.JoinEUI),
				DevEUI:   parseIntegrationEUI(t, device.DevEUI),
				DevNonce: 7,
				AppKey:   mustAppKey(device),
			})
			if err != nil {
				t.Fatalf("build real gateway Join Request: %v", err)
			}
			writeIntegrationPushData(t, gatewaySocket, engineAddress, [2]byte{0x43, 0x44}, eui, joinRequest, counter)
			readIntegrationPacket(t, gatewaySocket, semtechPushAck)

			pullResponse := readIntegrationPacket(t, gatewaySocket, semtechPullResp)
			var response semtechPullResponse
			if err := json.Unmarshal(pullResponse.body, &response); err != nil {
				t.Fatalf("decode real gateway Join-Accept response: %v", err)
			}
			want := counter + uint32(5_000_000)
			if response.TXPK.Timestamp == nil || *response.TXPK.Timestamp != want || response.TXPK.Immediate || response.TXPK.Time != "" || !response.TXPK.InvertPolarity {
				t.Fatalf("physical gateway would reject Join-Accept scheduling: %+v want tmst=%d", response.TXPK, want)
			}
			joinAccept, err := base64.StdEncoding.DecodeString(response.TXPK.Data)
			if err != nil {
				t.Fatalf("decode real gateway Join-Accept payload: %v", err)
			}
			decodedJoinAccept, err := lorawan.DecodeJoinAccept(joinAccept, mustAppKey(device))
			if err != nil || decodedJoinAccept.DevAddr == 0 {
				t.Fatalf("real gateway received an invalid Join-Accept: %+v error=%v", decodedJoinAccept, err)
			}
			waitForEventType(t, engine.Events(), contracts.JoinAcceptReceived)
			nwkSKey, appSKey, err := lorawan.DeriveSessionKeys(mustAppKey(device), decodedJoinAccept.AppNonce, decodedJoinAccept.NetID, 7)
			if err != nil {
				t.Fatal(err)
			}
			port := byte(1)
			uplink, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{DevAddr: decodedJoinAccept.DevAddr, FCnt: 1, FPort: &port, Payload: []byte("after join"), Confirmed: true, NwkSKey: nwkSKey, AppSKey: appSKey})
			if err != nil {
				t.Fatal(err)
			}
			uplinkCounter := counter + uint32(10_000_000)
			writeIntegrationPushData(t, gatewaySocket, engineAddress, [2]byte{0x45, 0x46}, eui, uplink, uplinkCounter)
			readIntegrationPacket(t, gatewaySocket, semtechPushAck)
			ackResponse := readIntegrationPacket(t, gatewaySocket, semtechPullResp)
			var ack semtechPullResponse
			if err := json.Unmarshal(ackResponse.body, &ack); err != nil {
				t.Fatal(err)
			}
			if ack.TXPK.Timestamp == nil || *ack.TXPK.Timestamp != uplinkCounter+uint32(1_000_000) || ack.TXPK.Time != "" || !ack.TXPK.InvertPolarity {
				t.Fatalf("invalid physical ACK schedule: %+v", ack.TXPK)
			}
			ackBytes, err := base64.StdEncoding.DecodeString(ack.TXPK.Data)
			if err != nil {
				t.Fatal(err)
			}
			decodedACK, err := lorawan.Parse(ackBytes)
			if err != nil || !decodedACK.ACK || !lorawan.VerifyDataMIC(decodedACK, nwkSKey, 1) {
				t.Fatalf("physical node would reject ACK: %+v %v", decodedACK, err)
			}

		})
	}
}

func TestEngineObservesRawLoRaUplinkOverSemtechUDP(t *testing.T) {
	gatewaySocket := listenIntegrationUDP(t)
	gatewayAddress := gatewaySocket.LocalAddr().(*net.UDPAddr)
	enginePort := reserveIntegrationUDPPort(t)
	factory := gateway.NewUDPFactory(gateway.UDPOptions{
		LocalAddress: net.JoinHostPort("127.0.0.1", strconv.Itoa(enginePort)),
		Timeout:      3 * time.Second,
	})

	gatewayConfig := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c96")
	gatewayConfig.Type = contracts.Real
	gatewayConfig.KeepAlive = nil
	gatewayConfig.GatewayIPv4 = stringPointer("127.0.0.1")
	gatewayConfig.GatewayPort = int32Pointer(gatewayAddress.Port)
	engine, err := New(
		contracts.SimulationConfig{Speed: 1},
		nil,
		[]contracts.Gateway{gatewayConfig},
		types.Options{EventBuffer: 256, GatewayAdapterFactory: factory},
	)
	if err != nil {
		t.Fatalf("create raw LoRa gateway engine: %v", err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)

	engineAddress := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: enginePort}
	eui, err := hex.DecodeString(gatewayConfig.GatewayEUI)
	if err != nil {
		t.Fatalf("decode gateway EUI: %v", err)
	}
	writeIntegrationPacket(t, gatewaySocket, engineAddress, semtechPullData, [2]byte{0x51, 0x52}, eui)
	readIntegrationPacket(t, gatewaySocket, semtechPullAck)
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)

	payload := []byte{0xde, 0xad, 0xbe, 0xef}
	writeIntegrationPushData(t, gatewaySocket, engineAddress, [2]byte{0x53, 0x54}, eui, payload)
	readIntegrationPacket(t, gatewaySocket, semtechPushAck)
	event := waitForEventType(t, engine.Events(), contracts.GatewayPacketIngress)
	if event.RadioFrame == nil {
		t.Fatal("raw LoRa UDP frame was not exposed on the ingress event")
	}
	if event.RadioFrame.Protocol != contracts.RawLoRa || string(event.RadioFrame.Payload) != string(payload) {
		t.Fatalf("unexpected raw LoRa frame: %+v", event.RadioFrame)
	}
	if event.RadioFrame.SenderID != gatewayConfig.ID {
		t.Fatalf("unaddressed raw frame was attributed to %q, want gateway %q", event.RadioFrame.SenderID, gatewayConfig.ID)
	}
	if event.DeviceID != nil {
		t.Fatalf("unaddressed Semtech frame was assigned to a device: %q", *event.DeviceID)
	}
}

func TestEngineRawLoRaBridgePreservesPayloadAndObservesUnaddressedDownlink(t *testing.T) {
	bridgeSocket := listenIntegrationUDP(t)
	address := bridgeSocket.LocalAddr().(*net.UDPAddr)
	factory := gateway.NewUDPFactory(gateway.UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: 3 * time.Second})
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: int32(address.Port)}); err != nil {
		t.Fatal(err)
	}
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000707")
	protocol := contracts.RawLoRa
	device.RadioProtocol = &protocol
	device.RadioReceiveMode = receiveModePointer(contracts.ReceiveAuto)
	device.PayloadConfig.Payload = "3q2+7w=="
	device.PayloadConfig.Base64Encoded = true
	gatewayConfig := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c97")
	clock := types.NewManualClock(0)
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gatewayConfig}, types.Options{Clock: clock, EventBuffer: 256, GatewayAdapterFactory: factory})
	if err != nil {
		t.Fatal(err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	keepalive := readIntegrationPacket(t, bridgeSocket, semtechPullData)
	writeIntegrationPacket(t, bridgeSocket, keepalive.remote, semtechPullAck, keepalive.token, nil)
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)
	if err := clock.Advance(time.Second); err != nil {
		t.Fatal(err)
	}
	uplink := readIntegrationPacket(t, bridgeSocket, semtechPushData)
	payload := decodeIntegrationUplink(t, uplink.body, gatewayConfig.GatewayEUI)
	if string(payload) != string([]byte{0xde, 0xad, 0xbe, 0xef}) {
		t.Fatalf("bridge altered raw payload: %x", payload)
	}
	writeIntegrationPullResponse(t, bridgeSocket, keepalive.remote, [2]byte{0x61, 0x62}, payload)
	readIntegrationPacket(t, bridgeSocket, semtechTxAck)
	observed := waitForEventType(t, engine.Events(), contracts.GatewayPacketIngress)
	if observed.RadioFrame == nil || observed.RadioFrame.Protocol != contracts.RawLoRa || string(observed.RadioFrame.Payload) != string(payload) {
		t.Fatalf("raw bridge response was not recorded: %+v", observed)
	}
	if observed.DeviceID != nil {
		t.Fatal("unaddressed UDP downlink must not be assigned to an arbitrary device")
	}
	if observed.RadioFrame.TimestampMilliseconds != observed.TimestampMilliseconds {
		t.Fatal("raw bridge response timestamp must use simulation time")
	}
}

func TestEngineABPDownlinkOverSemtechUDP(t *testing.T) {
	t.Run("bridge", func(t *testing.T) {
		bridgeSocket := listenIntegrationUDP(t)
		bridgeAddress := bridgeSocket.LocalAddr().(*net.UDPAddr)
		factory := gateway.NewUDPFactory(gateway.UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: 3 * time.Second})
		if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{
			Address: "127.0.0.1", Enabled: true, Port: int32(bridgeAddress.Port),
		}); err != nil {
			t.Fatalf("configure UDP bridge: %v", err)
		}
		device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000093")
		configureABP(&device)
		gatewayConfig := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c93")
		engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gatewayConfig}, types.Options{GatewayAdapterFactory: factory})
		if err != nil {
			t.Fatalf("create bridge ABP engine: %v", err)
		}
		startEngine(t, engine)
		defer stopEngine(t, engine)

		keepalive := readIntegrationPacket(t, bridgeSocket, semtechPullData)
		writeIntegrationPacket(t, bridgeSocket, keepalive.remote, semtechPullAck, keepalive.token, nil)
		waitForEventType(t, engine.Events(), contracts.GatewayConnected)
		downlink := buildIntegrationABPDownlink(t, device)
		writeIntegrationPullResponse(t, bridgeSocket, keepalive.remote, [2]byte{0x51, 0x52}, downlink)
		readIntegrationPacket(t, bridgeSocket, semtechTxAck)
		event := waitForEventType(t, engine.Events(), contracts.DeviceDownlinkTransmitted)
		if event.DeviceID == nil || *event.DeviceID != device.ID {
			t.Fatalf("bridge downlink was not applied to the expected device: %+v", event)
		}
	})

	t.Run("real-gateway", func(t *testing.T) {
		gatewaySocket := listenIntegrationUDP(t)
		gatewayAddress := gatewaySocket.LocalAddr().(*net.UDPAddr)
		enginePort := reserveIntegrationUDPPort(t)
		factory := gateway.NewUDPFactory(gateway.UDPOptions{LocalAddress: net.JoinHostPort("127.0.0.1", strconv.Itoa(enginePort)), Timeout: 3 * time.Second})
		device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000094")
		configureABP(&device)
		gatewayConfig := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c94")
		gatewayConfig.Type = contracts.Real
		gatewayConfig.KeepAlive = nil
		gatewayConfig.GatewayIPv4 = stringPointer("127.0.0.1")
		gatewayConfig.GatewayPort = int32Pointer(gatewayAddress.Port)
		engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gatewayConfig}, types.Options{GatewayAdapterFactory: factory})
		if err != nil {
			t.Fatalf("create real gateway ABP engine: %v", err)
		}
		startEngine(t, engine)
		defer stopEngine(t, engine)

		engineAddress := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: enginePort}
		eui, err := hex.DecodeString(gatewayConfig.GatewayEUI)
		if err != nil {
			t.Fatalf("decode gateway EUI: %v", err)
		}
		writeIntegrationPacket(t, gatewaySocket, engineAddress, semtechPullData, [2]byte{0x53, 0x54}, eui)
		readIntegrationPacket(t, gatewaySocket, semtechPullAck)
		waitForEventType(t, engine.Events(), contracts.GatewayConnected)
		downlink := buildIntegrationABPDownlink(t, device)
		writeIntegrationPullResponse(t, gatewaySocket, engineAddress, [2]byte{0x55, 0x56}, downlink)
		readIntegrationPacket(t, gatewaySocket, semtechTxAck)
		event := waitForEventType(t, engine.Events(), contracts.DeviceDownlinkTransmitted)
		if event.DeviceID == nil || *event.DeviceID != device.ID {
			t.Fatalf("real gateway downlink was not applied to the expected device: %+v", event)
		}
	})
}

func buildIntegrationABPDownlink(t *testing.T, device contracts.Device) []byte {
	t.Helper()
	nwkSKey, err := lorawan.DecodeHex(device.ABPConfig.NwkSKey)
	if err != nil {
		t.Fatalf("decode ABP network session key: %v", err)
	}
	appSKey, err := lorawan.DecodeHex(device.ABPConfig.AppSKey)
	if err != nil {
		t.Fatalf("decode ABP application session key: %v", err)
	}
	fPort := byte(1)
	frame, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr:   0x26011BDA,
		FCnt:      0,
		FPort:     &fPort,
		Payload:   []byte("server command"),
		Direction: 1,
		NwkSKey:   nwkSKey,
		AppSKey:   appSKey,
	})
	if err != nil {
		t.Fatalf("build ABP downlink: %v", err)
	}
	return frame
}

type integrationPacket struct {
	token  [2]byte
	remote *net.UDPAddr
	body   []byte
}

func listenIntegrationUDP(t *testing.T) *net.UDPConn {
	t.Helper()
	socket, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatalf("listen integration UDP socket: %v", err)
	}
	t.Cleanup(func() { _ = socket.Close() })
	return socket
}

func reserveIntegrationUDPPort(t *testing.T) int {
	t.Helper()
	socket := listenIntegrationUDP(t)
	port := socket.LocalAddr().(*net.UDPAddr).Port
	if err := socket.Close(); err != nil {
		t.Fatalf("release integration UDP port: %v", err)
	}
	return port
}

func readIntegrationPacket(t *testing.T, socket *net.UDPConn, expectedType byte) integrationPacket {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	if err := socket.SetReadDeadline(deadline); err != nil {
		t.Fatalf("set integration UDP read deadline: %v", err)
	}
	buffer := make([]byte, 64*1024)
	for {
		size, remote, err := socket.ReadFromUDP(buffer)
		if err != nil {
			t.Fatalf("read Semtech UDP packet type 0x%02x: %v", expectedType, err)
		}
		if size < 4 || buffer[0] != semtechVersion || buffer[3] != expectedType {
			continue
		}
		return integrationPacket{
			token:  [2]byte{buffer[1], buffer[2]},
			remote: remote,
			body:   append([]byte(nil), buffer[4:size]...),
		}
	}
}

func writeIntegrationPacket(t *testing.T, socket *net.UDPConn, remote *net.UDPAddr, packetType byte, token [2]byte, body []byte) {
	t.Helper()
	packet := []byte{semtechVersion, token[0], token[1], packetType}
	packet = append(packet, body...)
	if _, err := socket.WriteToUDP(packet, remote); err != nil {
		t.Fatalf("write Semtech UDP packet type 0x%02x: %v", packetType, err)
	}
}

func writeIntegrationPushData(t *testing.T, socket *net.UDPConn, remote *net.UDPAddr, token [2]byte, eui, payload []byte, counters ...uint32) {
	t.Helper()
	counter := uint32(491697203)
	if len(counters) > 0 {
		counter = counters[0]
	}
	body, err := json.Marshal(semtechPushDataPayload{RXPK: []semtechRXPacket{{
		Timestamp: &counter, Data: base64.StdEncoding.EncodeToString(payload), Frequency: 868.1, DataRate: "SF7BW125", Modulation: "LORA", Size: len(payload),
	}}})
	if err != nil {
		t.Fatalf("encode PUSH_DATA payload: %v", err)
	}
	packet := []byte{semtechVersion, token[0], token[1], semtechPushData}
	packet = append(packet, eui...)
	packet = append(packet, body...)
	if _, err := socket.WriteToUDP(packet, remote); err != nil {
		t.Fatalf("write PUSH_DATA packet: %v", err)
	}
}

func writeIntegrationPullResponse(t *testing.T, socket *net.UDPConn, remote *net.UDPAddr, token [2]byte, payload []byte) {
	t.Helper()
	body, err := json.Marshal(semtechPullResponse{TXPK: semtechTXPacket{
		Immediate: true, Frequency: 869.525, Modulation: "LORA", DataRate: "SF9BW125", CodingRate: "4/5",
		Size: len(payload), Data: base64.StdEncoding.EncodeToString(payload),
	}})
	if err != nil {
		t.Fatalf("encode PULL_RESP payload: %v", err)
	}
	writeIntegrationPacket(t, socket, remote, semtechPullResp, token, body)
}

func decodeIntegrationUplink(t *testing.T, body []byte, expectedEUI string) []byte {
	t.Helper()
	if len(body) < 8 {
		t.Fatalf("PUSH_DATA body is missing the gateway EUI")
	}
	if got := strings.ToUpper(hex.EncodeToString(body[:8])); got != strings.ToUpper(expectedEUI) {
		t.Fatalf("unexpected gateway EUI in PUSH_DATA: got %s want %s", got, expectedEUI)
	}
	var payload semtechPushDataPayload
	if err := json.Unmarshal(body[8:], &payload); err != nil {
		t.Fatalf("decode PUSH_DATA payload: %v", err)
	}
	if len(payload.RXPK) != 1 {
		t.Fatalf("expected one RXPK entry, got %d", len(payload.RXPK))
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.RXPK[0].Data)
	if err != nil {
		t.Fatalf("decode PUSH_DATA frame: %v", err)
	}
	return decoded
}

func parseIntegrationEUI(t *testing.T, value string) uint64 {
	t.Helper()
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != 8 {
		t.Fatalf("invalid EUI %q: %v", value, err)
	}
	var result uint64
	for _, part := range decoded {
		result = result<<8 | uint64(part)
	}
	return result
}

func stringPointer(value string) *string { return &value }

func TestEngineExternalMACNegotiationOverSemtechUDP(t *testing.T) {
	bridge := listenIntegrationUDP(t)
	address := bridge.LocalAddr().(*net.UDPAddr)
	factory := gateway.NewUDPFactory(gateway.UDPOptions{LocalAddress: "127.0.0.1:0", Timeout: 3 * time.Second})
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: int32(address.Port)}); err != nil {
		t.Fatal(err)
	}
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000998")
	configureABP(&device)
	device.Class = contracts.ClassA
	device.PayloadConfig.MType = contracts.UnconfirmedDataUp
	gw := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7998")
	clock := types.NewManualClock(0)
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gw}, types.Options{GatewayAdapterFactory: factory, Clock: clock, EventBuffer: 512})
	if err != nil {
		t.Fatal(err)
	}
	commands := append(newChannelCommand(3, 867100000), 3, 0x51, 15, 0, 1)
	request := externalTestFrame(t, engine.sessions[device.ID], true, commands, 1)
	startEngine(t, engine)
	defer stopEngine(t, engine)
	keepalive := readIntegrationPacket(t, bridge, semtechPullData)
	writeIntegrationPacket(t, bridge, keepalive.remote, semtechPullAck, keepalive.token, nil)
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)
	if err := clock.Advance(2 * time.Second); err != nil {
		t.Fatal(err)
	}
	readIntegrationPacket(t, bridge, semtechPushData)
	writeIntegrationPullResponse(t, bridge, keepalive.remote, [2]byte{0x71, 0x72}, request.Raw)
	readIntegrationPacket(t, bridge, semtechTxAck)
	waitForEventType(t, engine.Events(), contracts.DeviceDownlinkTransmitted)
	if _, err := engine.QueueUplink(device.ID); err != nil {
		t.Fatal(err)
	}
	if err := clock.Advance(time.Second); err != nil {
		t.Fatal(err)
	}
	uplink := readIntegrationPacket(t, bridge, semtechPushData)
	wire := decodeIntegrationUplink(t, uplink.body, gw.GatewayEUI)
	response, err := lorawan.Parse(wire)
	if err != nil || !bytes.Equal(response.FOpts, []byte{7, 3, 3, 7}) {
		t.Fatalf("MAC responses did not reach UDP bridge: %x %v", response.FOpts, err)
	}
}

func TestReactivatingVirtualGatewayRestartsBridgeTransport(t *testing.T) {
	bridge := listenIntegrationUDP(t)
	address := bridge.LocalAddr().(*net.UDPAddr)
	factory := gateway.NewUDPFactory(gateway.UDPOptions{LocalAddress: "127.0.0.1:0"})
	if err := factory.ConfigureGatewayBridge(contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: int32(address.Port)}); err != nil {
		t.Fatal(err)
	}
	gw := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7999")
	engine, err := New(contracts.SimulationConfig{Speed: 1}, nil, []contracts.Gateway{gw}, types.Options{GatewayAdapterFactory: factory})
	if err != nil {
		t.Fatal(err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	connect := func() {
		t.Helper()
		pull := readIntegrationPacket(t, bridge, semtechPullData)
		writeIntegrationPacket(t, bridge, pull.remote, semtechPullAck, pull.token, nil)
		waitForEventType(t, engine.Events(), contracts.GatewayConnected)
	}
	connect()
	gw.Active = false
	if err := engine.UpdateGateway(gw); err != nil {
		t.Fatal(err)
	}
	gw.Active = true
	if err := engine.UpdateGateway(gw); err != nil {
		t.Fatal(err)
	}
	connect()
}

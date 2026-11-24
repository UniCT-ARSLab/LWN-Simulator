package runtime

import (
	"encoding/hex"
	"net"
	"strconv"
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/gateway"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestUDPNetworkRecognizesLoRaWANRunsCodecAndPreservesRaw(t *testing.T) {
	peer := listenIntegrationUDP(t)
	port := reserveIntegrationUDPPort(t)
	config := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c13")
	config.Type = contracts.Real
	config.KeepAlive = nil
	config.GatewayIPv4 = stringPointer("127.0.0.1")
	config.GatewayPort = int32Pointer(peer.LocalAddr().(*net.UDPAddr).Port)
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000213")
	configureABP(&device)
	device.RadioReceiveMode = receiveModePointer(contracts.ReceiveAuto)
	device.PayloadConfig.PayloadCodec = &contracts.PayloadCodec{Enabled: true, Script: `const decodeUplink = input => ({ data: { temperature: input.bytes[0], humidity: input.bytes[1], sum: input.bytes.reduce((total, value) => total + value, 0), port: input.fPort } });`}
	factory := gateway.NewUDPFactory(gateway.UDPOptions{LocalAddress: net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), Timeout: 5 * time.Second})
	engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{config}, types.Options{Clock: types.NewManualClock(0), EventBuffer: 512, GatewayAdapterFactory: factory})
	if err != nil {
		t.Fatal(err)
	}
	startEngine(t, engine)
	defer stopEngine(t, engine)
	remote := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: port}
	eui, _ := hex.DecodeString(config.GatewayEUI)
	writeIntegrationPacket(t, peer, remote, semtechPullData, [2]byte{1, 1}, eui)
	readIntegrationPacket(t, peer, semtechPullAck)
	waitForEventType(t, engine.Events(), contracts.GatewayConnected)
	session := newDeviceSession(device)
	fPort := byte(1)
	frame, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{DevAddr: 0x26011bda, FCnt: 1, FPort: &fPort, Payload: []byte{21, 50}, NwkSKey: session.NwkSKey, AppSKey: session.AppSKey})
	if err != nil {
		t.Fatal(err)
	}
	writeIntegrationPushData(t, peer, remote, [2]byte{2, 1}, eui, frame)
	readIntegrationPacket(t, peer, semtechPushAck)
	event := waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	for event.GatewayID == nil || *event.GatewayID != config.ID {
		event = waitForEventType(t, engine.Events(), contracts.DeviceUplinkTransmitted)
	}
	if event.DecodedPayload == nil || (*event.DecodedPayload)["temperature"] != float64(21) || (*event.DecodedPayload)["humidity"] != float64(50) || (*event.DecodedPayload)["sum"] != float64(71) {
		t.Fatalf("network did not classify, decrypt, and decode LoRaWAN content: %+v", event)
	}
	frame[len(frame)-1] ^= 0xff
	writeIntegrationPushData(t, peer, remote, [2]byte{3, 1}, eui, frame)
	readIntegrationPacket(t, peer, semtechPushAck)
	dropped := waitForEventType(t, engine.Events(), contracts.PacketDropped)
	if dropped.DecodedPayload != nil {
		t.Fatal("invalid MIC was passed to the application codec")
	}
	raw := []byte("Hello LoRa world")
	writeIntegrationPushData(t, peer, remote, [2]byte{4, 1}, eui, raw)
	readIntegrationPacket(t, peer, semtechPushAck)
	observed := waitForEventType(t, engine.Events(), contracts.GatewayPacketIngress)
	if observed.RadioFrame == nil || observed.RadioFrame.Protocol != contracts.RawLoRa || string(observed.RadioFrame.Payload) != string(raw) || observed.DecodedPayload != nil {
		t.Fatalf("non-LoRaWAN content was not preserved as raw: %+v", observed)
	}
}

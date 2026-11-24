package runtime

import (
	"bytes"
	"fmt"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestAddressedGatewayFramesRespectReceiveMode(t *testing.T) {
	for _, mode := range []contracts.RadioReceiveMode{contracts.ReceiveRawLoRa, contracts.ReceiveLoRaWAN, contracts.ReceiveAuto} {
		for _, downlink := range []bool{false, true} {
			for _, frameType := range []string{"raw", "valid-lorawan", "invalid-mic", "empty"} {
				validFrame := frameType == "valid-lorawan" || frameType == "invalid-mic"
				t.Run(fmt.Sprintf("%s/downlink=%v/frame=%s", mode, downlink, frameType), func(t *testing.T) {
					device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000901")
					configureABP(&device)
					device.RadioReceiveMode = &mode
					gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7901")
					engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{EventBuffer: 64})
					if err != nil {
						t.Fatal(err)
					}
					payload := []byte{0, 255, 128, 1}
					kind := types.GatewayPacketUplink
					if downlink {
						kind = types.GatewayPacketDownlink
					}
					if validFrame {
						session := engine.sessions[device.ID]
						direction := byte(0)
						if downlink {
							direction = 1
						}
						port := byte(1)
						payload, err = lorawan.BuildDataFrame(lorawan.DataFrameOptions{FPort: &port, DevAddr: 0x26011bda, FCnt: 1, Direction: direction, Payload: []byte("test"), NwkSKey: session.NwkSKey, AppSKey: session.AppSKey})
						if err != nil {
							t.Fatal(err)
						}
					}
					if frameType == "invalid-mic" {
						payload[len(payload)-1] ^= 0xff
					}
					if frameType == "empty" {
						payload = nil
					}
					engine.handleGatewayPacket(types.GatewayPacket{GatewayID: gateway.ID, DeviceID: device.ID, Kind: kind, Payload: payload})
					rawExpected := frameType != "empty" && (mode == contracts.ReceiveRawLoRa || (mode == contracts.ReceiveAuto && !validFrame))
					success := contracts.DeviceUplinkTransmitted
					if downlink {
						success = contracts.DeviceDownlinkTransmitted
					}
					found := false
					for _, event := range engine.EventLog() {
						if event.Type != success {
							continue
						}
						found = true
						if rawExpected && (event.RadioFrame == nil || event.RadioFrame.Protocol != contracts.RawLoRa || !bytes.Equal(event.RadioFrame.Payload, payload)) {
							t.Fatalf("raw bytes changed: %+v", event)
						}
						if rawExpected && downlink && (event.FPort != nil || event.FrameCounter != nil || event.Confirmed != nil) {
							t.Fatal("raw reception inherited LoRaWAN metadata")
						}
					}
					expected := rawExpected || frameType == "valid-lorawan"
					if found != expected {
						t.Fatalf("unexpected delivery: mode=%s valid=%v events=%+v", mode, validFrame, engine.EventLog())
					}
				})
			}
		}
	}
}

func TestExternalJoinAcceptRespectsReceiveMode(t *testing.T) {
	for _, mode := range []contracts.RadioReceiveMode{contracts.ReceiveRawLoRa, contracts.ReceiveLoRaWAN, contracts.ReceiveAuto} {
		t.Run(string(mode), func(t *testing.T) {
			device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000902")
			device.RadioReceiveMode = &mode
			gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7902")
			engine, err := New(contracts.SimulationConfig{Speed: 1}, []contracts.Device{device}, []contracts.Gateway{gateway}, types.Options{})
			if err != nil {
				t.Fatal(err)
			}
			engine.sessions[device.ID].PendingJoinRequest = &types.PendingJoinRequest{PacketID: "join", GatewayIDs: []string{gateway.ID}}
			key, _ := lorawan.DecodeHex(device.OOTAConfig.AppKey)
			frame, err := lorawan.BuildJoinAccept(lorawan.JoinAcceptOptions{AppKey: key, DevAddr: 0x26011bda})
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := lorawan.Parse(frame)
			if err != nil {
				t.Fatal(err)
			}
			engine.processExternalJoinAcceptLocked(types.GatewayPacket{GatewayID: gateway.ID, Payload: frame}, decoded)
			if engine.sessions[device.ID].Joined != (mode != contracts.ReceiveRawLoRa) {
				t.Fatal("join ignored receive mode")
			}
		})
	}
}

func TestGatewayProtocolHintsAreStrict(t *testing.T) {
	for _, hint := range []contracts.RadioProtocol{"", contracts.RawLoRa, contracts.LoRaWAN, "unknown"} {
		t.Run(string(hint), func(t *testing.T) {
			packet := types.GatewayPacket{Protocol: hint, Payload: []byte("Hello LoRa world")}
			classified, err := classifyGatewayPacket(packet)
			if hint == "" || hint == contracts.RawLoRa {
				if err != nil || classified.Protocol != contracts.RawLoRa {
					t.Fatalf("raw bytes were lost: %+v %v", classified, err)
				}
			} else if err == nil {
				t.Fatal("explicit invalid protocol was silently accepted")
			}
		})
	}
}

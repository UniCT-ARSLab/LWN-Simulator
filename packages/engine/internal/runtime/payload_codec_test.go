package runtime

import (
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/lorawan"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestRealGatewayPayloadIsDecodedAfterDecryption(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000210")
	configureABP(&device)
	device.PayloadConfig.PayloadCodec = &contracts.PayloadCodec{
		Enabled: true,
		Script:  "function decodeUplink(input) { return { data: { first: input.bytes[0] } }; }",
	}
	session := newDeviceSession(device)
	port := byte(device.FrameConfig.FPort)
	frame, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{
		DevAddr:   0x26011bda,
		FCnt:      1,
		FPort:     &port,
		Payload:   []byte{42},
		NwkSKey:   session.NwkSKey,
		AppSKey:   session.AppSKey,
		Direction: 0,
	})
	if err != nil {
		t.Fatalf("build uplink: %v", err)
	}
	event := contracts.SimulationEvent{}
	decorateDecodedRadioPayload(&event, device, &types.RadioTransmission{
		RadioFrame: types.RadioFrame{Payload: frame},
	}, session)
	if event.DecodedPayload == nil || (*event.DecodedPayload)["first"] != float64(42) {
		t.Fatalf("unexpected decoded real-gateway payload: %+v", event.DecodedPayload)
	}
}

func TestSimulationPayloadCodecFromTransmissionToEvent(t *testing.T) {
	for _, activation := range []contracts.DeviceActivation{contracts.ABP, contracts.OOTA} {
		for _, failing := range []bool{false, true} {
			name := string(activation) + "/decoded"
			if failing {
				name = string(activation) + "/codec-error"
			}
			t.Run(name, func(t *testing.T) {
				device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000212")
				if activation == contracts.ABP {
					configureABP(&device)
				}
				device.PayloadConfig.Payload = "FTI="
				device.PayloadConfig.Base64Encoded = true
				script := `const decodeUplink = input => ({ data: { temperature: input.bytes[0], humidity: input.bytes[1], port: input.fPort } });`
				if failing {
					script = `function decodeUplink() { throw new Error('demonstration failure'); }`
				}
				device.PayloadConfig.PayloadCodec = &contracts.PayloadCodec{Enabled: true, Script: script}
				clock := types.NewManualClock(0)
				engine := newTestEngine(t, clock, device, validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c12"))
				defer stopEngine(t, engine)
				if err := clock.Advance(time.Minute); err != nil {
					t.Fatal(err)
				}
				event := waitForEventType(t, engine.Events(), contracts.GatewayPacketReceived)
				if failing {
					if event.Error == nil || event.DecodedPayload != nil {
						t.Fatalf("codec failure was not attached to the event: %+v", event)
					}
				} else if event.DecodedPayload == nil || (*event.DecodedPayload)["temperature"] != float64(21) || (*event.DecodedPayload)["humidity"] != float64(50) || (*event.DecodedPayload)["port"] != float64(1) {
					t.Fatalf("application bytes were not decoded after the radio transmission: %+v", event)
				}
				if engine.Snapshot().State.Status != contracts.SimulationStatusRunning {
					t.Fatal("codec execution stopped the simulation")
				}
			})
		}
	}
}

func TestRawBytesAreNotPassedToLoRaWANCodec(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000211")
	configureABP(&device)
	device.PayloadConfig.PayloadCodec = &contracts.PayloadCodec{Enabled: true, Script: `function decodeUplink() { throw new Error('must not run for raw'); }`}
	session := newDeviceSession(device)
	port := byte(1)
	frame, err := lorawan.BuildDataFrame(lorawan.DataFrameOptions{DevAddr: 0x26011bda, FCnt: 1, FPort: &port, Payload: []byte{42}, NwkSKey: session.NwkSKey, AppSKey: session.AppSKey})
	if err != nil {
		t.Fatal(err)
	}
	event := contracts.SimulationEvent{}
	decorateDecodedRadioPayload(&event, device, &types.RadioTransmission{RadioFrame: types.RadioFrame{Protocol: contracts.RawLoRa, Payload: frame}}, session)
	if event.Error != nil || event.DecodedPayload != nil {
		t.Fatalf("explicit raw frame was interpreted as LoRaWAN: %+v", event)
	}
}

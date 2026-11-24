package runtime

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/payloadcodec"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func decorateDecodedPayload(event *contracts.SimulationEvent, device contracts.Device, payload []byte, fPort int) {
	codec := device.PayloadConfig.PayloadCodec
	if codec == nil || !codec.Enabled || fPort < 1 || len(payload) == 0 {
		return
	}

	decoded, err := payloadcodec.Decode(codec.Script, payloadcodec.Input{Bytes: payload, FPort: fPort})
	if err != nil {
		message := err.Error()
		event.Error = &message
		return
	}
	event.DecodedPayload = &decoded
}

func decorateDecodedRadioPayload(event *contracts.SimulationEvent, device contracts.Device, transmission *types.RadioTransmission, session *types.DeviceSession) {
	if session == nil || transmission.Protocol == contracts.RawLoRa {
		return
	}
	packet, err := lorawanHandler.Decode(transmission.Payload)
	if err != nil || packet.FPort == nil || len(packet.FRMPayload) == 0 {
		return
	}
	key := session.AppSKey
	if *packet.FPort == 0 {
		key = session.NwkSKey
	}
	payload, err := lorawanHandler.DecryptPayload(packet, key, 0)
	if err != nil {
		message := err.Error()
		event.Error = &message
		return
	}
	decorateDecodedPayload(event, device, payload, int(*packet.FPort))
}

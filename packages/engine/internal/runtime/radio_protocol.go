package runtime

import contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"

func deviceRadioProtocol(device contracts.Device) contracts.RadioProtocol {
	if device.RadioProtocol == nil || *device.RadioProtocol == "" {
		return contracts.LoRaWAN
	}
	return *device.RadioProtocol
}

func deviceReceiveMode(device contracts.Device) contracts.RadioReceiveMode {
	if device.RadioReceiveMode == nil || *device.RadioReceiveMode == "" {
		return contracts.ReceiveLoRaWAN
	}
	return *device.RadioReceiveMode
}

func acceptsRadioProtocol(device contracts.Device, protocol contracts.RadioProtocol) bool {
	mode := deviceReceiveMode(device)
	return mode == contracts.ReceiveAuto || mode == contracts.RadioReceiveMode(protocol)
}

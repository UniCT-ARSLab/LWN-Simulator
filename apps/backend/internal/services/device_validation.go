package services

import (
	"fmt"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine"
)

const validationDeviceID = "00000000-0000-4000-8000-000000000000"

func ValidateDeviceRequest(req contracts.CreateDeviceRequest) error {
	return ValidateDevice(contracts.Device{
		ID: validationDeviceID, Active: true, Name: req.Name, DevEUI: req.DevEUI,
		Class: req.Class, Activation: req.Activation, OOTAConfig: req.OOTAConfig,
		RadioProtocol: req.RadioProtocol, RadioReceiveMode: req.RadioReceiveMode,
		ABPConfig: req.ABPConfig, LocationConfig: req.LocationConfig,
		RX1Config: req.RX1Config, RX2Config: req.RX2Config,
		FrameConfig: req.FrameConfig, PayloadConfig: req.PayloadConfig,
		AdvancedConfig: req.AdvancedConfig,
	})
}

func ValidateDevice(device contracts.Device) error {
	if device.RadioProtocol != nil && !device.RadioProtocol.Valid() {
		return fmt.Errorf("unsupported radio protocol %q", *device.RadioProtocol)
	}
	if device.RadioReceiveMode != nil && !device.RadioReceiveMode.Valid() {
		return fmt.Errorf("unsupported radio receive mode %q", *device.RadioReceiveMode)
	}
	return engine.ValidateHardware([]contracts.Device{device}, nil)
}

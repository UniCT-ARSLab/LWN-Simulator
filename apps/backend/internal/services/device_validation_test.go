package services

import (
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestValidateDeviceRejectsInvalidSemanticConfiguration(t *testing.T) {
	device := validSimulationDevice(testDeviceID1)
	if err := ValidateDevice(device); err != nil {
		t.Fatalf("valid device rejected: %v", err)
	}
	device.Activation = contracts.OOTA
	if err := ValidateDevice(device); err == nil {
		t.Fatal("OTAA device with ABP credentials should be rejected")
	}
	device = validSimulationDevice(testDeviceID1)
	device.PayloadConfig.MType = "invalid"
	if err := ValidateDevice(device); err == nil {
		t.Fatal("invalid message type should be rejected")
	}
}

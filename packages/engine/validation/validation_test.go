package validation

import (
	"math"
	"strings"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestValidateSimulationConfigAcceptsFinitePositiveSpeed(t *testing.T) {
	for _, speed := range []float64{0.01, 1, 100} {
		if err := ValidateSimulationConfig(contracts.SimulationConfig{Speed: speed}); err != nil {
			t.Fatalf("speed %v should be valid: %v", speed, err)
		}
	}
}

func TestValidateSimulationConfigRejectsNonFiniteOrNonPositiveSpeed(t *testing.T) {
	for _, speed := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if err := ValidateSimulationConfig(contracts.SimulationConfig{Speed: speed}); err == nil {
			t.Fatalf("speed %v should be rejected", speed)
		}
	}
}

func TestValidateSimulationConfigValidatesGatewayBridgeAddress(t *testing.T) {
	if err := ValidateSimulationConfig(contracts.SimulationConfig{
		Speed: 1,
		GatewayBridge: &contracts.GatewayBridgeConfig{
			Enabled: true,
			Address: "127.0.0.1",
			Port:    1700,
		},
	}); err != nil {
		t.Fatalf("valid gateway bridge address was rejected: %v", err)
	}

	if err := ValidateSimulationConfig(contracts.SimulationConfig{
		Speed: 1,
		GatewayBridge: &contracts.GatewayBridgeConfig{
			Enabled: true,
			Address: "not/an-address",
			Port:    1700,
		},
	}); err == nil || !strings.Contains(err.Error(), "gatewayBridge.address") {
		t.Fatalf("invalid gateway bridge address was not rejected: %v", err)
	}
}

func TestGatewayBridgeParameterMatrix(t *testing.T) {
	tests := []struct {
		name      string
		config    contracts.GatewayBridgeConfig
		wantError string
	}{
		{name: "disabled ignores endpoint", config: contracts.GatewayBridgeConfig{Enabled: false, Address: "", Port: 0}},
		{name: "IPv4", config: contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: 1}},
		{name: "URL", config: contracts.GatewayBridgeConfig{Enabled: true, Address: "udp://localhost", Port: 65535}},
		{name: "missing address", config: contracts.GatewayBridgeConfig{Enabled: true, Port: 1700}, wantError: "gatewayBridge.address"},
		{name: "port below range", config: contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: 0}, wantError: "gatewayBridge.port"},
		{name: "port above range", config: contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: 65536}, wantError: "gatewayBridge.port"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateSimulationConfig(contracts.SimulationConfig{Speed: 1, GatewayBridge: &test.config})
			if test.wantError == "" && err != nil {
				t.Fatalf("valid bridge configuration was rejected: %v", err)
			}
			if test.wantError != "" && (err == nil || !strings.Contains(err.Error(), test.wantError)) {
				t.Fatalf("expected %s validation error, got %v", test.wantError, err)
			}
		})
	}
}

func TestValidateHardwareAcceptsValidFleet(t *testing.T) {
	if err := ValidateHardware([]contracts.Device{validationDevice()}, []contracts.Gateway{validationGateway()}); err != nil {
		t.Fatalf("valid fleet was rejected: %v", err)
	}
}

func TestValidateHardwareRejectsUnknownRadioSettings(t *testing.T) {
	device := validationDevice()
	protocol := contracts.RadioProtocol("unsupported")
	device.RadioProtocol = &protocol
	if err := ValidateHardware([]contracts.Device{device}, nil); err == nil || !strings.Contains(err.Error(), "radioProtocol") {
		t.Fatalf("invalid radio protocol was accepted: %v", err)
	}
	device.RadioProtocol = nil
	mode := contracts.RadioReceiveMode("unsupported")
	device.RadioReceiveMode = &mode
	if err := ValidateHardware([]contracts.Device{device}, nil); err == nil || !strings.Contains(err.Error(), "radioReceiveMode") {
		t.Fatalf("invalid radio receive mode was accepted: %v", err)
	}
}

func TestValidatePayloadCodec(t *testing.T) {
	tests := []struct {
		name      string
		codec     *contracts.PayloadCodec
		wantField string
		wantNoErr bool
	}{
		{
			name:      "disabled codec ignores empty script",
			codec:     &contracts.PayloadCodec{Enabled: false},
			wantNoErr: true,
		},
		{
			name:      "valid decoder declaration",
			codec:     &contracts.PayloadCodec{Enabled: true, Script: "function decodeUplink(input) { return { data: {} }; }"},
			wantNoErr: true,
		},
		{
			name:      "valid decoder assignment",
			codec:     &contracts.PayloadCodec{Enabled: true, Script: "const decodeUplink = (input) => ({ data: {} });"},
			wantNoErr: true,
		},
		{
			name:      "empty script",
			codec:     &contracts.PayloadCodec{Enabled: true, Script: "  "},
			wantField: "payloadConfig.payloadCodec.script",
		},
		{
			name:      "invalid syntax",
			codec:     &contracts.PayloadCodec{Enabled: true, Script: "function decodeUplink(input) { return ;"},
			wantField: "payloadConfig.payloadCodec.script",
		},
		{
			name:      "missing decoder",
			codec:     &contracts.PayloadCodec{Enabled: true, Script: "function encodeUplink(input) { return input; }"},
			wantField: "payloadConfig.payloadCodec.script",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device := validationDevice()
			device.PayloadConfig.PayloadCodec = test.codec
			err := ValidateHardware([]contracts.Device{device}, nil)
			if test.wantNoErr {
				if err != nil {
					t.Fatalf("valid codec was rejected: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantField) {
				t.Fatalf("expected validation issue for %s, got %v", test.wantField, err)
			}
		})
	}
}

func TestCodecDeclarationValidationRejectsMisleadingOrAsynchronousScripts(t *testing.T) {
	for _, script := range []string{
		`// function decodeUplink(input) {}\nconst unrelated = 1;`,
		`const description = "function decodeUplink(input) {}";`,
		`function wrapper() { function decodeUplink(input) { return { data: {} }; } }`,
		`const decodeUplink = 42;`,
		`async function decodeUplink(input) { return { data: {} }; }`,
		`const decodeUplink = async input => ({ data: {} });`,
		`function* decodeUplink(input) { yield input; }`,
	} {
		t.Run(script, func(t *testing.T) {
			device := validationDevice()
			device.PayloadConfig.PayloadCodec = &contracts.PayloadCodec{Enabled: true, Script: script}
			if err := ValidateHardware([]contracts.Device{device}, nil); err == nil {
				t.Fatal("unavailable or asynchronous decoder was accepted")
			}
		})
	}
}

func TestValidateHardwareCoversDeviceAndGatewayRules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*contracts.Device, *contracts.Gateway)
		field  string
	}{
		{"device identity", func(device *contracts.Device, _ *contracts.Gateway) {
			device.ID = "bad"
			device.Name = " "
			device.DevEUI = "zz"
		}, "devices[0].id"},
		{"device enums and location", func(device *contracts.Device, _ *contracts.Gateway) {
			device.Class = contracts.DeviceClass("invalid")
			device.Activation = contracts.DeviceActivation("invalid")
			device.LocationConfig.Latitude = float32Pointer(91)
			device.LocationConfig.Longitude = float32Pointer(-181)
			device.LocationConfig.Altitude = nil
			device.LocationConfig.Region = contracts.DeviceRegion("invalid")
		}, "devices[0].class"},
		{"device radio configuration", func(device *contracts.Device, _ *contracts.Gateway) {
			device.RX1Config.Delay = nil
			device.RX1Config.Duration = intPointer(0)
			device.RX1Config.DataRateOffset = intPointer(6)
			device.RX2Config.ChannelFrequency = 0
			device.RX2Config.DataRate = intPointer(6)
			device.RX2Config.ACKTimeout = 999
		}, "devices[0].RX1Config.delay"},
		{"device frame and payload", func(device *contracts.Device, _ *contracts.Gateway) {
			device.FrameConfig.FPort = 0
			device.FrameConfig.Retransmission = intPointer(-1)
			device.PayloadConfig.UplinkInterval = 0
			device.PayloadConfig.OversizedPayloadBehavior = contracts.OversizedPayloadBehavior("invalid")
			device.PayloadConfig.MType = contracts.DeviceMType("invalid")
			device.PayloadConfig.Base64Encoded = true
			device.PayloadConfig.Payload = "!"
		}, "devices[0].frameConfig.fPort"},
		{"device advanced settings", func(device *contracts.Device, _ *contracts.Gateway) {
			device.AdvancedConfig.AntennaRange = float32(math.Inf(1))
			rate := 6
			device.AdvancedConfig.UplinkDataRate = &rate
		}, "devices[0].advancedConfig.antennaRange"},
		{"missing OTAA config", func(device *contracts.Device, _ *contracts.Gateway) { device.OOTAConfig = nil }, "devices[0].OOTAConfig"},
		{"incompatible ABP config", func(device *contracts.Device, _ *contracts.Gateway) { device.ABPConfig = &contracts.ABPConfig{} }, "devices[0].ABPConfig"},
		{"gateway identity and location", func(_ *contracts.Device, gateway *contracts.Gateway) {
			gateway.ID = "bad"
			gateway.Name = ""
			gateway.GatewayEUI = "bad"
			gateway.Latitude = float32Pointer(91)
			gateway.Longitude = float32Pointer(-181)
			gateway.Altitude = nil
		}, "gateways[0].id"},
		{"virtual gateway fields", func(_ *contracts.Device, gateway *contracts.Gateway) {
			endpoint := "127.0.0.1"
			gateway.GatewayIPv4 = &endpoint
			gateway.GatewayPort = int32Pointer(1700)
			gateway.KeepAlive = int32Pointer(0)
		}, "gateways[0].keepAlive"},
		{"real gateway fields", func(_ *contracts.Device, gateway *contracts.Gateway) {
			gateway.Type = contracts.Real
			gateway.KeepAlive = int32Pointer(30)
			gateway.GatewayIPv4 = stringPointer("not-ip")
			gateway.GatewayPort = int32Pointer(65536)
		}, "gateways[0].keepAlive"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device, gateway := validationDevice(), validationGateway()
			test.mutate(&device, &gateway)
			err := ValidateHardware([]contracts.Device{device}, []contracts.Gateway{gateway})
			if err == nil || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("expected validation issue for %s, got %v", test.field, err)
			}
		})
	}
}

func TestValidateHardwareRejectsDuplicateIdentityAcrossEntities(t *testing.T) {
	device := validationDevice()
	duplicateDevice := validationDevice()
	duplicateDevice.ID = "8b75c9a2-9e1b-4da2-9226-000000000002"
	duplicateDevice.DevEUI = device.DevEUI
	gateway := validationGateway()
	duplicateGateway := validationGateway()
	duplicateGateway.ID = "8b75c9a2-9e1b-4da2-9226-000000000005"
	duplicateGateway.GatewayEUI = gateway.GatewayEUI

	err := ValidateHardware([]contracts.Device{device, duplicateDevice}, []contracts.Gateway{gateway, duplicateGateway})
	if err == nil || !strings.Contains(err.Error(), "must be unique") {
		t.Fatalf("expected duplicate identity errors, got %v", err)
	}
}

func TestEveryDeviceParameterHasAValidationContract(t *testing.T) {
	tests := []struct {
		name   string
		field  string
		mutate func(*contracts.Device)
	}{
		{"name", "name", func(device *contracts.Device) { device.Name = " " }},
		{"DevEUI", "devEUI", func(device *contracts.Device) { device.DevEUI = "GG00000000000000" }},
		{"class", "class", func(device *contracts.Device) { device.Class = "class-z" }},
		{"activation", "activation", func(device *contracts.Device) { device.Activation = "invalid" }},
		{"latitude", "locationConfig.latitude", func(device *contracts.Device) { device.LocationConfig.Latitude = float32Pointer(-91) }},
		{"longitude", "locationConfig.longitude", func(device *contracts.Device) { device.LocationConfig.Longitude = float32Pointer(181) }},
		{"altitude", "locationConfig.altitude", func(device *contracts.Device) { device.LocationConfig.Altitude = float32Pointer(float32(math.NaN())) }},
		{"region", "locationConfig.region", func(device *contracts.Device) { device.LocationConfig.Region = "invalid" }},
		{"RX1 delay", "RX1Config.delay", func(device *contracts.Device) { device.RX1Config.Delay = intPointer(0) }},
		{"RX1 duration", "RX1Config.duration", func(device *contracts.Device) { device.RX1Config.Duration = intPointer(-1) }},
		{"RX1 data-rate offset", "RX1Config.dataRateOffset", func(device *contracts.Device) { device.RX1Config.DataRateOffset = intPointer(6) }},
		{"RX2 delay", "RX2Config.delay", func(device *contracts.Device) { device.RX2Config.Delay = nil }},
		{"RX2 duration", "RX2Config.duration", func(device *contracts.Device) { device.RX2Config.Duration = intPointer(0) }},
		{"RX2 frequency", "RX2Config.channelFrequency", func(device *contracts.Device) { device.RX2Config.ChannelFrequency = 0 }},
		{"RX2 data rate", "RX2Config.dataRate", func(device *contracts.Device) { device.RX2Config.DataRate = intPointer(99) }},
		{"ACK timeout", "RX2Config.ACKTimeout", func(device *contracts.Device) { device.RX2Config.ACKTimeout = 3001 }},
		{"FPort", "frameConfig.fPort", func(device *contracts.Device) { device.FrameConfig.FPort = 224 }},
		{"retransmissions", "frameConfig.retransmission", func(device *contracts.Device) { device.FrameConfig.Retransmission = intPointer(-1) }},
		{"uplink counter", "frameConfig.FCntUp", func(device *contracts.Device) { device.FrameConfig.FCntUp = optionalValidationCounter(-1) }},
		{"downlink counter", "frameConfig.FCntDown", func(device *contracts.Device) { device.FrameConfig.FCntDown = optionalValidationCounter(-1) }},
		{"uplink interval", "payloadConfig.uplinkInterval", func(device *contracts.Device) { device.PayloadConfig.UplinkInterval = float32(math.Inf(1)) }},
		{"oversized behavior", "payloadConfig.oversizedPayloadBehavior", func(device *contracts.Device) { device.PayloadConfig.OversizedPayloadBehavior = "invalid" }},
		{"MType", "payloadConfig.MType", func(device *contracts.Device) { device.PayloadConfig.MType = "invalid" }},
		{"base64 payload", "payloadConfig.payload", func(device *contracts.Device) {
			device.PayloadConfig.Base64Encoded = true
			device.PayloadConfig.Payload = "%%%"
		}},
		{"payload codec", "payloadConfig.payloadCodec.script", func(device *contracts.Device) {
			device.PayloadConfig.PayloadCodec = &contracts.PayloadCodec{Enabled: true, Script: "function decodeUplink(input) {"}
		}},
		{"antenna range", "advancedConfig.antennaRange", func(device *contracts.Device) { device.AdvancedConfig.AntennaRange = 0 }},
		{"uplink data rate", "advancedConfig.uplinkDataRate", func(device *contracts.Device) { device.AdvancedConfig.UplinkDataRate = intPointer(99) }},
		{"OTAA JoinEUI", "OOTAConfig.joinEUI", func(device *contracts.Device) { device.OOTAConfig.JoinEUI = "bad" }},
		{"OTAA AppKey", "OOTAConfig.appKey", func(device *contracts.Device) { device.OOTAConfig.AppKey = "bad" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			device := validationDevice()
			test.mutate(&device)
			err := ValidateHardware([]contracts.Device{device}, nil)
			if err == nil || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("expected validation for %s, got %v", test.field, err)
			}
		})
	}
}

func TestABPAndGatewayEndpointParameterValidation(t *testing.T) {
	abpMutations := []struct {
		name, field string
		mutate      func(*contracts.ABPConfig)
	}{
		{"DevAddr", "ABPConfig.devAddr", func(config *contracts.ABPConfig) { config.DevAddr = "bad" }},
		{"NwkSKey", "ABPConfig.nwkSKey", func(config *contracts.ABPConfig) { config.NwkSKey = "bad" }},
		{"AppSKey", "ABPConfig.appSKey", func(config *contracts.ABPConfig) { config.AppSKey = "bad" }},
	}
	for _, test := range abpMutations {
		t.Run("ABP "+test.name, func(t *testing.T) {
			device := validationDevice()
			device.Activation = contracts.ABP
			device.OOTAConfig = nil
			device.ABPConfig = &contracts.ABPConfig{DevAddr: "26011BDA", NwkSKey: "00112233445566778899AABBCCDDEEFF", AppSKey: "FFEEDDCCBBAA99887766554433221100"}
			test.mutate(device.ABPConfig)
			err := ValidateHardware([]contracts.Device{device}, nil)
			if err == nil || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("expected validation for %s, got %v", test.field, err)
			}
		})
	}

	real := validationGateway()
	real.Type = contracts.Real
	real.KeepAlive = nil
	real.GatewayIPv4 = stringPointer("127.0.0.1")
	real.GatewayPort = int32Pointer(1700)
	if err := ValidateHardware(nil, []contracts.Gateway{real}); err != nil {
		t.Fatalf("valid real gateway endpoint was rejected: %v", err)
	}
}

func validationDevice() contracts.Device {
	return contracts.Device{
		ID: "8b75c9a2-9e1b-4da2-9226-000000000001", Active: true, Name: "Validation Device", DevEUI: "70B3D57ED0000001", Class: contracts.ClassA, Activation: contracts.OOTA,
		OOTAConfig:     &contracts.OOTAConfig{JoinEUI: "70B3D57ED0000002", AppKey: "70B3D57ED00000000000000000000001"},
		LocationConfig: contracts.LocationConfig{Latitude: float32Pointer(37.5), Longitude: float32Pointer(15.1), Altitude: float32Pointer(100), Region: contracts.EU868},
		RX1Config:      contracts.RX1Config{Delay: intPointer(1000), Duration: intPointer(1000), DataRateOffset: intPointer(0)},
		RX2Config:      contracts.RX2Config{Delay: intPointer(2000), Duration: intPointer(1000), ChannelFrequency: 869525000, DataRate: intPointer(5), ACKTimeout: 2000},
		FrameConfig:    contracts.FrameConfig{FPort: 1, Retransmission: intPointer(1)},
		PayloadConfig:  contracts.PayloadConfig{UplinkInterval: 10, OversizedPayloadBehavior: contracts.Truncate, MType: contracts.UnconfirmedDataUp, Payload: "test"},
		AdvancedConfig: contracts.AdvancedConfig{AntennaRange: 1000},
	}
}

func validationGateway() contracts.Gateway {
	return contracts.Gateway{
		ID: "8b75c9a2-9e1b-4da2-9226-000000000003", Active: true, Name: "Validation Gateway", Type: contracts.Virtual, GatewayEUI: "A840410001000101", KeepAlive: int32Pointer(30),
		Latitude: float32Pointer(37.5), Longitude: float32Pointer(15.1), Altitude: float32Pointer(100),
	}
}

func intPointer(value int) *int             { return &value }
func int32Pointer(value int32) *int32       { return &value }
func float32Pointer(value float32) *float32 { return &value }
func stringPointer(value string) *string    { return &value }

func optionalValidationCounter(value int) **int {
	inner := &value
	return &inner
}

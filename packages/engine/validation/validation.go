package validation

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"net"
	"net/url"
	"strings"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

func ValidateSimulationConfig(config contracts.SimulationConfig) error {
	issues := &types.ValidationErrors{}
	if math.IsNaN(config.Speed) || math.IsInf(config.Speed, 0) || config.Speed <= 0 {
		issues.Add("speed", "must_be_positive", "speed must be a finite value greater than zero")
	}
	if config.GatewayBridge != nil && config.GatewayBridge.Enabled {
		address := strings.TrimSpace(config.GatewayBridge.Address)
		if address == "" {
			issues.Add("gatewayBridge.address", "required", "gateway bridge address is required")
		}
		if config.GatewayBridge.Port < 1 || config.GatewayBridge.Port > 65535 {
			issues.Add("gatewayBridge.port", "invalid_port", "gateway bridge port must be between 1 and 65535")
		}
		if address != "" && config.GatewayBridge.Port >= 1 && config.GatewayBridge.Port <= 65535 {
			if parsed, err := url.Parse(address); err == nil && parsed.Hostname() != "" {
				address = parsed.Hostname()
			}
			listenAddress := net.JoinHostPort(address, fmt.Sprint(config.GatewayBridge.Port))
			if _, err := net.ResolveUDPAddr("udp", listenAddress); err != nil {
				issues.Add("gatewayBridge.address", "invalid_udp_address", "gateway bridge address must be a valid IPv4 address or host")
			}
		}
	}
	return validationResult(issues)
}

func ValidateHardware(devices []contracts.Device, gateways []contracts.Gateway) error {
	issues := &types.ValidationErrors{}
	ids := make(map[string]string, len(devices)+len(gateways))
	devEUIs := make(map[string]struct{}, len(devices))
	gatewayEUIs := make(map[string]struct{}, len(gateways))

	for index, device := range devices {
		prefix := fmt.Sprintf("devices[%d]", index)
		validateDevice(issues, prefix, device)
		registerUnique(issues, ids, prefix+".id", device.ID, "device")

		devEUI := strings.ToUpper(device.DevEUI)
		if _, exists := devEUIs[devEUI]; exists {
			issues.Add(prefix+".devEUI", "duplicate", "DevEUI must be unique")
		}
		devEUIs[devEUI] = struct{}{}
	}

	for index, gateway := range gateways {
		prefix := fmt.Sprintf("gateways[%d]", index)
		validateGateway(issues, prefix, gateway)
		registerUnique(issues, ids, prefix+".id", gateway.ID, "gateway")

		gatewayEUI := strings.ToUpper(gateway.GatewayEUI)
		if _, exists := gatewayEUIs[gatewayEUI]; exists {
			issues.Add(prefix+".gatewayEUI", "duplicate", "Gateway EUI must be unique")
		}
		gatewayEUIs[gatewayEUI] = struct{}{}
	}

	return validationResult(issues)
}

func validationResult(issues *types.ValidationErrors) error {
	if len(issues.Issues) == 0 {
		return nil
	}
	return issues
}

func registerUnique(
	issues *types.ValidationErrors,
	ids map[string]string,
	field string,
	id string,
	kind string,
) {
	if id == "" {
		return
	}
	if previous, exists := ids[id]; exists {
		issues.Add(field, "duplicate", fmt.Sprintf("%s id duplicates %s", kind, previous))
		return
	}
	ids[id] = field
}

func validateDevice(issues *types.ValidationErrors, prefix string, device contracts.Device) {
	validateUUID(issues, prefix+".id", device.ID)
	validateRequired(issues, prefix+".name", device.Name)
	validateHex(issues, prefix+".devEUI", device.DevEUI, 16)
	if device.RadioProtocol != nil && !device.RadioProtocol.Valid() {
		issues.Add(prefix+".radioProtocol", "invalid_enum", "radio protocol is not supported")
	}
	if device.RadioReceiveMode != nil && !device.RadioReceiveMode.Valid() {
		issues.Add(prefix+".radioReceiveMode", "invalid_enum", "radio receive mode is not supported")
	}

	if !device.Class.Valid() {
		issues.Add(prefix+".class", "invalid_enum", "class is not supported")
	}
	if !device.Activation.Valid() {
		issues.Add(prefix+".activation", "invalid_enum", "activation is not supported")
	}
	validateLocation(issues, prefix+".locationConfig", device.LocationConfig)
	validateRX1(issues, prefix+".RX1Config", device.RX1Config)
	validateRX2(issues, prefix+".RX2Config", device.RX2Config, device.LocationConfig.Region)
	validateFrame(issues, prefix+".frameConfig", device.FrameConfig)
	validatePayload(issues, prefix+".payloadConfig", device.PayloadConfig)

	if device.AdvancedConfig.AntennaRange <= 0 ||
		math.IsNaN(float64(device.AdvancedConfig.AntennaRange)) ||
		math.IsInf(float64(device.AdvancedConfig.AntennaRange), 0) {
		issues.Add(prefix+".advancedConfig.antennaRange", "must_be_positive", "antenna range must be greater than zero")
	}
	plan, hasPlan := regional.Plan(device.LocationConfig.Region)
	if configured := device.AdvancedConfig.UplinkDataRate; configured != nil && hasPlan {
		if !regional.SupportsUplinkDataRate(device.LocationConfig.Region, *configured) {
			issues.Add(prefix+".advancedConfig.uplinkDataRate", "out_of_range", fmt.Sprintf("uplink data rate must be supported by %s (DR%d-DR%d)", device.LocationConfig.Region, plan.MinimumDataRate, plan.MaximumDataRate))
		}
	}

	switch device.Activation {
	case contracts.OOTA:
		if device.OOTAConfig == nil {
			issues.Add(prefix+".OOTAConfig", "required", "OTAA configuration is required for OTAA devices")
		} else {
			validateHex(issues, prefix+".OOTAConfig.joinEUI", device.OOTAConfig.JoinEUI, 16)
			validateHex(issues, prefix+".OOTAConfig.appKey", device.OOTAConfig.AppKey, 32)
		}
		if device.ABPConfig != nil {
			issues.Add(prefix+".ABPConfig", "forbidden", "ABP configuration cannot be set on an OTAA device")
		}
	case contracts.ABP:
		if device.ABPConfig == nil {
			issues.Add(prefix+".ABPConfig", "required", "ABP configuration is required for ABP devices")
		} else {
			validateHex(issues, prefix+".ABPConfig.devAddr", device.ABPConfig.DevAddr, 8)
			validateHex(issues, prefix+".ABPConfig.nwkSKey", device.ABPConfig.NwkSKey, 32)
			validateHex(issues, prefix+".ABPConfig.appSKey", device.ABPConfig.AppSKey, 32)
		}
		if device.OOTAConfig != nil {
			issues.Add(prefix+".OOTAConfig", "forbidden", "OTAA configuration cannot be set on an ABP device")
		}
	}
}

func validateGateway(issues *types.ValidationErrors, prefix string, gateway contracts.Gateway) {
	validateUUID(issues, prefix+".id", gateway.ID)
	validateRequired(issues, prefix+".name", gateway.Name)
	if !gateway.Type.Valid() {
		issues.Add(prefix+".type", "invalid_enum", "gateway type is not supported")
	}
	validateHex(issues, prefix+".gatewayEUI", gateway.GatewayEUI, 16)
	validateCoordinate(issues, prefix+".latitude", gateway.Latitude, -90, 90)
	validateCoordinate(issues, prefix+".longitude", gateway.Longitude, -180, 180)
	if gateway.Altitude == nil || math.IsNaN(float64(*gateway.Altitude)) || math.IsInf(float64(*gateway.Altitude), 0) {
		issues.Add(prefix+".altitude", "required", "altitude must be a finite number")
	}

	switch gateway.Type {
	case contracts.Virtual:
		if gateway.KeepAlive == nil || *gateway.KeepAlive <= 0 {
			issues.Add(prefix+".keepAlive", "required", "virtual gateways require a positive keep-alive interval")
		}
		if gateway.GatewayIPv4 != nil || gateway.GatewayPort != nil {
			issues.Add(prefix, "incompatible_fields", "virtual gateways cannot define a real gateway endpoint")
		}
	case contracts.Real:
		if gateway.KeepAlive != nil {
			issues.Add(prefix+".keepAlive", "forbidden", "real gateways cannot define keep-alive")
		}
		if gateway.GatewayIPv4 == nil || net.ParseIP(*gateway.GatewayIPv4).To4() == nil {
			issues.Add(prefix+".gatewayIPv4", "invalid_ipv4", "real gateways require a valid IPv4 address")
		}
		if gateway.GatewayPort == nil || *gateway.GatewayPort < 1 || *gateway.GatewayPort > 65535 {
			issues.Add(prefix+".gatewayPort", "invalid_port", "real gateways require a port between 1 and 65535")
		}
	}
}

func validateLocation(issues *types.ValidationErrors, prefix string, location contracts.LocationConfig) {
	validateCoordinate(issues, prefix+".latitude", location.Latitude, -90, 90)
	validateCoordinate(issues, prefix+".longitude", location.Longitude, -180, 180)
	if location.Altitude == nil || math.IsNaN(float64(*location.Altitude)) || math.IsInf(float64(*location.Altitude), 0) {
		issues.Add(prefix+".altitude", "required", "altitude must be a finite number")
	}
	if !location.Region.Valid() {
		issues.Add(prefix+".region", "invalid_enum", "region is not supported")
	}
}

func validateRX1(issues *types.ValidationErrors, prefix string, config contracts.RX1Config) {
	validatePositivePointer(issues, prefix+".delay", config.Delay)
	validatePositivePointer(issues, prefix+".duration", config.Duration)
	validateRangePointer(issues, prefix+".dataRateOffset", config.DataRateOffset, 0, 5)
}

func validateRX2(issues *types.ValidationErrors, prefix string, config contracts.RX2Config, region contracts.DeviceRegion) {
	validatePositivePointer(issues, prefix+".delay", config.Delay)
	validatePositivePointer(issues, prefix+".duration", config.Duration)
	if config.DataRate != nil {
		plan, ok := regional.Plan(region)
		if ok {
			if _, supported := plan.DataRates[*config.DataRate]; !supported {
				issues.Add(prefix+".dataRate", "out_of_range", fmt.Sprintf("RX2 data rate DR%d is not supported by %s", *config.DataRate, region))
			}
		}
	}
	if config.ChannelFrequency <= 0 || math.IsNaN(float64(config.ChannelFrequency)) || math.IsInf(float64(config.ChannelFrequency), 0) {
		issues.Add(prefix+".channelFrequency", "must_be_positive", "channel frequency must be greater than zero")
	}
	if config.ACKTimeout < 1000 || config.ACKTimeout > 3000 {
		issues.Add(prefix+".ACKTimeout", "out_of_range", "ACK timeout must be between 1000 and 3000 milliseconds")
	}
}

func validateFrame(issues *types.ValidationErrors, prefix string, config contracts.FrameConfig) {
	if config.FPort < 1 || config.FPort > 223 {
		issues.Add(prefix+".fPort", "out_of_range", "FPort must be between 1 and 223")
	}
	validateNonNegativePointer(issues, prefix+".retransmission", config.Retransmission)
	validateOptionalDoublePointer(issues, prefix+".FCntUp", config.FCntUp)
	validateOptionalDoublePointer(issues, prefix+".FCntDown", config.FCntDown)
}

func validatePayload(issues *types.ValidationErrors, prefix string, config contracts.PayloadConfig) {
	if config.UplinkInterval <= 0 || math.IsNaN(float64(config.UplinkInterval)) || math.IsInf(float64(config.UplinkInterval), 0) {
		issues.Add(prefix+".uplinkInterval", "must_be_positive", "uplink interval must be greater than zero")
	} else if time.Duration(float64(config.UplinkInterval)*float64(time.Second)) <= 0 {
		issues.Add(prefix+".uplinkInterval", "too_small", "uplink interval must be representable as a positive duration")
	}
	if !config.OversizedPayloadBehavior.Valid() {
		issues.Add(prefix+".oversizedPayloadBehavior", "invalid_enum", "oversized payload behavior is not supported")
	}
	if !config.MType.Valid() {
		issues.Add(prefix+".MType", "invalid_enum", "message type is not supported")
	}
	if config.Base64Encoded {
		if _, err := base64.StdEncoding.DecodeString(config.Payload); err != nil {
			issues.Add(prefix+".payload", "invalid_base64", "payload must be valid standard base64 when base64Encoded is true")
		}
	}
	validatePayloadCodec(issues, prefix+".payloadCodec", config.PayloadCodec)
}

func validateUUID(issues *types.ValidationErrors, field string, value string) {
	if _, err := uuid.Parse(value); err != nil {
		issues.Add(field, "invalid_uuid", "value must be a valid UUID")
	}
}

func validateRequired(issues *types.ValidationErrors, field string, value string) {
	if strings.TrimSpace(value) == "" {
		issues.Add(field, "required", "value is required")
	}
}

func validateHex(issues *types.ValidationErrors, field string, value string, length int) {
	if len(value) != length {
		issues.Add(field, "invalid_length", fmt.Sprintf("value must contain exactly %d hexadecimal characters", length))
		return
	}
	if _, err := hex.DecodeString(value); err != nil {
		issues.Add(field, "invalid_hex", "value must contain only hexadecimal characters")
	}
}

func validateCoordinate(issues *types.ValidationErrors, field string, value *float32, min float64, max float64) {
	if value == nil || math.IsNaN(float64(*value)) || math.IsInf(float64(*value), 0) {
		issues.Add(field, "required", "coordinate must be a finite number")
		return
	}
	if float64(*value) < min || float64(*value) > max {
		issues.Add(field, "out_of_range", fmt.Sprintf("coordinate must be between %g and %g", min, max))
	}
}

func validateNonNegativePointer(issues *types.ValidationErrors, field string, value *int) {
	if value == nil {
		issues.Add(field, "required", "value is required")
		return
	}
	if *value < 0 {
		issues.Add(field, "must_be_non_negative", "value cannot be negative")
	}
}

func validatePositivePointer(issues *types.ValidationErrors, field string, value *int) {
	if value == nil {
		issues.Add(field, "required", "value is required")
		return
	}
	if *value <= 0 {
		issues.Add(field, "must_be_positive", "value must be greater than zero")
	}
}

func validateRangePointer(issues *types.ValidationErrors, field string, value *int, min int, max int) {
	if value == nil {
		issues.Add(field, "required", "value is required")
		return
	}
	if *value < min || *value > max {
		issues.Add(field, "out_of_range", fmt.Sprintf("value must be between %d and %d", min, max))
	}
}

func validateOptionalDoublePointer(issues *types.ValidationErrors, field string, value **int) {
	if value == nil || *value == nil {
		return
	}
	if **value < 0 {
		issues.Add(field, "must_be_non_negative", "value cannot be negative")
	} else if uint64(**value) > uint64(^uint32(0)) {
		issues.Add(field, "out_of_range", "frame counter cannot exceed the LoRaWAN 32-bit range")
	}
}

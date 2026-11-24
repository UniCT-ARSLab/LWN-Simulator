package runtime

import (
	"fmt"
	"math"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/regional"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
	"github.com/google/uuid"
)

func validateMACCommand(region contracts.DeviceRegion, command types.MACCommand) error {
	plan := regional.MustPlan(region)
	switch command.Type {
	case types.MACLinkCheckAns:
		return validateOptionalRange("margin", command.Margin, 0, 254)
	case types.MACLinkADRReq:
		if command.DataRate != nil && !regional.SupportsUplinkDataRate(region, *command.DataRate) {
			return fmt.Errorf("data rate DR%d is not supported by %s", *command.DataRate, region)
		}
		if err := validateOptionalRange("TxPower", command.TxPower, 0, 15); err != nil {
			return err
		}
		return validateOptionalRange("NbTrans", command.NbTrans, 1, 15)
	case types.MACDutyCycleReq:
		return validateRequiredRange("max duty-cycle exponent", command.MaxDutyCycleExponent, 0, 15)
	case types.MACRXParamSetupReq:
		if command.DataRate == nil {
			return fmt.Errorf("RX parameter setup requires a data rate")
		}
		if _, ok := plan.DataRates[*command.DataRate]; !ok {
			return fmt.Errorf("RX data rate DR%d is not supported by %s", *command.DataRate, region)
		}
		if err := validateRequiredFrequency(command.Frequency); err != nil {
			return err
		}
		return validateOptionalRange("RX1 data-rate offset", command.RX1DataRateOffset, 0, 7)
	case types.MACDevStatusReq, types.MACPingSlotInfoAns:
		return nil
	case types.MACNewChannelReq:
		if err := validateRequiredRange("channel index", command.ChannelIndex, 0, 255); err != nil {
			return err
		}
		if err := validateRequiredFrequency(command.Frequency); err != nil {
			return err
		}
		if err := validateRequiredRange("minimum data rate", command.MinimumDataRate, plan.MinimumDataRate, plan.MaximumDataRate); err != nil {
			return err
		}
		return validateRequiredRange("maximum data rate", command.MaximumDataRate, *command.MinimumDataRate, plan.MaximumDataRate)
	case types.MACRXTimingSetupReq:
		if command.Delay == nil || *command.Delay < time.Second || *command.Delay > 15*time.Second || *command.Delay%time.Second != 0 {
			return fmt.Errorf("RX timing delay must be a whole number of seconds between 1 and 15")
		}
		return nil
	case types.MACTXParamSetupReq:
		return validateOptionalRange("maximum EIRP", command.MaximumEIRP, 0, 15)
	case types.MACDLChannelReq:
		if err := validateRequiredRange("channel index", command.ChannelIndex, 0, 255); err != nil {
			return err
		}
		return validateRequiredFrequency(command.Frequency)
	case types.MACDeviceTimeAns:
		if command.DeviceTime == nil {
			return fmt.Errorf("device time is required")
		}
		return nil
	case types.MACPingSlotChannelReq:
		if err := validateRequiredFrequency(command.Frequency); err != nil {
			return err
		}
		if command.DataRate == nil {
			return fmt.Errorf("ping-slot data rate is required")
		}
		if _, ok := plan.DataRates[*command.DataRate]; !ok {
			return fmt.Errorf("ping-slot data rate DR%d is not supported by %s", *command.DataRate, region)
		}
		return nil
	case types.MACBeaconFreqReq:
		return validateRequiredFrequency(command.Frequency)
	default:
		return fmt.Errorf("unsupported MAC command %q", command.Type)
	}
}

func applyMACCommands(device contracts.Device, session *types.DeviceSession, commands []types.MACCommand) {
	for _, command := range commands {
		switch command.Type {
		case types.MACLinkADRReq:
			if command.DataRate != nil {
				session.CurrentDataRate = *command.DataRate
				session.CurrentSpreadingFactor = spreadingFactorForDataRate(device.LocationConfig.Region, *command.DataRate)
			}
			if command.TxPower != nil {
				session.CurrentTxPower = *command.TxPower
			}
			if command.NbTrans != nil {
				session.UnconfirmedRepetitions = *command.NbTrans
			}
		case types.MACDutyCycleReq:
			session.MaximumDutyCycle = 1 / math.Pow(2, float64(*command.MaxDutyCycleExponent))
		case types.MACRXParamSetupReq:
			session.RX2DataRate = *command.DataRate
			session.RX2Frequency = *command.Frequency
			if command.RX1DataRateOffset != nil {
				session.RX1DataRateOffset = *command.RX1DataRateOffset
			}
		case types.MACNewChannelReq:
			profile := regional.Profile(device.LocationConfig.Region, *command.MinimumDataRate)
			session.AdditionalChannels = append(session.AdditionalChannels, types.RadioChannel{
				Frequency: *command.Frequency, Bandwidth: profile.Bandwidth,
				SpreadingFactor: profile.SpreadingFactor, DataRate: *command.MinimumDataRate,
			})
		case types.MACRXTimingSetupReq:
			session.ReceiveDelay = *command.Delay
		case types.MACTXParamSetupReq:
			if command.UplinkDwellTime != nil {
				session.UplinkDwellTime = *command.UplinkDwellTime
			}
			if command.DownlinkDwellTime != nil {
				session.DownlinkDwellTime = *command.DownlinkDwellTime
			}
			if command.MaximumEIRP != nil {
				session.MaximumEIRP = *command.MaximumEIRP
			}
		case types.MACPingSlotChannelReq:
			session.PingSlotFrequency = *command.Frequency
			session.PingSlotDataRate = *command.DataRate
		case types.MACPingSlotInfoAns:
			if command.PingSlotPeriodicity != nil {
				session.PingSlotPeriodicity = *command.PingSlotPeriodicity
			}
		case types.MACBeaconFreqReq:
			session.BeaconFrequency = *command.Frequency
		}
	}
}

func (e *Engine) applyDownlinkEffectsLocked(device contracts.Device, session *types.DeviceSession, downlink types.Downlink, at time.Duration) []contracts.SimulationEvent {
	applyMACCommands(device, session, downlink.MACCommands)
	if !downlink.FPending || device.Class != contracts.ClassA {
		return nil
	}
	poll := types.ScheduledEvent{
		ID: uuid.NewString(), At: at + time.Millisecond,
		Type: contracts.DeviceUplinkTransmitted, Message: "FPending poll uplink scheduled",
		DeviceID: device.ID, Kind: types.ScheduledEventDeviceUplink, Attempt: 1,
		FPendingPoll: true,
	}
	if err := e.scheduler.Schedule(poll); err != nil {
		return []contracts.SimulationEvent{e.newEventLocked(contracts.PacketDropped, "FPending poll uplink could not be scheduled", device.ID, "", poll.ID)}
	}
	return []contracts.SimulationEvent{e.newEventLocked(contracts.DeviceUplinkScheduled, "FPending poll uplink scheduled", device.ID, "", poll.ID)}
}

func classBPingSlotPeriod(session *types.DeviceSession) time.Duration {
	if session == nil || session.PingSlotPeriodicity <= 0 {
		return types.ClassBPingSlotPeriod
	}
	periodicity := session.PingSlotPeriodicity
	if periodicity > 7 {
		periodicity = 7
	}
	return time.Duration(1<<periodicity) * time.Second
}

func validateOptionalRange(name string, value *int, minimum, maximum int) error {
	if value == nil {
		return nil
	}
	return validateRequiredRange(name, value, minimum, maximum)
}

func validateRequiredRange(name string, value *int, minimum, maximum int) error {
	if value == nil {
		return fmt.Errorf("%s is required", name)
	}
	if *value < minimum || *value > maximum {
		return fmt.Errorf("%s must be between %d and %d", name, minimum, maximum)
	}
	return nil
}

func validateRequiredFrequency(value *int64) error {
	if value == nil || *value <= 0 || *value%100 != 0 || *value/100 > 0xffffff {
		return fmt.Errorf("frequency must be a positive 100 Hz multiple representable in 24 bits")
	}
	return nil
}

package regional

import (
	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

const bandwidth125KHz int64 = 125_000

var plans = map[contracts.DeviceRegion]types.RegionalPlan{
	contracts.EU868: euLikePlan(contracts.EU868, 868_100_000, 200_000, 3, 869_525_000, 0, 869_525_000, 3),
	contracts.EU433: euLikePlan(contracts.EU433, 433_175_000, 200_000, 3, 434_665_000, 0, 434_665_000, 3),
	contracts.CN779: euLikePlan(contracts.CN779, 779_500_000, 200_000, 3, 786_000_000, 0, 785_000_000, 3),
	contracts.CN470: euLikePlan(contracts.CN470, 470_300_000, 200_000, 96, 505_300_000, 0, 508_300_000, 2),
	contracts.AS923: euLikePlan(contracts.AS923, 923_200_000, 200_000, 2, 923_200_000, 2, 923_400_000, 3),
	contracts.IN865: planWithGroups(
		contracts.IN865,
		[]types.RegionalChannelGroup{
			{InitialFrequency: 865_062_500, FrequencyStep: 340_000, ChannelCount: 2, MinimumDataRate: 0, MaximumDataRate: 5},
			{InitialFrequency: 865_985_000, ChannelCount: 1, MinimumDataRate: 0, MaximumDataRate: 5},
		},
		866_550_000, 2, 866_550_000, 4, euLikeDataRates(),
	),
	contracts.KR920: planWithGroups(
		contracts.KR920,
		[]types.RegionalChannelGroup{
			{InitialFrequency: 922_100_000, FrequencyStep: 200_000, ChannelCount: 3, MinimumDataRate: 0, MaximumDataRate: 5},
			{InitialFrequency: 920_900_000, FrequencyStep: 200_000, ChannelCount: 6, MinimumDataRate: 0, MaximumDataRate: 5},
			{InitialFrequency: 922_700_000, FrequencyStep: 200_000, ChannelCount: 4, MinimumDataRate: 0, MaximumDataRate: 5},
		},
		921_900_000, 0, 923_100_000, 3, euLikeDataRates(),
	),
	contracts.RU864: withPingSlotFrequency(euLikePlan(contracts.RU864, 868_900_000, 200_000, 2, 869_100_000, 0, 869_100_000, 3), 868_900_000),
	contracts.US915: withDefaultDataRate(planWithGroups(
		contracts.US915,
		[]types.RegionalChannelGroup{
			{InitialFrequency: 902_300_000, FrequencyStep: 200_000, ChannelCount: 64, MinimumDataRate: 0, MaximumDataRate: 3},
			{InitialFrequency: 903_000_000, FrequencyStep: 1_600_000, ChannelCount: 8, MinimumDataRate: 4, MaximumDataRate: 4},
		},
		923_300_000, 8, 923_300_000, 8, us915DataRates(),
	), 3),
	contracts.AU915: planWithGroups(
		contracts.AU915,
		[]types.RegionalChannelGroup{
			{InitialFrequency: 915_200_000, FrequencyStep: 200_000, ChannelCount: 64, MinimumDataRate: 0, MaximumDataRate: 5},
			{InitialFrequency: 915_900_000, FrequencyStep: 1_600_000, ChannelCount: 8, MinimumDataRate: 6, MaximumDataRate: 6},
		},
		923_300_000, 8, 923_300_000, 8, au915DataRates(),
	),
}

func Plan(region contracts.DeviceRegion) (types.RegionalPlan, bool) {
	plan, ok := plans[region]
	if !ok {
		return types.RegionalPlan{}, false
	}
	return clonePlan(plan), true
}

func MustPlan(region contracts.DeviceRegion) types.RegionalPlan {
	if plan, ok := Plan(region); ok {
		return plan
	}
	plan, _ := Plan(contracts.EU868)
	return plan
}

func ClampDataRate(region contracts.DeviceRegion, dataRate int) int {
	plan := MustPlan(region)
	if _, ok := plan.DataRates[dataRate]; ok {
		return dataRate
	}
	if dataRate < plan.MinimumDataRate {
		return plan.MinimumDataRate
	}
	if dataRate > plan.MaximumDataRate {
		return plan.MaximumDataRate
	}
	return plan.DefaultDataRate
}

func Profile(region contracts.DeviceRegion, dataRate int) types.DataRateProfile {
	plan := MustPlan(region)
	dataRate = ClampDataRate(region, dataRate)
	if profile, ok := plan.DataRates[dataRate]; ok {
		return profile
	}
	return plan.DataRates[plan.DefaultDataRate]
}

func DataRateForModulation(region contracts.DeviceRegion, spreadingFactor int, bandwidth int64) (int, bool) {
	plan := MustPlan(region)
	for dataRate, profile := range plan.DataRates {
		if profile.SpreadingFactor == spreadingFactor && profile.Bandwidth == bandwidth {
			return dataRate, true
		}
	}
	return 0, false
}

func UplinkChannels(region contracts.DeviceRegion, dataRate int) []types.RadioChannel {
	plan := MustPlan(region)
	if !SupportsUplinkDataRate(region, dataRate) {
		return nil
	}
	profile := Profile(region, dataRate)
	channels := make([]types.RadioChannel, 0)
	for _, group := range plan.UplinkChannels {
		if dataRate < group.MinimumDataRate || dataRate > group.MaximumDataRate {
			continue
		}
		for index := 0; index < group.ChannelCount; index++ {
			channels = append(channels, types.RadioChannel{
				Frequency:       group.InitialFrequency + int64(index)*group.FrequencyStep,
				Bandwidth:       profile.Bandwidth,
				SpreadingFactor: profile.SpreadingFactor,
				DataRate:        dataRate,
			})
		}
	}
	return channels
}

func SupportsUplinkDataRate(region contracts.DeviceRegion, dataRate int) bool {
	plan := MustPlan(region)
	if _, ok := plan.DataRates[dataRate]; !ok {
		return false
	}
	for _, group := range plan.UplinkChannels {
		if dataRate >= group.MinimumDataRate && dataRate <= group.MaximumDataRate {
			return true
		}
	}
	return false
}

func euLikePlan(region contracts.DeviceRegion, initial, step int64, count int, rx2Frequency int64, rx2DataRate int, beaconFrequency int64, beaconDataRate int) types.RegionalPlan {
	return planWithGroups(region, []types.RegionalChannelGroup{{
		InitialFrequency: initial,
		FrequencyStep:    step,
		ChannelCount:     count,
		MinimumDataRate:  0,
		MaximumDataRate:  5,
	}}, rx2Frequency, rx2DataRate, beaconFrequency, beaconDataRate, euLikeDataRates())
}

func planWithGroups(region contracts.DeviceRegion, groups []types.RegionalChannelGroup, rx2Frequency int64, rx2DataRate int, beaconFrequency int64, beaconDataRate int, dataRates map[int]types.DataRateProfile) types.RegionalPlan {
	return types.RegionalPlan{
		Region:            region,
		DefaultDataRate:   5,
		MinimumDataRate:   0,
		MaximumDataRate:   maximumUplinkDataRate(groups),
		RX2Frequency:      rx2Frequency,
		RX2DataRate:       rx2DataRate,
		BeaconFrequency:   beaconFrequency,
		BeaconDataRate:    beaconDataRate,
		PingSlotFrequency: beaconFrequency,
		PingSlotDataRate:  beaconDataRate,
		UplinkChannels:    groups,
		DataRates:         dataRates,
	}
}

func euLikeDataRates() map[int]types.DataRateProfile {
	return map[int]types.DataRateProfile{
		0: profile(0, 12, bandwidth125KHz, 51),
		1: profile(1, 11, bandwidth125KHz, 51),
		2: profile(2, 10, bandwidth125KHz, 51),
		3: profile(3, 9, bandwidth125KHz, 115),
		4: profile(4, 8, bandwidth125KHz, 222),
		5: profile(5, 7, bandwidth125KHz, 222),
	}
}

func us915DataRates() map[int]types.DataRateProfile {
	return map[int]types.DataRateProfile{
		0:  profile(0, 10, bandwidth125KHz, 11),
		1:  profile(1, 9, bandwidth125KHz, 53),
		2:  profile(2, 8, bandwidth125KHz, 125),
		3:  profile(3, 7, bandwidth125KHz, 242),
		4:  profile(4, 8, 500_000, 242),
		8:  profile(8, 12, 500_000, 33),
		9:  profile(9, 11, 500_000, 109),
		10: profile(10, 10, 500_000, 222),
		11: profile(11, 9, 500_000, 222),
		12: profile(12, 8, 500_000, 222),
		13: profile(13, 7, 500_000, 222),
	}
}

func au915DataRates() map[int]types.DataRateProfile {
	profiles := euLikeDataRates()
	profiles[6] = profile(6, 8, 500_000, 222)
	for dataRate, value := range us915DataRates() {
		if dataRate >= 8 {
			profiles[dataRate] = value
		}
	}
	return profiles
}

func profile(dataRate, spreadingFactor int, bandwidth int64, maximumPayload int) types.DataRateProfile {
	return types.DataRateProfile{DataRate: dataRate, SpreadingFactor: spreadingFactor, Bandwidth: bandwidth, MaximumPayload: maximumPayload}
}

func maximumUplinkDataRate(groups []types.RegionalChannelGroup) int {
	maximum := 0
	for _, group := range groups {
		if group.MaximumDataRate > maximum {
			maximum = group.MaximumDataRate
		}
	}
	return maximum
}

func withDefaultDataRate(plan types.RegionalPlan, dataRate int) types.RegionalPlan {
	plan.DefaultDataRate = dataRate
	return plan
}

func withPingSlotFrequency(plan types.RegionalPlan, frequency int64) types.RegionalPlan {
	plan.PingSlotFrequency = frequency
	return plan
}

func clonePlan(plan types.RegionalPlan) types.RegionalPlan {
	plan.UplinkChannels = append([]types.RegionalChannelGroup(nil), plan.UplinkChannels...)
	plan.DataRates = make(map[int]types.DataRateProfile, len(plan.DataRates))
	for dataRate, profile := range plans[plan.Region].DataRates {
		plan.DataRates[dataRate] = profile
	}
	return plan
}

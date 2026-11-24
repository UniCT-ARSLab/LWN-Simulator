package runtime

import (
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

func TestUS915AcceptsRegionalRX2DownlinkDataRate(t *testing.T) {
	device := validDevice("a1c6e32b-4f0d-4b50-9fc5-000000000051")
	configureABP(&device)
	device.Class = contracts.ClassC
	device.LocationConfig.Region = contracts.US915
	device.RX2Config.ChannelFrequency = 923_300_000
	device.RX2Config.DataRate = intPtr(8)
	uplinkDataRate := 3
	device.AdvancedConfig.UplinkDataRate = &uplinkDataRate
	gateway := validGateway("6f0f1f30-7dc5-4bb3-a6f5-11b2ed9a7c51")
	engine := newTestEngine(t, types.NewManualClock(0), device, gateway)
	defer stopEngine(t, engine)

	if _, err := engine.QueueDownlink(types.Downlink{DeviceID: device.ID, Payload: []byte("US915 RX2"), DataRate: 8}); err != nil {
		t.Fatalf("queue US915 DR8 downlink: %v", err)
	}
}

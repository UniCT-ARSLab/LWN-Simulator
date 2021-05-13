package device

import (
	"encoding/base64"

	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
)

func (d *Device) DataRateToString() string {
	_, drString := d.Info.Configuration.Region.GetDataRate(d.Info.Status.DataRate)
	return drString
}

func (d *Device) GetModulation() string {
	modu, _ := d.Info.Configuration.Region.GetDataRate(d.Info.Status.DataRate)
	return modu
}

func (d *Device) SetInfo(payload []byte, joinRequest bool) pkt.RXPK {

	indexChannelUp := int(d.Info.Status.IndexchannelActive)
	datarate := d.DataRateToString()

	if joinRequest {
		datarate, indexChannelUp = d.Info.Configuration.Region.SetupInfoRequest(int(d.Info.Status.IndexchannelActive))
	}

	info := pkt.RXPK{
		CodR:      d.Info.Configuration.Region.GetCodR(d.Info.Status.DataRate),
		Channel:   uint16(indexChannelUp),
		Frequency: float64(d.Info.Configuration.Channels[d.Info.Status.IndexchannelActive].FrequencyUplink) / float64(1000000.0),
		DatR:      datarate,
		Size:      uint16(len(payload)),
		Data:      base64.StdEncoding.EncodeToString(payload),
		Modu:      d.GetModulation(),
	}

	return info
}

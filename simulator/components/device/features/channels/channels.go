package channels

import "errors"

//Channel that every device must handle
type Channel struct {
	Active            bool   `json:"Active"`
	EnableUplink      bool   `json:"EnableUplink"` //true enable | false avoid
	FrequencyUplink   uint32 `json:"FreqUplink"`   // forse c'è una sola frequenza
	FrequencyDownlink uint32 `json:"FreqDownlink"`
	MinDR             uint8  `json:"MinDR"`
	MaxDR             uint8  `json:"MaxDR"`
}

//UpdateChannel sets new field of channel
func (c *Channel) UpdateChannel(freq uint32, minDR uint8, maxDR uint8) {

	if freq == 0 {
		c.Active = false
		c.EnableUplink = false
	} else {
		c.Active = true
		c.EnableUplink = true
	}

	c.FrequencyUplink = freq
	c.FrequencyDownlink = freq

	c.MinDR = minDR
	c.MaxDR = maxDR
}

//NewChannel è il setup per un canale nuovo
func NewChannel(freq uint32, minDR uint8, maxDR uint8) Channel {
	c := Channel{
		Active:            true,
		EnableUplink:      true,
		FrequencyUplink:   freq,
		FrequencyDownlink: freq,
		MinDR:             minDR,
		MaxDR:             maxDR,
	}

	return c

}

//IsSupportedDR return feedback if datarate is supported from channel
func (c *Channel) IsSupportedDR(datarate uint8) error {
	if datarate >= c.MinDR && datarate <= c.MaxDR {
		return nil
	}
	return errors.New("Not supported DataRate")
}

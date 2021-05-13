package channels

import (
	"errors"
)

//Channel that every device must handle
type Channel struct {
	Active            bool   `json:"active"`
	EnableUplink      bool   `json:"enableUplink"` //true enable | false avoid
	FrequencyUplink   uint32 `json:"freqUplink"`
	FrequencyDownlink uint32 `json:"freqDownlink"`
	MinDR             uint8  `json:"minDR"`
	MaxDR             uint8  `json:"maxDR"`
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

func (c *Channel) IsSupportedDR(datarate uint8) error {

	if datarate >= c.MinDR && datarate <= c.MaxDR {
		return nil
	}

	return errors.New("Not supported DataRate")
}

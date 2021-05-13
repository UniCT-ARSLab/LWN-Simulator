package features

import (
	"encoding/json"
	"time"

	c "github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
	"github.com/brocaar/lorawan"
)

const (
	//ReceiveDelay delay to open receive window rx1
	ReceiveDelay = 1
)

//Window is a receive window
type Window struct {
	Channel      c.Channel     `json:"channel"` // RX1's frequency is same of last uplink's frequency
	Delay        time.Duration `json:"delay"`
	DurationOpen time.Duration `json:"durationOpen"`
	DataRate     uint8         `json:"dataRate"`
}

//GetListeningFrequency get window's listening frequency
func (w *Window) GetListeningFrequency() uint32 {
	return w.Channel.FrequencyDownlink
}

//SetListeningFrequency set window's listening frequency
func (w *Window) SetListeningFrequency(freq uint32) {
	w.Channel.FrequencyDownlink = freq
}

func (w *Window) OpenWindow(Delay time.Duration, ReceivedDownlink *dl.ReceivedDownlink) *lorawan.PHYPayload {

	if Delay == 0 {
		Delay = w.Delay
	}

	timerWindow := time.NewTimer(Delay)
	<-timerWindow.C //delay
	timerWindow.Stop()

	for {

		go func(durate time.Duration, buf *dl.ReceivedDownlink) {

			timer := time.NewTimer(durate)
			<-timer.C
			timer.Stop()

			buf.Signal()

		}(w.DurationOpen, ReceivedDownlink)

		return ReceivedDownlink.Pull()

	}
}

//MarshalJSON of device's Receive window
func (w *Window) MarshalJSON() ([]byte, error) {
	type Alias Window

	return json.Marshal(&struct {
		Delay        int `json:"delay"`
		DurationOpen int `json:"durationOpen"`
		*Alias
	}{
		Delay:        int(w.Delay / time.Millisecond),
		DurationOpen: int(w.DurationOpen / time.Millisecond),
		Alias:        (*Alias)(w),
	})

}

//UnmarshalJSON of device's Receive window
func (w *Window) UnmarshalJSON(data []byte) error {

	type Alias Window

	aux := &struct {
		Delay        int `json:"delay"`
		DurationOpen int `json:"durationOpen"`
		*Alias
	}{
		Alias: (*Alias)(w),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	w.Delay = time.Duration(aux.Delay) * time.Millisecond
	w.DurationOpen = time.Duration(aux.DurationOpen) * time.Millisecond

	return nil
}

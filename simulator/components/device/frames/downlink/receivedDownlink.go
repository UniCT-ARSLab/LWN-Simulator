package downlink

import (
	"sync"

	"github.com/brocaar/lorawan"
)

//ReceivedDownlink is for every Device
type ReceivedDownlink struct {
	Mux      sync.Mutex
	Downlink lorawan.PHYPayload
	Notify   chan struct{}
}

//Push data in struct
func (b *ReceivedDownlink) Push(data *lorawan.PHYPayload) {

	if data == nil {
		return
	}

	b.Mux.Lock()

	b.Downlink = *data

	b.Mux.Unlock()

	b.Notify <- struct{}{}
}

// Pull the current value of the counter for the given key.
func (b *ReceivedDownlink) Pull() *lorawan.PHYPayload {

	var phy lorawan.PHYPayload

	b.Mux.Lock()
	defer b.Mux.Unlock()

	phy = b.Downlink
	b.Downlink = lorawan.PHYPayload{}

	return &phy

}

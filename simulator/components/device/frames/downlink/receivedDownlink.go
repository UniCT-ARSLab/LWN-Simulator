package downlink

import (
	"sync"

	"github.com/brocaar/lorawan"
)

type ReceivedDownlink struct {
	Mutex    sync.Mutex
	Downlink *lorawan.PHYPayload
	Notify   *sync.Cond
	IsOpen   bool
}

func (b *ReceivedDownlink) Push(data *lorawan.PHYPayload) {

	if data == nil {
		return
	}

	b.Mutex.Lock()
	if b.IsOpen {

		b.Downlink = data
		b.Notify.Broadcast()

	}

	b.Mutex.Unlock()

}

func (b *ReceivedDownlink) Pull() *lorawan.PHYPayload {

	b.Mutex.Lock()

	defer b.Mutex.Unlock()

	if b.Downlink == nil {
		b.Notify.Wait()
	}

	phy := b.Downlink

	b.Downlink = nil //reset

	return phy

}

func (b *ReceivedDownlink) Wait() {
	b.Mutex.Lock()
	b.Notify.Wait()
	b.Mutex.Unlock()
}

func (b *ReceivedDownlink) Signal() {
	b.Mutex.Lock()
	b.Notify.Broadcast()
	b.Mutex.Unlock()
}

func (b *ReceivedDownlink) Open() {
	b.Mutex.Lock()
	b.IsOpen = true
	b.Mutex.Unlock()
}

func (b *ReceivedDownlink) Close() {
	b.Mutex.Lock()
	b.IsOpen = false
	b.Mutex.Unlock()
}

package buffer

import (
	"sync"

	"github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
)

type BufferUplink struct {
	Mutex   sync.Mutex
	Uplinks []packets.RXPK
	Notify  *sync.Cond
}

func (p *BufferUplink) Push(pkt packets.RXPK) {

	p.Mutex.Lock()

	p.Uplinks = append(p.Uplinks, pkt) // push

	if len(p.Uplinks) == 1 {
		p.Notify.Broadcast()
	}

	p.Mutex.Unlock()

}

func (p *BufferUplink) Pop() packets.RXPK {

	var upl packets.RXPK

	p.Mutex.Lock()
	defer p.Mutex.Unlock()

	if len(p.Uplinks) == 0 {
		p.Notify.Wait()
	}

	switch len(p.Uplinks) {
	case 0:
		return packets.RXPK{}
	case 1:
		upl, p.Uplinks = p.Uplinks[0], p.Uplinks[:0] // pop
	default:
		upl, p.Uplinks = p.Uplinks[0], p.Uplinks[1:]

	}

	return upl

}

func (p *BufferUplink) Signal() {

	p.Mutex.Lock()
	p.Notify.Broadcast()
	p.Mutex.Unlock()

}

package buffer

import (
	"sync"

	"github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
)

type BufferUplink struct {
	Mux         sync.Mutex
	Uplinks     []packets.RXPK
	NewUplinkCh chan struct{}
}

func (p *BufferUplink) Push(pkt packets.RXPK) {

	p.Mux.Lock()

	p.Uplinks = append(p.Uplinks, pkt) // push

	p.Mux.Unlock()

	p.NewUplinkCh <- struct{}{} //signal to gw sender

}

func (p *BufferUplink) Pop() packets.RXPK {

	var upl packets.RXPK

	p.Mux.Lock()
	defer p.Mux.Unlock()

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

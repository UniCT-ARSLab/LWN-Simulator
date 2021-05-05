package gateway

import (
	"sync"

	f "github.com/arslab/lwnsimulator/simulator/components/forwarder"
	res "github.com/arslab/lwnsimulator/simulator/resources"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/buffer"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/udp"
	"github.com/arslab/lwnsimulator/simulator/util"
)

func (g *Gateway) Setup(BridgeAddress *string,
	Resources *res.Resources, Forwarder *f.Forwarder) {

	g.State = util.Stopped

	g.Info.BridgeAddress = BridgeAddress

	g.Resources = Resources
	g.Forwarder = Forwarder

	g.BufferUplink = buffer.BufferUplink{}
	g.BufferUplink.Notify = sync.NewCond(&g.BufferUplink.Mutex)

	g.Print("Setup OK!", nil, util.PrintOnlyConsole)

}

func (g *Gateway) TurnON() {

	var err error

	g.State = util.Running

	//udp
	if g.Info.TypeGateway { //real
		g.Info.Connection, err = udp.ConnectTo(g.Info.AddrIP + ":" + g.Info.Port)
	} else { //virtual
		g.Info.Connection, err = udp.ConnectTo(*g.Info.BridgeAddress)
	}

	if err != nil {
		g.Print("", err, util.PrintOnlyConsole)
	} else {
		g.Print("UDP connection with "+g.Info.Connection.RemoteAddr().String(), nil, util.PrintOnlyConsole)
	}

	go g.Receiver()

	if g.Info.TypeGateway { //real
		go g.SenderReal()
	} else { //virtual
		go g.SenderVirtual()
	}

	g.Print("Turn ON", nil, util.PrintBoth)
}

func (g *Gateway) TurnOFF() {

	g.State = util.Stopped

	g.BufferUplink.Signal()   //signal to sender
	g.Info.Connection.Close() //signal to receiver

}

func (g *Gateway) IsOn() bool {

	if g.State == util.Running {
		return true
	}

	return false

}

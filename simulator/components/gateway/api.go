package packetforwarder

import (
	f "github.com/arslab/lwnsimulator/simulator/components/forwarder"
	res "github.com/arslab/lwnsimulator/simulator/resources"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/buffer"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/udp"
	"github.com/arslab/lwnsimulator/simulator/util"
)

func (g *Gateway) Setup(BridgeAddress *string,
	Resources *res.Resources,
	StateSimulator *uint8, Forwarder *f.Forwarder) {

	var err error

	g.Info.BridgeAddress = BridgeAddress
	g.StateSimulator = StateSimulator
	g.Resources = Resources
	g.Forwarder = Forwarder

	//udp
	if g.Info.TypeGateway { //real
		g.Info.Connection, err = udp.ConnectTo(g.Info.AddrIP + ":" + g.Info.Port)
	} else { //virtual
		g.Info.Connection, err = udp.ConnectTo(*g.Info.BridgeAddress)
	}

	if err != nil {
		g.Print("", err, util.PrintBoth)
	} else {
		g.Print("UDP connection with "+g.Info.Connection.RemoteAddr().String(), nil, util.PrintBoth)
	}

	g.BufferUplink = buffer.BufferUplink{
		NewUplinkCh: make(chan struct{}),
	}

	g.Print("Setup OK!", nil, util.PrintBoth)

}

func (g *Gateway) OnStart() {

	go g.Receiver()

	if g.Info.TypeGateway {
		go g.SenderReal()
	} else {
		go g.SenderVirtual()
	}

}

func (g *Gateway) OnStop() {

	g.BufferUplink.NewUplinkCh <- struct{}{} //signal to sender
	g.Info.Connection.Close()                //signal to receiver

}

func (g *Gateway) TurnON() {

	g.Info.Active = true

	g.OnStart()

	g.Print("Turn ON", nil, util.PrintBoth)
}

func (g *Gateway) TurnOFF() {
	g.Info.Active = false
	g.OnStop()
}

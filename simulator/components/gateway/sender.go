package gateway

import (
	"errors"
	"fmt"
	"time"

	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/udp"

	"github.com/arslab/lwnsimulator/simulator/util"
)

func (g *Gateway) SenderVirtual() {

	defer g.Print("Sender Turn OFF", nil, util.PrintOnlyConsole)

	go g.KeepAlive()

	for {

		rxpk := g.BufferUplink.Pop() //wait uplink

		if !g.CanExecute() {
			return
		}

		g.Stat.RXNb++
		g.Stat.RXOK++

		packet, err := g.createPacket(rxpk)
		if err != nil {
			g.Print("", err, util.PrintBoth)
		}

		_, err = udp.SendDataUDP(g.Info.Connection, packet)
		if err != nil {

			msg := fmt.Sprintf("Unable to send data to %v, it may be off", *g.Info.BridgeAddress)
			g.Print("", errors.New(msg), util.PrintBoth)

		} else {
			g.Print("PUSH DATA send", nil, util.PrintBoth)
		}

	}

}

func (g *Gateway) SenderReal() {

	defer g.Print("Sender Turn OFF", nil, util.PrintOnlyConsole)

	for {

		rxpk := g.BufferUplink.Pop() //wait uplink

		if !g.CanExecute() {
			return
		}

		g.Stat.RXNb++
		g.Stat.RXOK++

		packet, err := g.createPacket(rxpk)
		if err != nil {
			g.Print("", err, util.PrintBoth)
		}

		_, err = udp.SendDataUDP(g.Info.Connection, packet)
		if err != nil {

			msg := fmt.Sprintf("Unable to send data to %v, it may be off", *g.Info.BridgeAddress)
			g.Print("", errors.New(msg), util.PrintBoth)

		} else {
			msg := fmt.Sprintf("Forward PUSH DATA to %v:%v", g.Info.AddrIP, g.Info.Port)
			g.Print(msg, nil, util.PrintBoth)
		}

	}
}

func (g *Gateway) sendPullData() error {

	if !g.CanExecute() {
		return nil
	}

	pulldata, _ := pkt.CreatePacket(pkt.TypePullData, g.Info.MACAddress, pkt.Stat{}, nil, 0)

	_, err := udp.SendDataUDP(g.Info.Connection, pulldata)

	return err
}

func (g *Gateway) createPacket(info pkt.RXPK) ([]byte, error) {

	stat := pkt.Stat{
		Time: pkt.GetTime(),
		Lati: g.Info.Location.Latitude,
		Long: g.Info.Location.Longitude,
		Alti: g.Info.Location.Altitude,
		RXNb: g.Stat.RXNb,
		RXOK: g.Stat.RXOK,
		RXFW: g.Stat.RXFW,
		ACKR: g.Stat.ACKR,
		DWNb: g.Stat.DWNb,
		TXNb: g.Stat.TXNb,
	}

	rxpks := []pkt.RXPK{
		info,
	}

	return pkt.CreatePacket(pkt.TypePushData, g.Info.MACAddress, stat, rxpks, 0)
}

func (g *Gateway) KeepAlive() {

	tickerKeepAlive := time.NewTicker(g.Info.KeepAlive)

	for {
		if !g.CanExecute() {

			return

		} else {

			err := g.sendPullData()
			if err != nil {
				g.Print("", err, util.PrintBoth)
			} else {
				g.Print("PULL DATA send", nil, util.PrintBoth)
			}

		}

		<-tickerKeepAlive.C
	}

}

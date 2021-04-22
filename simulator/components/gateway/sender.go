package packetforwarder

import (
	"time"

	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/udp"

	"github.com/arslab/lwnsimulator/simulator/util"
)

func (g *Gateway) SenderVirtual() {

	err := g.sendPullData()
	if err != nil {
		g.Print("", err, util.PrintBoth)
	}

	tickerKeepAlive := time.NewTicker(g.Info.KeepAlive)

	for {

		select {

		case <-tickerKeepAlive.C:

			if *g.StateSimulator == util.Stopped {

				g.Print("Sender STOP", nil, util.PrintOnlyConsole)
				return

			} else {

				err := g.sendPullData()
				if err != nil {
					g.Print("", err, util.PrintBoth)
				}

			}

			break

		case <-g.BufferUplink.NewUplinkCh: //wait uplink

			rxpk := g.BufferUplink.Pop()

			ok := g.CanExecute()
			if !ok {

				g.Print("Sender STOP", nil, util.PrintOnlyConsole)
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
				g.Print("", err, util.PrintBoth)
			} else {
				g.Print("PUSH DATA send", nil, util.PrintBoth)
			}

		}

	}
}

func (g *Gateway) SenderReal() {

	for {

		<-g.BufferUplink.NewUplinkCh //wait uplink

		rxpk := g.BufferUplink.Pop()

		ok := g.CanExecute()
		if !ok {

			g.Print("Sender STOP", nil, util.PrintOnlyConsole)
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
			g.Print("", err, util.PrintBoth)
		} else {
			g.Print("PUSH DATA send", nil, util.PrintOnlySocket)
		}

	}
}

func (g *Gateway) sendPullData() error {

	ok := g.CanExecute()
	if !ok {

		g.Print("Sender STOP", nil, util.PrintOnlyConsole)
		return nil

	}

	pulldata, _ := pkt.CreatePacket(pkt.TypePullData, g.Info.MACAddress, pkt.Stat{}, nil, 0)

	_, err := udp.SendDataUDP(g.Info.Connection, pulldata)
	if err == nil {
		g.Print("PULL DATA send", nil, util.PrintBoth)
	}

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

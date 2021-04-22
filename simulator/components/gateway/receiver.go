package packetforwarder

import (
	"fmt"
	"time"

	pkt "github.com/arslab/lwnsimulator/simulator/resources/communication/packets"
	"github.com/arslab/lwnsimulator/simulator/resources/communication/udp"
	"github.com/arslab/lwnsimulator/simulator/util"
)

func (g *Gateway) Receiver() {

	ReceiveBuffer := make([]byte, 1024)

	defer g.Resources.ExitGroup.Done()

	for {
		var n int
		var err error

		ok := g.CanExecute()
		if !ok {

			g.Print("STOP", nil, util.PrintBoth)
			return

		}

		for g.Info.Connection == nil {

			ok := g.CanExecute()
			if !ok {

				g.Print("STOP", nil, util.PrintBoth)
				return

			}

			g.Info.Connection, err = udp.ConnectTo(*g.Info.BridgeAddress) //stabilish new connection
			if err != nil {

				g.Print("", err, util.PrintBoth)
				continue

			}

		}

		n, _, err = g.Info.Connection.ReadFromUDP(ReceiveBuffer)

		ok = g.CanExecute()
		if !ok {

			g.Print("STOP", nil, util.PrintBoth)
			return

		}

		if err != nil {

			g.Print("", err, util.PrintBoth)
			continue

		}

		receivedPack := ReceiveBuffer[:n]

		g.Stat.DWNb++

		err = pkt.ParseReceivePacket(receivedPack)
		if err != nil {

			msg := fmt.Sprintf("Error Parse Packet, %v", err)
			g.Print(msg, nil, util.PrintOnlySocket)

			continue
		}

		time.Sleep(time.Second) //sync le print

		msg := fmt.Sprintf("%v received", pkt.PacketToString(receivedPack[3]))
		g.Print(msg, nil, util.PrintBoth)

		typepkt := pkt.GetTypePacket(receivedPack)
		switch *typepkt {

		case pkt.TypePushAck:
			g.Stat.ACKR++

		case pkt.TypePullAck:
			break

		case pkt.TypePullResp:

			phy, freq, err := pkt.ExtractInfo(receivedPack)
			if err != nil {
				g.Print("", err, util.PrintBoth)
			}

			g.Forwarder.Downlink(phy, *freq)

			g.Stat.RXFW++

			//TX ACK
			packet, err := pkt.CreatePacket(pkt.TypeTxAck, g.Info.MACAddress, pkt.Stat{}, nil, pkt.GetTokenFromPullResp(receivedPack))
			if err != nil {
				g.Print("", err, util.PrintBoth)
			}

			_, err = udp.SendDataUDP(g.Info.Connection, packet)
			if err != nil {
				g.Print("", err, util.PrintBoth)

			} else {
				g.Stat.TXNb++
				g.Print("TX ACK sent", nil, util.PrintBoth)
			}

		default:
			g.Print("Packet not supported", nil, util.PrintOnlySocket)

		}

	}

}

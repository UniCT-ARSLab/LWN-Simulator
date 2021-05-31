package gateway

import (
	"errors"
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

		if !g.CanExecute() {

			g.Print("Turn OFF", nil, util.PrintBoth)
			return

		}

		for g.Info.Connection == nil {

			if !g.CanExecute() {

				g.Print("Turn OFF", nil, util.PrintBoth)
				return

			}

			g.Info.Connection, err = udp.ConnectTo(*g.Info.BridgeAddress) //stabilish new connection
			if err != nil {

				msg := fmt.Sprintf("Unable Connect to %v", g.Info.BridgeAddress)
				g.Print("", errors.New(msg), util.PrintBoth)

				continue

			}

		}

		n, _, err = g.Info.Connection.ReadFromUDP(ReceiveBuffer)

		if !g.CanExecute() {
			g.Print("Turn OFF", nil, util.PrintBoth)
			return
		}

		if err != nil {

			msg := fmt.Sprintf("No connection with %v, it may be off", *g.Info.BridgeAddress)
			g.Print("", errors.New(msg), util.PrintBoth)

			continue

		}

		receivedPack := ReceiveBuffer[:n]

		g.Stat.DWNb++

		err = pkt.ParseReceivePacket(receivedPack)
		if err != nil {
			g.Print("Packet not supported", nil, util.PrintBoth)
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

			phy, freq, err := pkt.GetInfoPullResp(receivedPack)
			if err != nil {
				g.Print("", err, util.PrintBoth)
				continue
			}

			g.Forwarder.Downlink(phy, *freq, g.Info.MACAddress)

			g.Stat.RXFW++

			//TX ACK
			packet, err := pkt.CreatePacket(pkt.TypeTxAck, g.Info.MACAddress, pkt.Stat{}, nil, pkt.GetTokenFromPullResp(receivedPack))
			if err != nil {
				g.Print("", err, util.PrintBoth)
			}

			_, err = udp.SendDataUDP(g.Info.Connection, packet)

			if !g.CanExecute() {
				g.Print("Turn OFF", nil, util.PrintBoth)
				return
			}

			if err != nil {
				msg := fmt.Sprintf("No connection with %v, it may be off", *g.Info.BridgeAddress)
				g.Print("", errors.New(msg), util.PrintBoth)
			} else {

				g.Stat.TXNb++
				g.Print("TX ACK sent", nil, util.PrintBoth)

			}

		default:
			g.Print("Packet not supported", nil, util.PrintBoth)

		}

	}

}

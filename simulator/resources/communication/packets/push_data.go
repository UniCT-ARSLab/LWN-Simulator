package packets

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brocaar/lorawan"
)

type PDPacket struct {
	Header  []byte
	Payload PushDataPayload
}

type PushDataPayload struct {
	RXPK []RXPK `json:"rxpk,omitempty"`
	Stat *Stat  `json:"stat,omitempty"`
}

type Stat struct {
	Time string  `json:"time"` // UTC 'system' time of the gateway, ISO 8601 'expanded' format (e.g 2014-01-12 08:59:28 GMT)
	Lati float64 `json:"lati"` // GPS latitude of the gateway in degree (float, N is +)
	Long float64 `json:"long"` // GPS latitude of the gateway in degree (float, E is +)
	Alti int32   `json:"alti"` // GPS altitude of the gateway in meter RX (integer)
	RXNb uint32  `json:"rxnb"` // Number of radio packets received (unsigned integer)
	RXOK uint32  `json:"rxok"` // Number of radio packets received with a valid PHY CRC
	RXFW uint32  `json:"rxfw"` // Number of radio packets forwarded (unsigned integer)
	ACKR float64 `json:"ackr"` // Percentage of upstream datagrams that were acknowledged
	DWNb uint32  `json:"dwnb"` // Number of downlink datagrams received (unsigned integer)
	TXNb uint32  `json:"txnb"` // Number of packets emitted (unsigned integer)
}

type RXPK struct {
	Time      string  `json:"time"` // UTC time of pkt RX, us precision, ISO 8601 'compact' format (e.g. 2013-03-31T16:21:17.528002Z)
	Tmms      *int64  `json:"tmms"` // GPS time of pkt RX, number of milliseconds since 06.Jan.1980
	Tmst      uint32  `json:"tmst"` // Internal timestamp of "RX finished" event (32b unsigned)
	Channel   uint16  `json:"chan"` // Concentrator "IF" channel used for RX (unsigned integer)
	RFCH      uint8   `json:"rfch"` // Concentrator "RF chain" used for RX (unsigned integer)
	Stat      int8    `json:"stat"` // CRC status: 1 = OK, -1 = fail, 0 = no CRC
	Frequency float64 `json:"freq"` // RX central frequency in MHz (unsigned float, Hz precision)
	Brd       uint32  `json:"brd"`  // Concentrator board used for RX (unsigned integer)
	RSSI      int16   `json:"rssi"` // RSSI in dBm (signed integer, 1 dB precision)
	DatR      string  `json:"datr"` // LoRa datarate identifier (eg. SF12BW500) || FSK datarate (unsigned, in bits per second)
	Modu      string  `json:"modu"` // Modulation identifier "LORA" or "FSK"
	CodR      string  `json:"codr"` // LoRa ECC coding rate identifier
	LSNR      float64 `json:"lsnr"` // Lora SNR ratio in dB (signed float, 0.1 dB precision)
	Size      uint16  `json:"size"` // RF packet payload size in bytes (unsigned integer)
	Data      string  `json:"data"` // Base64 encoded RF packet payload, padded
}

func CreatePushDataPacket(GatewayMACAddr lorawan.EUI64, stat Stat, info []RXPK) ([]byte, error) {

	header := GetHeader(TypePushData, GatewayMACAddr, 0)

	payload := PushDataPayload{
		RXPK: info,
		Stat: &stat,
	}

	pkt := PDPacket{
		header,
		payload,
	}

	return pkt.MarshalBinary()

}

func (p *PDPacket) MarshalBinary() ([]byte, error) {

	JSONPayload, err := json.Marshal(p.Payload)

	if err != nil {
		return nil, err
	}

	out := append(p.Header, JSONPayload...)

	return out, nil
}

func GetTime() string {

	t := time.Now().UTC()
	y, mon, d := t.Date()
	h, min, sec := t.Clock()

	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d UTC", y, mon, d, h, min, sec)

}

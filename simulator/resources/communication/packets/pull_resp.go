package packets

import (
	"encoding/binary"
	"encoding/json"
	"errors"

	"github.com/brocaar/lorawan"
)

type PullRespPacket struct {
	Header  Header
	Payload PullRespPayload
}

type PullRespPayload struct {
	TXPK TXPK `json:"txpk"`
}

type TXPK struct {
	Imme bool    `json:"imme"`           // Send packet immediately (will ignore tmst & time)
	RFCh uint8   `json:"rfch"`           // Concentrator "RF chain" used for TX (unsigned integer)
	Powe uint8   `json:"powe"`           // TX output power in dBm (unsigned integer, dBm precision)
	Ant  uint8   `json:"ant"`            // Antenna number on which signal has been received
	Brd  uint32  `json:"brd"`            // Concentrator board used for RX (unsigned integer)
	Tmst *uint32 `json:"tmst,omitempty"` // Send packet on a certain timestamp value (will ignore time)
	Tmms *int64  `json:"tmms,omitempty"` // Send packet at a certain GPS time (GPS synchronization required)
	Freq float64 `json:"freq"`           // TX central frequency in MHz (unsigned float, Hz precision)
	Modu string  `json:"modu"`           // Modulation identifier "LORA" or "FSK"
	DatR string  `json:"datr"`           // LoRa datarate identifier (eg. SF12BW500) || FSK datarate (unsigned, in bits per second)
	CodR string  `json:"codr,omitempty"` // LoRa ECC coding rate identifier
	FDev uint16  `json:"fdev,omitempty"` // FSK frequency deviation (unsigned integer, in Hz)
	NCRC bool    `json:"ncrc,omitempty"` // If true, disable the CRC of the physical layer (optional)
	IPol bool    `json:"ipol"`           // Lora modulation polarization inversion
	Prea uint16  `json:"prea,omitempty"` // RF preamble size (unsigned integer)
	Size uint16  `json:"size"`           // RF packet payload size in bytes (unsigned integer)
	Data []byte  `json:"data"`           // Base64 encoded RF packet payload, padding optional
}

func GetInfoPullResp(pullResp []byte) (*lorawan.PHYPayload, *uint32, error) {

	var phy lorawan.PHYPayload
	var packet PullRespPacket
	var frequency uint32

	//getPacket
	if err := packet.UnmarshalBinary(pullResp); err != nil {
		return nil, nil, err
	}

	frequency = uint32(packet.Payload.TXPK.Freq * 1000000.0)

	//getPayload
	if err := phy.UnmarshalBinary(packet.Payload.TXPK.Data); err != nil {
		return nil, nil, err
	}

	return &phy, &frequency, nil

}

func (p *PullRespPacket) UnmarshalBinary(data []byte) error {

	if len(data) < MinLenPullResp {
		return errors.New("error: short packet( < 5 byte )")
	}
	if data[3] != byte(TypePullResp) {
		return errors.New("it's not a PULL RESP packet")
	}

	tok := []byte{data[1], data[2]}
	p.Header.ProtocolVersion = data[0]

	binary.LittleEndian.PutUint16(tok, p.Header.RandomToken)
	p.Header.IDPacket = data[3]

	return json.Unmarshal(data[4:], &p.Payload)
}

func (p *PullRespPacket) MarshalJSON() ([]byte, error) {

	JSONPayload, err := json.Marshal(p.Payload)
	if err != nil {
		return nil, err
	}

	return JSONPayload, nil
}

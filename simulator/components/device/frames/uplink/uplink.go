package uplink

import (
	"encoding/json"

	"github.com/arslab/lwnsimulator/simulator/components/device/features/adr"
	mac "github.com/arslab/lwnsimulator/simulator/components/device/macCommands"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/brocaar/lorawan"
)

type InfoUplink struct {
	DwellTime     lorawan.DwellTime `json:"-"`
	ClassB        bool              `json:"-"`
	FCnt          uint32            `json:"fcnt"`
	FOpts         []lorawan.Payload `json:"-"`
	FPort         *uint8            `json:"fport"`
	ADR           adr.ADRInfo       `json:"-"`
	AckMacCommand mac.AckMacCommand `json:"-"` //to create new Uplink
}

func (up *InfoUplink) GetFrame(mtype lorawan.MType, payload lorawan.DataPayload,
	devAddr lorawan.DevAddr, AppSKey, NwkSKey [16]byte, ack bool) ([]byte, error) {

	FOpts := up.loadFOpts()

	phy := lorawan.PHYPayload{
		MHDR: lorawan.MHDR{
			MType: mtype,
			Major: lorawan.LoRaWANR1,
		},
		MACPayload: &lorawan.MACPayload{
			FHDR: lorawan.FHDR{
				DevAddr: devAddr,
				FCtrl: lorawan.FCtrl{
					ADR:       up.ADR.ADR,
					ADRACKReq: up.ADR.ADRACKReq,
					ACK:       ack,
					ClassB:    up.ClassB,
				},
				FCnt:  up.FCnt,
				FOpts: FOpts,
			},
			FPort: up.FPort,
			FRMPayload: []lorawan.Payload{
				&payload,
			},
		},
	}

	bytes, err := encryptFrame(phy, AppSKey, NwkSKey)
	if err != nil {
		return []byte{}, err
	}

	up.FCnt = (up.FCnt + 1) % util.MAXFCNTGAP
	up.ADR.ADRACKCnt++

	return bytes, nil

}

func (up *InfoUplink) loadFOpts() []lorawan.Payload {

	FOpts := up.AckMacCommand.GetAll()
	if len(up.FOpts) > 0 {

		if len(up.FOpts)+len(FOpts) < 15 {
			FOpts = append(FOpts, up.FOpts...)
			up.FOpts = up.FOpts[:0] //reset
		} else {
			FOpts = append(FOpts, up.FOpts[:15-len(FOpts)]...)
			up.FOpts = up.FOpts[15-len(FOpts):] //reset
		}

	}

	return FOpts
}

func encryptFrame(phy lorawan.PHYPayload, AppSKey, NwkSKey [16]byte) ([]byte, error) {

	if err := phy.EncryptFRMPayload(AppSKey); err != nil {
		return []byte{}, err
	}

	if err := phy.SetUplinkDataMIC(lorawan.LoRaWAN1_0, 0, 0, 0, NwkSKey, lorawan.AES128Key{}); err != nil {
		return []byte{}, err
	}

	bytes, err := phy.MarshalBinary()
	if err != nil {
		return []byte{}, err
	}

	return bytes, nil
}

func (up *InfoUplink) IsTherePingSlotInfoReq() bool {

	for _, cmd := range up.FOpts {

		cid, _, err := mac.ParseMACCommand(cmd, true)
		if err != nil {
			return false
		}

		if cid == lorawan.PingSlotInfoReq {
			return true
		}
	}

	return false

}

func Fragmentation(size int, payload lorawan.Payload) []lorawan.DataPayload {

	var FRMPayload []lorawan.DataPayload

	payloadBytes, _ := payload.MarshalBinary()

	if size == 0 {
		return FRMPayload
	}

	nFrame := len(payloadBytes) / size

	for i := 0; i <= nFrame; i++ {

		var data lorawan.DataPayload

		offset := i * size

		if i != nFrame {
			data.Bytes = payloadBytes[offset : offset+size]
		} else {
			data.Bytes = payloadBytes[offset:len(payloadBytes)]
		}

		FRMPayload = append(FRMPayload, data)

	}

	return FRMPayload
}

func Truncate(size int, payload lorawan.Payload) lorawan.DataPayload {
	var FRMPayload lorawan.DataPayload

	payloadBytes, _ := payload.MarshalBinary()

	if len(payloadBytes) > size {
		FRMPayload.Bytes = payloadBytes[:size]
	} else {
		FRMPayload.Bytes = payloadBytes
	}

	return FRMPayload
}

//*******************************JSON**************************************/

func (up *InfoUplink) MarshalJSON() ([]byte, error) {

	type Alias InfoUplink

	return json.Marshal(&struct {
		FPort uint8 `json:"fport"`
		*Alias
	}{

		FPort: *up.FPort,
		Alias: (*Alias)(up),
	})

}

func (up *InfoUplink) UnmarshalJSON(data []byte) error {

	type Alias InfoUplink

	aux := &struct {
		FPort uint8 `json:"fport"`
		*Alias
	}{
		Alias: (*Alias)(up),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	up.FPort = &aux.FPort

	return nil
}

package uplink

import (
	"encoding/json"

	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/brocaar/lorawan"
)

func (up *Uplink) GetFrame(mtype lorawan.MType, payload lorawan.DataPayload,
	devAddr lorawan.DevAddr, AppSKey, NwkSKey [16]byte, ack bool) ([]byte, error) {

	FOpts := up.LoadFOpts()

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

	bytes, err := EncryptFrame(phy, AppSKey, NwkSKey)
	if err != nil {
		return []byte{}, err
	}

	up.FCnt = (up.FCnt + 1) % util.MAXFCNTGAP
	up.ADR.ADRACKCnt++

	return bytes, nil

}

func (up *Uplink) LoadFOpts() []lorawan.Payload {

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

func EncryptFrame(phy lorawan.PHYPayload, AppSKey, NwkSKey [16]byte) ([]byte, error) {

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

//*******************************JSON**************************************/

func (up *Uplink) MarshalJSON() ([]byte, error) {

	type Alias Uplink

	return json.Marshal(&struct {
		FPort uint8 `json:"FPort"`
		*Alias
	}{

		FPort: *up.FPort,
		Alias: (*Alias)(up),
	})

}

func (up *Uplink) UnmarshalJSON(data []byte) error {

	type Alias Uplink

	aux := &struct {
		FPort uint8 `json:"FPort"`
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

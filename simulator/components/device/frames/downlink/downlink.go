package downlink

import (
	"errors"

	"github.com/brocaar/lorawan"
)

//Downlink set with info of resp
type InformationDownlink struct {
	MType         lorawan.MType     `json:"-"` //per FPending
	FOptsReceived []lorawan.Payload `json:"-"`
	ACK           bool              `json:"-"`
	DataPayload   []byte            `json:"-"`
	FPending      bool              `json:"-"`
	DwellTime     lorawan.DwellTime `json:"-"`
}

func GetDownlink(phy lorawan.PHYPayload, disableCounter bool, counter uint32, NwkSKey [16]byte, AppSKey [16]byte) (*InformationDownlink, error) {

	var downlink InformationDownlink

	//validate mic
	ok, err := phy.ValidateDownlinkDataMIC(lorawan.LoRaWAN1_0, 0, NwkSKey)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("Invalid MIC")
	}

	macPL, ok := phy.MACPayload.(*lorawan.MACPayload)
	if !ok {
		return nil, errors.New("*MACPayload expected")
	}

	//validate counter
	if !disableCounter {

		if macPL.FHDR.FCnt != counter {
			return nil, errors.New("Invalid downlink counter")
		}

	}

	if err := phy.DecodeFOptsToMACCommands(); err != nil {
		return nil, err
	}

	downlink.MType = phy.MHDR.MType
	downlink.FPending = macPL.FHDR.FCtrl.FPending

	downlink.ACK = macPL.FHDR.FCtrl.ACK

	//MACCommand
	if len(macPL.FHDR.FOpts) != 0 {

		if macPL.FPort == nil || *macPL.FPort != uint8(0) { // MACCommand in Fopts
			downlink.FOptsReceived = append(downlink.FOptsReceived, macPL.FHDR.FOpts...)
		}

	}

	if macPL.FPort != nil {

		switch *macPL.FPort {

		case uint8(0):
			//decrypt frame payload
			if err := phy.DecryptFRMPayload(NwkSKey); err != nil {
				return nil, err
			}

			downlink.FOptsReceived = append(downlink.FOptsReceived, macPL.FRMPayload...)

		default:
			//Datapayload
			if err := phy.DecryptFRMPayload(AppSKey); err != nil {
				return nil, err
			}

			pl, ok := macPL.FRMPayload[0].(*lorawan.DataPayload)
			if !ok {
				return nil, errors.New("*DataPayload expected")
			}

			downlink.DataPayload = pl.Bytes

		}

	}

	return &downlink, nil
}

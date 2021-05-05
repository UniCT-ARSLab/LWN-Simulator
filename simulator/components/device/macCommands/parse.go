package macCommands

import (
	"github.com/brocaar/lorawan"
)

func ParseMACCommand(cmd lorawan.Payload, uplink bool) (lorawan.CID, []byte, error) {

	MacCommand := lorawan.MACCommand{}

	bytesCMD, err := cmd.MarshalBinary() //MAC Command in bytes
	if err != nil {
		return 0x00, nil, err
	}

	err = MacCommand.UnmarshalBinary(uplink, bytesCMD) //insert mac command in struct
	if err != nil {
		return 0x00, nil, err
	}

	if MacCommand.Payload != nil {

		MACCmdPLBytes, err := MacCommand.Payload.MarshalBinary() //Payload in bytes
		if err != nil {
			return 0x00, nil, err
		}

		//create type struct payload
		MACpayload, _, err := lorawan.GetMACPayloadAndSize(uplink, MacCommand.CID)
		if err != nil {
			return 0x00, nil, err
		}

		MACpayload.UnmarshalBinary(MACCmdPLBytes)       //insert mac cmd payload in struct
		bytesPayload, err := MACpayload.MarshalBinary() //mac payload in bytes
		if err != nil {
			return 0x00, nil, err
		}
		return MacCommand.CID, bytesPayload, nil
	}

	return MacCommand.CID, []byte{}, nil
}

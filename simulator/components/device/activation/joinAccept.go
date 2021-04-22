package activation

import (
	"errors"

	"github.com/brocaar/lorawan"
)

func DecryptJoinAccept(phy lorawan.PHYPayload, DevNonce lorawan.DevNonce, JoinEUI lorawan.EUI64, AppKey [16]byte) (*lorawan.JoinAcceptPayload, error) {

	err := phy.DecryptJoinAcceptPayload(AppKey)
	if err != nil {
		return nil, err
	}

	JoinAccPayload, ok := phy.MACPayload.(*lorawan.JoinAcceptPayload)
	if !ok {
		return nil, errors.New("*JoinAcceptPayload expected")
	}

	//validate MIC
	okMIC, err := phy.ValidateDownlinkJoinMIC(lorawan.JoinRequestType, JoinEUI, DevNonce, AppKey)
	if err != nil {
		return nil, err
	}
	if !okMIC {
		return nil, errors.New("Invalid MIC")
	}

	return JoinAccPayload, nil

}

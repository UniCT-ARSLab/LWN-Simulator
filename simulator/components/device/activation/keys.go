package activation

import (
	"crypto/aes"
	"errors"
	"fmt"

	"github.com/brocaar/lorawan"
)

const (
	//PadNwkSKey padding for create NwkSKey
	PadNwkSKey = byte(0x01)
	//PadAppSKey padding for create AppSKey
	PadAppSKey = byte(0x02)
)

func GetKey(NetID lorawan.NetID, JoinNonce lorawan.JoinNonce, DevNonce lorawan.DevNonce,
	AppKey [16]byte, typeKey byte) (lorawan.AES128Key, error) {
	var key lorawan.AES128Key

	src := make([]byte, 16)

	netIDB, err := NetID.MarshalBinary()
	if err != nil {
		return key, err
	}

	joinNonceB, err := JoinNonce.MarshalBinary()
	if err != nil {
		return key, err
	}

	devNonceB, err := DevNonce.MarshalBinary()
	if err != nil {
		return key, err
	}

	src[0] = typeKey
	copy(src[1:4], joinNonceB)
	copy(src[4:7], netIDB)
	copy(src[7:9], devNonceB)
	//il src[15] byte è già settato a 0 di default(padding)

	block, err := aes.NewCipher(AppKey[:])
	if err != nil {
		return key, err
	}

	if block.BlockSize() != len(src) {
		msg := fmt.Sprintf("block-size of %d bytes is expected", len(src))
		return key, errors.New(msg)
	}

	block.Encrypt(key[:], src)
	return key, nil
}

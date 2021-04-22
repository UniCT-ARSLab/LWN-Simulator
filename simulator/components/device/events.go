package device

import (
	"encoding/hex"

	"github.com/arslab/lwnsimulator/socket"
)

func (d *Device) EmitStatus() {

	NwkSKey := hex.EncodeToString(d.Info.NwkSKey[:])
	AppSKey := hex.EncodeToString(d.Info.AppSKey[:])

	data := socket.InfoStatus{
		DevEUI:   d.Info.DevEUI,
		DevAddr:  d.Info.DevAddr,
		NwkSKey:  NwkSKey,
		AppSKey:  AppSKey,
		FCntDown: d.Info.Status.FCntDown,
		FCnt:     d.Info.Status.DataUplink.FCnt,
	}

	d.Resources.WebSocket.Emit(socket.EventSaveStatus, data)

}

package device

import (
	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	"github.com/arslab/lwnsimulator/simulator/util"
)

func (d *Device) DownlinkReceivedRX2ClassC() {

	for d.Class.GetClass() == classes.ClassC {

		if !d.CanExecute() {
			return
		}

		d.Info.Status.InfoClassC.WaitDevice()

		if !d.CanExecute() {
			return
		}

		d.Info.Status.InfoClassC.Mutex.Lock()

		downlink := d.Info.Status.InfoClassC.Downlink

		d.ExecuteMACCommand(downlink)

		d.ADRProcedure()

		if d.Info.Status.Mode != util.Retransmission {
			d.FPendingProcedure(&downlink)
		}

		d.Info.Status.InfoClassC.WakeUpClass()

		d.Info.Status.InfoClassC.Mutex.Unlock()
	}

}

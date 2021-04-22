package device

import "github.com/arslab/lwnsimulator/simulator/components/device/classes"

func (d *Device) DownlinkReceivedRX2ClassC() {

	for d.Mode.GetMode() == classes.ModeC {

		ok := d.CanExecute()
		if !ok {
			return
		}

		d.Info.Status.InfoClassC.WaitDevice()

		ok = d.CanExecute()
		if !ok {
			return
		}

		d.Info.Status.InfoClassC.Mutex.Lock()

		downlink := d.Info.Status.InfoClassC.Downlink

		d.ExecuteMACCommand(downlink)

		d.ADRProcedure()

		if !d.Info.Status.RetransmissionActive {
			d.FPendingProcedure(downlink)
		}

		d.Info.Status.InfoClassC.WakeUpClass()

		d.Info.Status.InfoClassC.Mutex.Unlock()
	}

}

package device

import (
	"errors"
	"fmt"

	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	"github.com/arslab/lwnsimulator/simulator/components/device/features/channels"
	"github.com/arslab/lwnsimulator/simulator/util"
)

func (d *Device) CanExecute() bool {

	if *d.Info.StateSimulator == util.Stopped ||
		!d.Info.Status.Active {

		if d.Mode.GetMode() == classes.ModeC {

			d.Mode.CloseRX2()

			d.Info.Status.InfoClassC.WakeUpDevice()
			d.Info.Status.InfoClassC.WakeUpClass()
			d.Info.Status.InfoClassC.Exit <- struct{}{}

		}

		return false
	}

	return true

}

func (d *Device) IsON() bool { //per una corretta print

	d.Mutex.Lock()
	defer d.Mutex.Unlock()

	return d.Info.Status.Active

}

func (d *Device) isSupportedFrequency(freq uint32) error {
	return d.Info.Configuration.Region.FrequencySupported(freq)
}

func (d *Device) isSupportedDR(dr uint8) error {
	return d.Info.Configuration.Region.DataRateSupported(dr)
}

func (d *Device) isSupportedDataRateRange(minDR uint8, maxDR uint8) error {

	if minDR < d.Info.Configuration.Region.GetMinDataRate() ||
		minDR > d.Info.Configuration.Region.GetMaxDataRate() {
		return errors.New("Invalid Range")
	}

	if maxDR < d.Info.Configuration.Region.GetMinDataRate() ||
		maxDR > d.Info.Configuration.Region.GetMaxDataRate() {
		return errors.New("Invalid Range")
	}

	if minDR > maxDR {
		return errors.New("Invalid Range")
	}

	return nil

}

func (d *Device) setChannel(index uint8, freq uint32, minDR uint8, maxDR uint8) (bool, bool) {

	//first, second and third channel are reserved in EU868
	if index < 3 {

		d.Print("can't modify a reserved channel", nil, util.PrintOnlySocket)

		return false, false
	}

	Fok, DRok := false, false

	err := d.isSupportedFrequency(freq)
	if err == nil {
		Fok = true
	}

	err = d.isSupportedDataRateRange(minDR, maxDR)
	if err != nil {
		return DRok, Fok
	}
	DRok = true

	if int(index) >= len(d.Info.Configuration.Channels) {
		channel := channels.NewChannel(freq, minDR, maxDR)
		//new channel
		d.Info.Configuration.Channels = append(d.Info.Configuration.Channels, channel)
	} else { //update channel
		d.Info.Configuration.Channels[index].UpdateChannel(freq, minDR, maxDR)
	}

	msg := fmt.Sprintf("SET Channel[%v] {F[%v], MinDR[%v], MaxDR[%v] } ", index, freq, minDR, maxDR)
	d.Print(msg, nil, util.PrintOnlySocket)

	return DRok, Fok
}

func (d *Device) setFrequencyDownlink(index uint8, freq uint32) bool {

	if d.Info.Configuration.Channels[index].FrequencyUplink == 0 || index < 3 { //channel non disponibile
		return false
	}

	d.Info.Configuration.Channels[index].FrequencyDownlink = freq

	return true
}

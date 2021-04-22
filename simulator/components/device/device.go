package device

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	res "github.com/arslab/lwnsimulator/simulator/resources"
	"github.com/arslab/lwnsimulator/socket"

	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	"github.com/arslab/lwnsimulator/simulator/components/device/models"
	"github.com/arslab/lwnsimulator/simulator/util"
)

type Device struct {
	Info      models.InformationDevice `json:"Info"`
	Mode      classes.Class            `json:"-"`
	Resources *res.Resources           `json:"-"`
	Mutex     sync.Mutex               `json:"-"`
}

//*******************Intern func*******************/
func (d *Device) Run() {

	d.Print("START", nil, util.PrintBoth)

	//otaa activation
	d.OtaaActivation()

	ticker := time.NewTicker(d.Info.Configuration.SendInterval)

	defer d.SaveStatus()

	for {

		if d.IsON() {

			if d.Info.Status.Joined {

				<-ticker.C

				if ok := d.CanExecute(); ok { //stop

					if d.Info.Configuration.SupportedClassC {
						d.SwitchClass(classes.ModeC)
					} else if d.Info.Configuration.SupportedClassB {
						d.SwitchClass(classes.ModeB)
					}

					d.Execute()

				}

			} else {
				d.OtaaActivation()
			}

		} else {

			d.Print("Turn OFF", nil, util.PrintBoth)
			return
		}

		if *d.Info.StateSimulator == util.Stopped {

			d.Print("STOP", nil, util.PrintBoth)

			return
		}
	}
}

func (d *Device) SaveStatus() {

	var devices []Device

	d.Resources.Mutex.Lock()

	path, err := util.GetPath()
	if err != nil {
		d.Print("", err, util.PrintOnlyConsole)
		return
	}

	pathfile := path + "/devices.json"

	err = util.RecoverConfigFile(pathfile, &devices)
	if err != nil {

		d.Print("", err, util.PrintOnlyConsole)
		return

	}

	for j := range devices {
		if devices[j].Info.DevEUI == d.Info.DevEUI {

			devices[j].Info.DevAddr = d.Info.DevAddr
			devices[j].Info.NwkSKey = d.Info.NwkSKey
			devices[j].Info.AppSKey = d.Info.AppSKey
			//status
			devices[j].Info.Status.Battery = d.Info.Status.Battery

			//counter
			devices[j].Info.Status.DataUplink.FCnt = d.Info.Status.DataUplink.FCnt
			devices[j].Info.Status.FCntDown = d.Info.Status.FCntDown

			devices[j].Info.Status.MType = d.Info.Status.MType
			devices[j].Info.Status.Payload = d.Info.Status.Payload

			devices[j].Info.Location.Latitude = d.Info.Location.Latitude
			devices[j].Info.Location.Longitude = d.Info.Location.Longitude
			devices[j].Info.Location.Altitude = d.Info.Location.Altitude

			d.EmitStatus()

			break
		}
	}

	devBytes, _ := json.MarshalIndent(&devices, "", "\t")

	err = util.WriteConfigFile(pathfile, devBytes)
	if err != nil {
		d.Print("", err, util.PrintOnlyConsole)
		return
	}

	d.Resources.Mutex.Unlock()

	d.Print("Status saved", nil, util.PrintBoth)

	d.Resources.ExitGroup.Done()

}

func (d *Device) Print(content string, err error, printType int) {

	now := time.Now()
	message := ""
	messageLog := ""
	event := socket.EventDev

	if err == nil {
		message = fmt.Sprintf("[ %s ] DEV[%s] {%s}: %s", now.Format(time.Stamp), d.Info.Name, d.Mode.ToString(), content)
		messageLog = fmt.Sprintf(" DEV[%s] {%s}: %s", d.Info.Name, d.Mode.ToString(), content)
	} else {
		message = fmt.Sprintf("[ %s ] DEV[%s] {%s} [ERROR]: %s", now.Format(time.Stamp), d.Info.Name, d.Mode.ToString(), err)
		messageLog = fmt.Sprintf(" DEV[%s] {%s} [ERROR]: %s", d.Info.Name, d.Mode.ToString(), err)
		event = socket.EventError
	}

	data := socket.ConsoleLog{
		Name: d.Info.Name,
		Msg:  message,
	}

	switch printType {
	case util.PrintBoth:
		d.Resources.WebSocket.Emit(event, data)
		log.Println(messageLog)
	case util.PrintOnlySocket:
		d.Resources.WebSocket.Emit(event, data)
	case util.PrintOnlyConsole:
		log.Println(messageLog)
	}

}

package device

import (
	"fmt"
	"sync"
	"time"

	c "github.com/arslab/lwnsimulator/simulator/console"
	res "github.com/arslab/lwnsimulator/simulator/resources"

	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	"github.com/arslab/lwnsimulator/simulator/components/device/models"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/arslab/lwnsimulator/socket"
)

type Device struct {
	State     int                      `json:"-"`
	Exit      chan struct{}            `json:"-"`
	Id        int                      `json:"id"`
	Info      models.InformationDevice `json:"info"`
	Class     classes.Class            `json:"-"`
	Resources *res.Resources           `json:"-"`
	Mutex     sync.Mutex               `json:"-"`
	Console   c.Console                `json:"-"`
}

// *******************Intern func*******************/
func (d *Device) Run() {

	defer d.Resources.ExitGroup.Done()

	d.OtaaActivation()

	ticker := time.NewTicker(d.Info.Configuration.SendInterval)

	for {

		select {

		case <-ticker.C:
			break

		case <-d.Exit:
			d.Print("Turn OFF", nil, util.PrintBoth)
			return
		}

		if d.CanExecute() {

			if d.Info.Status.Joined {

				if d.Info.Configuration.SupportedClassC {
					d.SwitchClass(classes.ClassC)
				} else if d.Info.Configuration.SupportedClassB {
					d.SwitchClass(classes.ClassB)
				}

				d.Execute()

			} else {
				d.OtaaActivation()

				d.Info.Status.DoSwitchChannel = true
			}
		}

	}

}

func (d *Device) modeToString() string {

	switch d.Info.Status.Mode {

	case util.Normal:
		return "Normal"

	case util.Retransmission:
		return "Retransmission"

	case util.FPending:
		return "FPending"

	case util.Activation:
		return "Activation"

	default:
		return ""

	}
}

func (d *Device) Print(content string, err error, printType int) {

	now := time.Now()
	message := ""
	messageLog := ""
	event := socket.EventDev
	class := d.Class.ToString()
	mode := d.modeToString()

	if err == nil {
		message = fmt.Sprintf("[ %s ] DEV[%s] |%s| {%s}: %s", now.Format(time.Stamp), d.Info.Name, mode, class, content)
		messageLog = fmt.Sprintf("DEV[%s] |%s| {%s}: %s", d.Info.Name, mode, class, content)
	} else {
		message = fmt.Sprintf("[ %s ] DEV[%s] |%s| {%s} [ERROR]: %s", now.Format(time.Stamp), d.Info.Name, mode, class, err)
		messageLog = fmt.Sprintf("DEV[%s] |%s| {%s} [ERROR]: %s", d.Info.Name, mode, class, err)
		event = socket.EventError
	}

	data := socket.ConsoleLog{
		Name: d.Info.Name,
		Msg:  message,
	}

	switch printType {
	case util.PrintBoth:
		d.Console.PrintSocket(event, data)
		d.Console.PrintLog(messageLog)
	case util.PrintOnlySocket:
		d.Console.PrintSocket(event, data)
	case util.PrintOnlyConsole:
		d.Console.PrintLog(messageLog)
	}
}

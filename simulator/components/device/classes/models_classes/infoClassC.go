package models_classes

import (
	"sync"

	dl "github.com/arslab/lwnsimulator/simulator/components/device/frames/downlink"
)

type InfoClassC struct {
	Downlink   dl.InformationDownlink
	Mutex      sync.Mutex
	CondClass  *sync.Cond
	CondDevice *sync.Cond
	ACK        bool
}

func (i *InfoClassC) Setup() {
	i.CondClass = sync.NewCond(&i.Mutex)
	i.CondDevice = sync.NewCond(&i.Mutex)
}

func (i *InfoClassC) InsertDownlink(downlink dl.InformationDownlink) {
	i.Mutex.Lock()
	i.Downlink = downlink
	i.Mutex.Unlock()
}

func (i *InfoClassC) SetACK(value bool) {
	i.Mutex.Lock()
	i.ACK = value
	i.Mutex.Unlock()
}

func (i *InfoClassC) GetACK() bool {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()

	return i.ACK

}

func (i *InfoClassC) WaitClass() {
	i.Mutex.Lock()
	i.CondClass.Wait()
	i.Mutex.Unlock()
}

func (i *InfoClassC) WakeUpClass() {
	i.Mutex.Lock()
	i.CondClass.Broadcast()
	i.Mutex.Unlock()
}

func (i *InfoClassC) WaitDevice() {
	i.Mutex.Lock()
	i.CondDevice.Wait()
	i.Mutex.Unlock()
}

func (i *InfoClassC) WakeUpDevice() {
	i.Mutex.Lock()
	i.CondDevice.Broadcast()
	i.Mutex.Unlock()
}

package device

import (
	"github.com/arslab/lwnsimulator/simulator/components/device/classes"
	"github.com/arslab/lwnsimulator/simulator/util"
	"github.com/brocaar/lorawan"
)

func Fragmentation(size int, payload lorawan.Payload) []lorawan.DataPayload {

	var FRMPayload []lorawan.DataPayload

	payloadBytes, _ := payload.MarshalBinary()
	nFrame := len(payloadBytes) / size

	for i := 0; i <= nFrame; i++ {

		var data lorawan.DataPayload

		offset := i * size

		if i != nFrame {
			data.Bytes = payloadBytes[offset : offset+size]
		} else {
			data.Bytes = payloadBytes[offset:len(payloadBytes)]
		}

		FRMPayload = append(FRMPayload, data)

	}

	return FRMPayload
}

func Truncate(size int, payload lorawan.Payload) lorawan.DataPayload {
	var FRMPayload lorawan.DataPayload

	payloadBytes, _ := payload.MarshalBinary()

	if len(payloadBytes) > size {
		FRMPayload.Bytes = payloadBytes[:size]
	} else {
		FRMPayload.Bytes = payloadBytes
	}

	return FRMPayload
}

func (d *Device) CreateUplink() [][]byte {

	var mtype lorawan.MType
	var payload lorawan.Payload

	if d.Mode.GetMode() == classes.ModeB {
		d.Info.Status.DataUplink.ClassB = true
	} else {
		d.Info.Status.DataUplink.ClassB = false
	}

	if d.Info.Status.RetransmissionActive { //ritrasmissione

		return d.Info.Status.LastUplinks

	} else { //new uplink

		if len(d.Info.Status.BufferUplinks) > 0 {

			mtype = d.Info.Status.BufferUplinks[0].MType
			payload = d.Info.Status.BufferUplinks[0].Payload

			switch len(d.Info.Status.BufferUplinks) {
			case 1:
				d.Info.Status.BufferUplinks = d.Info.Status.BufferUplinks[:0]

			default:
				d.Info.Status.BufferUplinks = d.Info.Status.BufferUplinks[1:]

			}

		} else {
			mtype = d.Info.Status.MType
			payload = d.Info.Status.Payload
		}

		d.Info.Status.LastMType = mtype

	}

	m, n := d.Info.Configuration.Region.GetPayloadSize(d.Info.Status.DataRate, d.Info.Status.DataUplink.DwellTime)

	var DataPayload []lorawan.DataPayload
	if d.Info.Configuration.SupportedFragment { //frammentazione

		if len(d.Info.Status.DataUplink.FOpts) > 0 {
			DataPayload = Fragmentation(n, payload)
		} else {
			DataPayload = Fragmentation(m, payload)
		}

	} else { //troncamento

		if len(d.Info.Status.DataUplink.FOpts) > 0 {
			DataPayload = append(DataPayload, Truncate(n, payload))
		} else {
			DataPayload = append(DataPayload, Truncate(m, payload))
		}
	}

	var frames [][]byte

	for i := 0; i < len(DataPayload); i++ {
		frame, err := d.Info.Status.DataUplink.GetFrame(mtype, DataPayload[i], d.Info.DevAddr, d.Info.AppSKey, d.Info.NwkSKey, false)
		if err != nil {
			d.Print("", err, util.PrintBoth)
			continue
		}
		frames = append(frames, frame)
	}

	d.Info.Status.LastUplinks = frames

	return frames
}

func (d *Device) CreateACK() []byte {

	var emptyPayload lorawan.DataPayload

	frame, err := d.Info.Status.DataUplink.GetFrame(lorawan.UnconfirmedDataUp, emptyPayload, d.Info.DevAddr, d.Info.AppSKey, d.Info.NwkSKey, true)
	if err != nil {
		d.Print("", err, util.PrintBoth)
		return []byte{}
	}

	return frame

}

func (d *Device) CreateEmptyFrame() []byte {

	var emptyPayload lorawan.DataPayload

	frame, err := d.Info.Status.DataUplink.GetFrame(lorawan.UnconfirmedDataUp, emptyPayload, d.Info.DevAddr, d.Info.AppSKey, d.Info.NwkSKey, false)
	if err != nil {
		d.Print("", err, util.PrintBoth)
		return []byte{}
	}

	return frame

}

func (d *Device) SendEmptyFrame() {

	emptyFrame := d.CreateEmptyFrame()
	info := d.SetInfo(emptyFrame, false)

	d.Mode.SendData(info)

	d.Print("Empty Frame sent", nil, util.PrintOnlySocket)
}

func (d *Device) SendAck() {

	ack := d.CreateACK()
	info := d.SetInfo(ack, false)

	d.Mode.SendData(info)

	d.Print("ACK sent", nil, util.PrintOnlySocket)
}

func (d *Device) SendJoinRequest() {

	JoinRequest := d.CreateJoinRequest()
	info := d.SetInfo(JoinRequest, true)

	d.Mode.SendData(info)

	d.Print("JOIN REQUEST sent", nil, util.PrintOnlySocket)
}

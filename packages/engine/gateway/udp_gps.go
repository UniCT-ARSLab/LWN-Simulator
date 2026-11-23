package gateway

import (
	"context"
	"errors"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

const gpsEpochUnixMilliseconds = 315964800000

// Four 128-second Class B beacon periods bound the virtual transmission queue.
const maxGPSDownlinkAdvance = 512 * time.Second

// UTC instants when the GPS-UTC offset increased by one second, from the
// IERS leap-seconds.list distributed with the IANA time zone database.
var gpsLeapSeconds = [...]int64{
	362793600, 394329600, 425865600, 489024000, 567993600, 631152000,
	662688000, 709948800, 741484800, 773020800, 820454400, 867715200,
	915148800, 1136073600, 1230768000, 1341100800, 1435708800, 1483228800,
}

func gpsMilliseconds(utc time.Time) uint64 {
	unix := utc.UnixMilli()
	leaps := sort.Search(len(gpsLeapSeconds), func(index int) bool {
		return gpsLeapSeconds[index]*1000 > unix
	})
	return uint64(unix - gpsEpochUnixMilliseconds + int64(leaps)*1000)
}

func timeFromGPSMilliseconds(value uint64) (time.Time, bool) {
	if value > math.MaxInt64-gpsEpochUnixMilliseconds {
		return time.Time{}, false
	}
	unix := int64(value) + gpsEpochUnixMilliseconds
	leaps := sort.Search(len(gpsLeapSeconds), func(index int) bool {
		return (gpsLeapSeconds[index]+int64(index+1))*1000 > unix
	})
	utc := time.UnixMilli(unix - int64(leaps)*1000).UTC()
	// A UTC leap second itself is not representable by time.Time.
	return utc, gpsMilliseconds(utc) == value
}

type gpsDownlinkQueue struct {
	timer   *time.Timer
	packets []types.GatewayPacket
}

func (queue *gpsDownlinkQueue) deadline() <-chan time.Time {
	if len(queue.packets) == 0 {
		return nil
	}
	return queue.timer.C
}

func (queue *gpsDownlinkQueue) add(packet types.GatewayPacket) {
	index := sort.Search(len(queue.packets), func(index int) bool {
		return queue.packets[index].TransmitAt.After(packet.TransmitAt)
	})
	queue.packets = slices.Insert(queue.packets, index, packet)
	if index == 0 {
		queue.scheduleNext()
	}
}

func (queue *gpsDownlinkQueue) pop() types.GatewayPacket {
	packet := queue.packets[0]
	queue.packets = slices.Delete(queue.packets, 0, 1)
	if len(queue.packets) > 0 {
		queue.scheduleNext()
	}
	return packet
}

func (queue *gpsDownlinkQueue) scheduleNext() {
	delay := time.Until(queue.packets[0].TransmitAt)
	if queue.timer == nil {
		queue.timer = time.NewTimer(delay)
		return
	}
	queue.timer.Reset(delay)
}

func (queue *gpsDownlinkQueue) close() {
	if queue.timer != nil {
		queue.timer.Stop()
	}
}

func (adapter *udpAdapter) gpsDownlinkStatus(packet types.GatewayPacket) (string, error) {
	advance := time.Until(packet.TransmitAt)
	switch {
	case advance <= 0:
		return "TOO_LATE", errors.New("GPS transmission time has passed")
	case advance > maxGPSDownlinkAdvance:
		return "TOO_EARLY", errors.New("GPS transmission is too far in advance")
	case len(adapter.gpsDownlinks.packets) >= adapter.options.PacketBuffer:
		return "COLLISION_PACKET", errors.New("GPS transmission queue is full")
	default:
		return "NONE", nil
	}
}

func (adapter *udpAdapter) deliverDownlink(ctx context.Context, packet types.GatewayPacket) error {
	select {
	case adapter.packets <- packet:
		if adapter.virtual {
			adapter.recordDownlink(packet.TransmitAt)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

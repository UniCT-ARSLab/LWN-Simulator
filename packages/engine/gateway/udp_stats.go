package gateway

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

const defaultStatisticsInterval = 30 * time.Second

type gatewayStatistics struct {
	Time           string   `json:"time"`
	Latitude       *float32 `json:"lati,omitempty"`
	Longitude      *float32 `json:"long,omitempty"`
	Altitude       *int     `json:"alti,omitempty"`
	RadioReceived  uint32   `json:"rxnb"`
	ValidCRC       uint32   `json:"rxok"`
	RadioForwarded uint32   `json:"rxfw"`
	AckPercentage  float64  `json:"ackr"`
	Downlinks      uint32   `json:"dwnb"`
	Transmitted    uint32   `json:"txnb"`
}

type gatewayPosition struct {
	latitude, longitude *float32
	altitude            *int
}

type pushTraffic struct {
	received, forwarded, sent, acknowledged uint32
}

type udpStatistics struct {
	position             gatewayPosition
	push                 pushTraffic
	pushTokens           map[[2]byte]struct{}
	downlinks            uint32
	transmitted          uint32
	pendingTransmissions []time.Time
}

func statisticsInterval(gateway contracts.Gateway) time.Duration {
	if gateway.KeepAlive != nil && *gateway.KeepAlive > 0 {
		return time.Duration(*gateway.KeepAlive) * time.Second
	}
	return defaultStatisticsInterval
}

// SetGatewayPosition refreshes the location advertised by the virtual forwarder.
func (adapter *udpAdapter) SetGatewayPosition(latitude, longitude, altitude *float32) {
	position := gatewayPosition{}
	if latitude != nil && longitude != nil && altitude != nil {
		lat, lng, alt := *latitude, *longitude, int(math.Round(float64(*altitude)))
		position = gatewayPosition{latitude: &lat, longitude: &lng, altitude: &alt}
	}
	adapter.mu.Lock()
	adapter.statistics.position = position
	adapter.mu.Unlock()
}

func (adapter *udpAdapter) sendPushData(ctx context.Context, payload pushDataPayload) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode PUSH_DATA: %w", err)
	}
	eui, err := hex.DecodeString(adapter.gatewayEUI)
	if err != nil {
		return fmt.Errorf("decode gateway EUI: %w", err)
	}
	if !adapter.virtual {
		token, err := randomToken()
		if err != nil {
			return fmt.Errorf("create PUSH_DATA token: %w", err)
		}
		adapter.mu.Lock()
		destination := adapter.lastPeer
		if destination == nil {
			destination = adapter.remote
		}
		adapter.mu.Unlock()
		return adapter.send(append(append(packetHeader(token, pushDataType), eui...), body...), destination)
	}
	adapter.mu.Lock()
	token, err := adapter.reservePushTokenLocked()
	if err != nil {
		adapter.mu.Unlock()
		return err
	}
	adapter.statistics.push.sent++
	received := uint32(len(payload.RXPK))
	adapter.statistics.push.received += received
	adapter.statistics.push.forwarded += received
	adapter.mu.Unlock()

	frame := append(packetHeader(token, pushDataType), eui...)
	frame = append(frame, body...)
	if err := adapter.send(frame, adapter.remote); err != nil {
		adapter.mu.Lock()
		if _, pending := adapter.statistics.pushTokens[token]; pending {
			delete(adapter.statistics.pushTokens, token)
			adapter.statistics.push.sent--
			adapter.statistics.push.forwarded -= received
		}
		adapter.mu.Unlock()
		return err
	}
	return nil
}

func (adapter *udpAdapter) reservePushTokenLocked() ([2]byte, error) {
	if adapter.statistics.pushTokens == nil {
		adapter.statistics.pushTokens = make(map[[2]byte]struct{})
	}
	for range 16 {
		token, err := randomToken()
		if err != nil {
			return token, fmt.Errorf("create PUSH_DATA token: %w", err)
		}
		if _, pending := adapter.statistics.pushTokens[token]; !pending {
			adapter.statistics.pushTokens[token] = struct{}{}
			return token, nil
		}
	}
	return [2]byte{}, errors.New("could not reserve a unique PUSH_DATA token")
}

func (adapter *udpAdapter) sendStatistics(ctx context.Context, now time.Time) error {
	adapter.mu.Lock()
	stats := &adapter.statistics
	pending := stats.pendingTransmissions[:0]
	for _, transmission := range stats.pendingTransmissions {
		if transmission.After(now) {
			pending = append(pending, transmission)
		} else {
			stats.transmitted++
		}
	}
	stats.pendingTransmissions = pending
	traffic := stats.push
	report := gatewayStatistics{
		Time:     now.UTC().Format("2006-01-02 15:04:05 GMT"),
		Latitude: stats.position.latitude, Longitude: stats.position.longitude, Altitude: stats.position.altitude,
		RadioReceived: traffic.received, ValidCRC: traffic.received, RadioForwarded: traffic.forwarded,
		Downlinks: stats.downlinks, Transmitted: stats.transmitted,
	}
	if traffic.sent > 0 {
		report.AckPercentage = 100 * float64(traffic.acknowledged) / float64(traffic.sent)
	}
	stats.push = pushTraffic{}
	stats.downlinks = 0
	stats.transmitted = 0
	clear(stats.pushTokens)
	adapter.mu.Unlock()

	if err := adapter.sendPushData(ctx, pushDataPayload{Stat: &report}); err != nil {
		adapter.mu.Lock()
		stats.push.received += traffic.received
		stats.push.forwarded += traffic.forwarded
		stats.push.sent += traffic.sent
		stats.push.acknowledged += traffic.acknowledged
		stats.downlinks += report.Downlinks
		stats.transmitted += report.Transmitted
		adapter.mu.Unlock()
		return fmt.Errorf("send gateway statistics: %w", err)
	}
	return nil
}

func (adapter *udpAdapter) acceptBridgeDatagram(datagram udpDatagram) bool {
	data := datagram.data
	if len(data) < 4 || data[0] != protocolVersion || datagram.remote == nil ||
		!adapter.remote.IP.Equal(datagram.remote.IP) || adapter.remote.Port != datagram.remote.Port ||
		adapter.remote.Zone != datagram.remote.Zone {
		return false
	}
	switch data[3] {
	case pullResponseType:
		return true
	case pushAckType, pullAckType:
		if len(data) != 4 {
			return false
		}
		token := [2]byte{data[1], data[2]}
		adapter.mu.Lock()
		defer adapter.mu.Unlock()
		if data[3] == pullAckType {
			if !adapter.pullPending || token != adapter.pullToken {
				return false
			}
			adapter.pullPending = false
		} else {
			if _, pending := adapter.statistics.pushTokens[token]; !pending {
				return false
			}
			delete(adapter.statistics.pushTokens, token)
			adapter.statistics.push.acknowledged++
		}
		return true
	default:
		return false
	}
}

func (adapter *udpAdapter) recordDownlink(transmitAt time.Time) {
	adapter.mu.Lock()
	defer adapter.mu.Unlock()
	stats := &adapter.statistics
	if transmitAt.After(time.Now()) {
		stats.pendingTransmissions = append(stats.pendingTransmissions, transmitAt)
	} else {
		stats.transmitted++
	}
}

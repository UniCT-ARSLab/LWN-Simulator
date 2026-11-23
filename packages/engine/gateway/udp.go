package gateway

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/Giuseppe-Compagnone/lwn-engine/types"
)

const (
	protocolVersion  byte = 2
	pushDataType     byte = 0x00
	pushAckType      byte = 0x01
	pullDataType     byte = 0x02
	pullResponseType byte = 0x03
	pullAckType      byte = 0x04
	txAckType        byte = 0x05
)

var (
	ErrInvalidGatewayEndpoint = errors.New("invalid gateway UDP endpoint")
	ErrGatewayNotConnected    = errors.New("gateway UDP adapter is not connected")
	ErrGatewayHeartbeat       = errors.New("gateway heartbeat timeout")
	errGatewayGPSUnlocked     = errors.New("GPS-timed transmission requires a synchronized GPS clock")
)

type UDPOptions struct {
	LocalAddress      string
	HeartbeatInterval time.Duration
	Timeout           time.Duration
	ReconnectInitial  time.Duration
	ReconnectMax      time.Duration
	PacketBuffer      int
	EventBuffer       int
}

func (options UDPOptions) withDefaults() UDPOptions {
	if options.LocalAddress == "" {
		options.LocalAddress = ":1700"
	}
	if options.HeartbeatInterval <= 0 {
		options.HeartbeatInterval = 5 * time.Second
	}
	if options.Timeout <= 0 {
		options.Timeout = options.HeartbeatInterval * 3
	}
	if options.ReconnectInitial <= 0 {
		options.ReconnectInitial = time.Second
	}
	if options.ReconnectMax <= 0 {
		options.ReconnectMax = 30 * time.Second
	}
	if options.PacketBuffer <= 0 {
		options.PacketBuffer = 64
	}
	if options.EventBuffer <= 0 {
		options.EventBuffer = 64
	}
	return options
}

type UDPFactory struct {
	mu           sync.Mutex
	options      UDPOptions
	runtime      *udpRuntime
	bridgeRemote *net.UDPAddr
}

type udpRuntime struct {
	conn     *net.UDPConn
	cancel   context.CancelFunc
	done     chan struct{}
	adapters map[string]*udpAdapter
}

func NewUDPFactory(options UDPOptions) *UDPFactory {
	return &UDPFactory{options: options.withDefaults()}
}

// ConfigureGatewayBridge applies the external Gateway Bridge endpoint selected
// for the next simulation run. The factory is configured before adapters are
// started; a running factory must be closed by the engine before it can be
// reconfigured.
func (factory *UDPFactory) ConfigureGatewayBridge(config contracts.GatewayBridgeConfig) error {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	if factory.runtime != nil {
		return errors.New("cannot configure gateway bridge while it is running")
	}
	if !config.Enabled {
		factory.bridgeRemote = nil
		return nil
	}
	remote, err := gatewayBridgeEndpoint(config)
	if err != nil {
		return err
	}
	factory.bridgeRemote = remote
	return nil
}

func gatewayBridgeEndpoint(config contracts.GatewayBridgeConfig) (*net.UDPAddr, error) {
	address := strings.TrimSpace(config.Address)
	if address == "" {
		return nil, fmt.Errorf("%w: bridge address is required", ErrInvalidGatewayEndpoint)
	}
	if parsed, err := url.Parse(address); err == nil && parsed.Hostname() != "" {
		address = parsed.Hostname()
	}
	if config.Port < 1 || config.Port > 65535 {
		return nil, fmt.Errorf("%w: invalid gateway bridge port: %d", ErrInvalidGatewayEndpoint, config.Port)
	}
	if net.ParseIP(address) != nil && net.ParseIP(address).IsUnspecified() {
		return nil, fmt.Errorf("%w: bridge address must be routable", ErrInvalidGatewayEndpoint)
	}
	if strings.Contains(address, "/") || strings.ContainsAny(address, " \t\r\n") {
		return nil, fmt.Errorf("%w: invalid gateway bridge address %q", ErrInvalidGatewayEndpoint, address)
	}
	remote, err := net.ResolveUDPAddr("udp", net.JoinHostPort(address, strconv.Itoa(int(config.Port))))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidGatewayEndpoint, err)
	}
	return remote, nil
}

func (factory *UDPFactory) LocalAddress() string {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	return factory.options.LocalAddress
}

func cloneUDPAddr(address *net.UDPAddr) *net.UDPAddr {
	if address == nil {
		return nil
	}
	clone := *address
	clone.IP = append(net.IP(nil), address.IP...)
	return &clone
}

func (factory *UDPFactory) SupportsVirtualGateways() bool {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	return factory.bridgeRemote != nil
}

func (factory *UDPFactory) NewGatewayAdapter(gateway contracts.Gateway) (types.GatewayAdapter, error) {
	if len(gateway.GatewayEUI) != 16 {
		return nil, fmt.Errorf("%w: gateway EUI must contain 16 hexadecimal characters", ErrInvalidGatewayEndpoint)
	}
	if _, err := hex.DecodeString(gateway.GatewayEUI); err != nil {
		return nil, fmt.Errorf("%w: gateway EUI is invalid", ErrInvalidGatewayEndpoint)
	}

	factory.mu.Lock()
	options := factory.options
	bridgeRemote := cloneUDPAddr(factory.bridgeRemote)
	factory.mu.Unlock()
	remote := bridgeRemote
	virtual := gateway.Type == contracts.Virtual
	if gateway.Type == contracts.Real {
		if gateway.GatewayIPv4 == nil || net.ParseIP(*gateway.GatewayIPv4).To4() == nil {
			return nil, fmt.Errorf("%w: gateway IPv4 is invalid", ErrInvalidGatewayEndpoint)
		}
		if gateway.GatewayPort == nil || *gateway.GatewayPort < 1 || *gateway.GatewayPort > 65535 {
			return nil, fmt.Errorf("%w: gateway port is invalid", ErrInvalidGatewayEndpoint)
		}
		var err error
		remote, err = net.ResolveUDPAddr("udp", net.JoinHostPort(*gateway.GatewayIPv4, strconv.Itoa(int(*gateway.GatewayPort))))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidGatewayEndpoint, err)
		}
	} else if gateway.Type != contracts.Virtual {
		return nil, fmt.Errorf("gateway %s has unsupported type %q", gateway.ID, gateway.Type)
	} else if remote == nil {
		return nil, fmt.Errorf("%w: virtual gateway bridge is not configured", ErrGatewayNotConnected)
	}
	adapter := &udpAdapter{
		factory:            factory,
		gatewayID:          gateway.ID,
		gatewayEUI:         strings.ToUpper(gateway.GatewayEUI),
		remote:             remote,
		virtual:            virtual,
		options:            options,
		inbound:            make(chan udpDatagram, options.PacketBuffer),
		packets:            make(chan types.GatewayPacket, options.PacketBuffer),
		events:             make(chan types.GatewayAdapterEvent, options.EventBuffer),
		failures:           make(chan error, 4),
		done:               make(chan struct{}),
		pending:            make(map[string]types.GatewayPacket),
		statisticsInterval: statisticsInterval(gateway),
	}
	if virtual {
		adapter.SetGatewayPosition(gateway.Latitude, gateway.Longitude, gateway.Altitude)
	}
	return adapter, nil
}

func (factory *UDPFactory) register(adapter *udpAdapter) error {
	factory.mu.Lock()
	defer factory.mu.Unlock()

	if factory.runtime == nil {
		local, err := net.ResolveUDPAddr("udp4", factory.options.LocalAddress)
		if err != nil {
			return fmt.Errorf("resolve UDP listen address: %w", err)
		}
		conn, err := net.ListenUDP("udp4", local)
		if err != nil {
			return fmt.Errorf("listen for gateway UDP packets: %w", err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		runtime := &udpRuntime{
			conn:     conn,
			cancel:   cancel,
			done:     make(chan struct{}),
			adapters: make(map[string]*udpAdapter),
		}
		factory.runtime = runtime
		go factory.readLoop(ctx, runtime)
	}
	if _, exists := factory.runtime.adapters[adapter.gatewayEUI]; exists {
		return fmt.Errorf("gateway EUI %s is already registered", adapter.gatewayEUI)
	}
	factory.runtime.adapters[adapter.gatewayEUI] = adapter
	return nil
}

func (factory *UDPFactory) unregister(adapter *udpAdapter) {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	if factory.runtime == nil {
		return
	}
	delete(factory.runtime.adapters, adapter.gatewayEUI)
	if len(factory.runtime.adapters) != 0 {
		return
	}
	runtime := factory.runtime
	factory.runtime = nil
	runtime.cancel()
	_ = runtime.conn.Close()
}

func (factory *UDPFactory) reconnect(adapter *udpAdapter) error {
	factory.unregister(adapter)
	return factory.register(adapter)
}

func (factory *UDPFactory) send(adapter *udpAdapter, data []byte, destination *net.UDPAddr) error {
	factory.mu.Lock()
	defer factory.mu.Unlock()
	if factory.runtime == nil {
		return ErrGatewayNotConnected
	}
	if destination == nil {
		destination = adapter.remote
	}
	if _, err := factory.runtime.conn.WriteToUDP(data, destination); err != nil {
		return fmt.Errorf("send gateway UDP packet: %w", err)
	}
	return nil
}

func (factory *UDPFactory) Close() error {
	factory.mu.Lock()
	runtime := factory.runtime
	adapters := make([]*udpAdapter, 0)
	if runtime != nil {
		for _, adapter := range runtime.adapters {
			adapters = append(adapters, adapter)
		}
	}
	factory.mu.Unlock()
	for _, adapter := range adapters {
		_ = adapter.Close()
	}

	if runtime != nil {
		factory.mu.Lock()
		if factory.runtime == runtime {
			factory.runtime = nil
			runtime.cancel()
			_ = runtime.conn.Close()
		}
		factory.mu.Unlock()
		<-runtime.done
	}
	return nil
}

func (factory *UDPFactory) readLoop(ctx context.Context, runtime *udpRuntime) {
	defer close(runtime.done)
	buffer := make([]byte, 64*1024)
	for {
		if ctx.Err() != nil {
			return
		}
		if err := runtime.conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond)); err != nil {
			factory.broadcastError(runtime, err)
			return
		}
		size, remote, err := runtime.conn.ReadFromUDP(buffer)
		if err != nil {
			var netError net.Error
			if errors.As(err, &netError) && netError.Timeout() {
				continue
			}
			factory.broadcastError(runtime, err)
			return
		}
		factory.routeDatagram(ctx, runtime, remote, append([]byte(nil), buffer[:size]...))
	}
}

func (factory *UDPFactory) routeDatagram(ctx context.Context, runtime *udpRuntime, remote *net.UDPAddr, data []byte) {
	if len(data) < 4 || data[0] != protocolVersion {
		factory.broadcastError(runtime, errors.New("invalid Semtech UDP packet header"))
		return
	}

	factory.mu.Lock()
	if factory.runtime != runtime {
		factory.mu.Unlock()
		return
	}
	var adapter *udpAdapter
	if len(data) >= 12 && (data[3] == pushDataType || data[3] == pullDataType || data[3] == txAckType) {
		gatewayEUI := strings.ToUpper(hex.EncodeToString(data[4:12]))
		adapter = runtime.adapters[gatewayEUI]
	} else {
		for _, candidate := range runtime.adapters {
			if candidate.remote.IP.Equal(remote.IP) && candidate.remote.Port == remote.Port {
				adapter = candidate
				break
			}
			if adapter == nil && candidate.remote.IP.Equal(remote.IP) {
				adapter = candidate
			}
		}
	}
	factory.mu.Unlock()
	if adapter == nil {
		return
	}

	switch data[3] {
	case pushDataType:
		_ = factory.writeAck(runtime.conn, remote, data[1:3], pushAckType)
	case pullDataType:
		_ = factory.writeAck(runtime.conn, remote, data[1:3], pullAckType)
	}
	select {
	case adapter.inbound <- udpDatagram{data: data, remote: remote}:
	case <-ctx.Done():
	}
}

func (factory *UDPFactory) writeAck(conn *net.UDPConn, remote *net.UDPAddr, token []byte, packetType byte) error {
	if len(token) != 2 {
		return errors.New("invalid Semtech UDP token")
	}
	_, err := conn.WriteToUDP([]byte{protocolVersion, token[0], token[1], packetType}, remote)
	return err
}

func (factory *UDPFactory) broadcastError(runtime *udpRuntime, err error) {
	factory.mu.Lock()
	adapters := make([]*udpAdapter, 0, len(runtime.adapters))
	for _, adapter := range runtime.adapters {
		adapters = append(adapters, adapter)
	}
	factory.mu.Unlock()
	for _, adapter := range adapters {
		select {
		case adapter.failures <- err:
		default:
		}
	}
}

type udpAdapter struct {
	factory            *UDPFactory
	gatewayID          string
	gatewayEUI         string
	remote             *net.UDPAddr
	virtual            bool
	options            UDPOptions
	statisticsInterval time.Duration

	mu              sync.Mutex
	cancel          context.CancelFunc
	conn            *net.UDPConn
	started         bool
	closed          bool
	lastPeer        *net.UDPAddr
	clockTimestamp  uint32
	clockReceivedAt time.Time
	pullToken       [2]byte
	pullPending     bool
	statistics      udpStatistics
	gpsDownlinks    gpsDownlinkQueue

	inbound  chan udpDatagram
	packets  chan types.GatewayPacket
	events   chan types.GatewayAdapterEvent
	failures chan error
	done     chan struct{}
	pending  map[string]types.GatewayPacket
}

type udpDatagram struct {
	data   []byte
	remote *net.UDPAddr
}

func (adapter *udpAdapter) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("gateway adapter context cannot be nil")
	}
	adapter.mu.Lock()
	if adapter.closed {
		adapter.mu.Unlock()
		return errors.New("gateway adapter is closed")
	}
	if adapter.started {
		adapter.mu.Unlock()
		return errors.New("gateway adapter is already started")
	}
	if adapter.virtual {
		network := "udp4"
		listenIP := net.IPv4zero
		if adapter.remote != nil && adapter.remote.IP.To4() == nil {
			network = "udp6"
			listenIP = net.IPv6zero
		}
		conn, err := net.ListenUDP(network, &net.UDPAddr{IP: listenIP, Port: 0})
		if err != nil {
			adapter.mu.Unlock()
			return fmt.Errorf("listen virtual gateway UDP socket: %w", err)
		}
		adapter.conn = conn
	} else if err := adapter.factory.register(adapter); err != nil {
		adapter.mu.Unlock()
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	adapter.cancel = cancel
	adapter.started = true
	adapter.mu.Unlock()
	go adapter.run(runCtx)
	if adapter.virtual {
		go adapter.readVirtualLoop(runCtx)
	}
	return nil
}

func (adapter *udpAdapter) Send(ctx context.Context, packet types.GatewayPacket) error {
	if ctx == nil {
		return errors.New("gateway packet context cannot be nil")
	}
	adapter.mu.Lock()
	clockTimestamp, clockReceivedAt := adapter.clockTimestamp, adapter.clockReceivedAt
	destination := adapter.lastPeer
	if destination == nil {
		destination = adapter.remote
	}
	adapter.mu.Unlock()

	dataRate := packet.DataRate
	if dataRate == "" {
		dataRate = "SF7BW125"
	}
	frequency := float64(packet.Frequency) / 1_000_000
	if frequency <= 0 {
		frequency = 868.1
	}
	txpk := txPacket{
		Immediate:      packet.TransmitAt.IsZero() && packet.ConcentratorTimestamp == nil,
		InvertPolarity: packet.Protocol != contracts.RawLoRa && packet.Kind != types.GatewayPacketClassBBeacon,
		Frequency:      frequency,
		Modulation:     "LORA",
		DataRate:       dataRate,
		CodingRate:     "4/5",
		Power:          packet.Power,
		Size:           len(packet.Payload),
		Data:           base64.StdEncoding.EncodeToString(packet.Payload),
	}
	if packet.ConcentratorTimestamp != nil {
		timestamp := *packet.ConcentratorTimestamp
		txpk.Timestamp = &timestamp
	} else if !packet.TransmitAt.IsZero() {
		if adapter.virtual {
			txpk.Time = packet.TransmitAt.UTC().Format(time.RFC3339Nano)
		} else {
			if clockReceivedAt.IsZero() {
				return errors.New("cannot schedule gateway downlink without a concentrator timestamp")
			}
			delta := packet.TransmitAt.Sub(clockReceivedAt).Microseconds()
			if time.Since(clockReceivedAt) > time.Duration(1<<31-1)*time.Microsecond || delta < -1<<31 || delta > 1<<31-1 {
				return errors.New("concentrator timestamp reference is too old or transmission time is out of range")
			}
			timestamp := clockTimestamp + uint32(delta)
			txpk.Timestamp = &timestamp
		}
	}
	body, err := json.Marshal(struct {
		TXPK txPacket `json:"txpk"`
	}{TXPK: txpk})
	if err != nil {
		return fmt.Errorf("encode gateway downlink: %w", err)
	}
	token, err := randomToken()
	if err != nil {
		return fmt.Errorf("create gateway packet token: %w", err)
	}
	frame := append(packetHeader(token, pullResponseType), body...)
	tokenKey := hex.EncodeToString(token[:])
	adapter.mu.Lock()
	adapter.pending[tokenKey] = types.GatewayPacket{
		GatewayID: packet.GatewayID, DeviceID: packet.DeviceID, Kind: packet.Kind, Protocol: packet.Protocol,
		Payload: append([]byte(nil), packet.Payload...), Frequency: packet.Frequency, Bandwidth: packet.Bandwidth,
		SpreadingFactor: packet.SpreadingFactor, Power: packet.Power, DataRate: packet.DataRate, TransmitAt: packet.TransmitAt,
		ConcentratorTimestamp: txpk.Timestamp,
	}
	adapter.mu.Unlock()
	if err := adapter.send(frame, destination); err != nil {
		adapter.mu.Lock()
		delete(adapter.pending, tokenKey)
		adapter.mu.Unlock()
		return err
	}
	return nil
}

func (adapter *udpAdapter) SendUplink(ctx context.Context, packet types.GatewayPacket) error {
	if ctx == nil {
		return errors.New("gateway uplink context cannot be nil")
	}
	if len(packet.Payload) == 0 {
		return errors.New("gateway uplink payload cannot be empty")
	}
	dataRate := packet.DataRate
	if dataRate == "" {
		dataRate = dataRateFromTransmission(packet.SpreadingFactor, packet.Bandwidth)
	}
	if dataRate == "" {
		dataRate = "SF7BW125"
	}
	frequency := float64(packet.Frequency) / 1_000_000
	if frequency <= 0 {
		frequency = 868.1
	}
	now := time.Now()
	timestamp := uint32(now.UnixMicro())
	uplink := rxPacket{
		Data: base64.StdEncoding.EncodeToString(packet.Payload), Frequency: frequency,
		Timestamp: &timestamp, Channel: 0, RFChannel: 0, Status: 1,
		DataRate: dataRate, Modulation: "LORA", CodingRate: "4/5",
		RSSI: int(math.Round(packet.RSSI)), SNR: packet.SNR, Size: len(packet.Payload),
	}
	if adapter.virtual {
		gps := gpsMilliseconds(now)
		uplink.GPSMilliseconds = &gps
	}
	return adapter.sendPushData(ctx, pushDataPayload{RXPK: []rxPacket{uplink}})
}

func (adapter *udpAdapter) Packets() <-chan types.GatewayPacket { return adapter.packets }

func (adapter *udpAdapter) Events() <-chan types.GatewayAdapterEvent { return adapter.events }

func (adapter *udpAdapter) Close() error {
	adapter.mu.Lock()
	if adapter.closed {
		started := adapter.started
		adapter.mu.Unlock()
		if started {
			<-adapter.done
		}
		return nil
	}
	adapter.closed = true
	cancel := adapter.cancel
	started := adapter.started
	adapter.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if adapter.virtual {
		adapter.mu.Lock()
		conn := adapter.conn
		adapter.conn = nil
		adapter.mu.Unlock()
		if conn != nil {
			_ = conn.Close()
		}
	} else {
		adapter.factory.unregister(adapter)
	}
	if started {
		<-adapter.done
		return nil
	}
	close(adapter.done)
	close(adapter.packets)
	close(adapter.events)
	return nil
}

func (adapter *udpAdapter) run(ctx context.Context) {
	defer adapter.gpsDownlinks.close()
	defer func() {
		if !adapter.virtual {
			adapter.factory.unregister(adapter)
		}
		close(adapter.done)
		close(adapter.packets)
		close(adapter.events)
	}()

	lastActivity := time.Now()
	lastKeepAlive := time.Time{}
	connected := false
	backoff := adapter.options.ReconnectInitial
	reconnectAt := time.Time{}
	tickInterval := 250 * time.Millisecond
	if heartbeatTick := adapter.options.HeartbeatInterval / 2; heartbeatTick > 0 && heartbeatTick < tickInterval {
		tickInterval = heartbeatTick
	}
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	var statisticsTicks <-chan time.Time
	if adapter.virtual {
		statisticsTicker := time.NewTicker(adapter.statisticsInterval)
		defer statisticsTicker.Stop()
		statisticsTicks = statisticsTicker.C
		if err := adapter.sendKeepAlive(ctx); err != nil {
			_ = adapter.emitError(ctx, err, false)
		}
		lastKeepAlive = time.Now()
		if err := adapter.sendStatistics(ctx, lastKeepAlive); err != nil && !adapter.emitError(ctx, err, false) {
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case datagram, ok := <-adapter.inbound:
			if !ok {
				return
			}
			if adapter.virtual && !adapter.acceptBridgeDatagram(datagram) {
				continue
			}
			adapter.mu.Lock()
			adapter.lastPeer = datagram.remote
			adapter.mu.Unlock()
			lastActivity = time.Now()
			if !connected {
				if !adapter.emitState(ctx, contracts.Connected) {
					return
				}
				connected = true
				backoff = adapter.options.ReconnectInitial
				reconnectAt = time.Time{}
			}
			if err := adapter.handleDatagram(ctx, datagram); err != nil && !adapter.emitError(ctx, err, false) {
				return
			}
		case now := <-statisticsTicks:
			if err := adapter.sendStatistics(ctx, now); err != nil && !adapter.emitError(ctx, err, false) {
				return
			}
		case <-adapter.gpsDownlinks.deadline():
			if err := adapter.deliverDownlink(ctx, adapter.gpsDownlinks.pop()); err != nil {
				return
			}
		case err := <-adapter.failures:
			if err == nil {
				continue
			}
			if !adapter.emitPendingFailures(ctx, err, false) {
				return
			}
			if !adapter.emitError(ctx, err, false) || !adapter.emitState(ctx, contracts.Disconnected) || !adapter.emitState(ctx, contracts.Reconnecting) {
				return
			}
			connected = false
			reconnectAt = time.Now().Add(backoff)
			backoff = minDuration(backoff*2, adapter.options.ReconnectMax)
		case <-ticker.C:
			now := time.Now()
			if adapter.virtual && (lastKeepAlive.IsZero() || now.Sub(lastKeepAlive) >= adapter.options.HeartbeatInterval) {
				if err := adapter.sendKeepAlive(ctx); err != nil && !adapter.emitError(ctx, err, false) {
					return
				}
				lastKeepAlive = now
			}
			if now.Sub(lastActivity) >= adapter.options.Timeout {
				if !adapter.emitPendingFailures(ctx, ErrGatewayHeartbeat, true) {
					return
				}
				if !adapter.emitError(ctx, ErrGatewayHeartbeat, true) || !adapter.emitState(ctx, contracts.Disconnected) || !adapter.emitState(ctx, contracts.Reconnecting) {
					return
				}
				connected = false
				lastActivity = now
				reconnectAt = now.Add(backoff)
				backoff = minDuration(backoff*2, adapter.options.ReconnectMax)
			} else if !adapter.virtual && !connected && !reconnectAt.IsZero() && !now.Before(reconnectAt) {
				if ctx.Err() != nil {
					return
				}
				if err := adapter.factory.reconnect(adapter); err != nil && !adapter.emitError(ctx, err, false) {
					return
				}
				reconnectAt = now.Add(backoff)
				backoff = minDuration(backoff*2, adapter.options.ReconnectMax)
			}
		}
	}
}

func (adapter *udpAdapter) readVirtualLoop(ctx context.Context) {
	buffer := make([]byte, 64*1024)
	for {
		adapter.mu.Lock()
		conn := adapter.conn
		adapter.mu.Unlock()
		if conn == nil {
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		size, remote, err := conn.ReadFromUDP(buffer)
		if err != nil {
			var netError net.Error
			if errors.As(err, &netError) && netError.Timeout() {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			if ctx.Err() == nil {
				select {
				case adapter.failures <- err:
				default:
				}
			}
			return
		}
		select {
		case adapter.inbound <- udpDatagram{data: append([]byte(nil), buffer[:size]...), remote: remote}:
		case <-ctx.Done():
			return
		}
	}
}

func (adapter *udpAdapter) sendKeepAlive(ctx context.Context) error {
	if ctx == nil {
		return errors.New("gateway keepalive context cannot be nil")
	}
	token, err := randomToken()
	if err != nil {
		return fmt.Errorf("create gateway keepalive token: %w", err)
	}
	eui, err := hex.DecodeString(adapter.gatewayEUI)
	if err != nil {
		return fmt.Errorf("decode gateway EUI: %w", err)
	}
	frame := append(packetHeader(token, pullDataType), eui...)
	adapter.mu.Lock()
	adapter.pullToken = token
	adapter.pullPending = true
	adapter.mu.Unlock()
	if err := adapter.send(frame, adapter.remote); err != nil {
		adapter.mu.Lock()
		adapter.pullPending = false
		adapter.mu.Unlock()
		return err
	}
	return nil
}

func (adapter *udpAdapter) send(data []byte, destination *net.UDPAddr) error {
	if destination == nil {
		return ErrGatewayNotConnected
	}
	if adapter.virtual {
		adapter.mu.Lock()
		conn := adapter.conn
		adapter.mu.Unlock()
		if conn == nil {
			return ErrGatewayNotConnected
		}
		if _, err := conn.WriteToUDP(data, destination); err != nil {
			return fmt.Errorf("send virtual gateway UDP packet: %w", err)
		}
		return nil
	}
	return adapter.factory.send(adapter, data, destination)
}

func (adapter *udpAdapter) handleDatagram(ctx context.Context, datagram udpDatagram) error {
	data := datagram.data
	if len(data) < 4 || data[0] != protocolVersion {
		return errors.New("invalid Semtech UDP packet header")
	}
	switch data[3] {
	case pullDataType:
		return adapter.emitHeartbeat(ctx)
	case pushDataType:
		if len(data) < 12 {
			return errors.New("invalid PUSH_DATA packet")
		}
		if err := adapter.validateGatewayEUI(data[4:12]); err != nil {
			return err
		}
		return adapter.handlePushData(ctx, data[12:])
	case pushAckType, pullAckType:
		// PUSH_ACK and PULL_ACK are the acknowledgements for the two
		// gateway-to-server keepalive/data messages. They carry no payload;
		// receiving either one proves that the UDP path is usable.
		return adapter.emitHeartbeat(ctx)
	case pullResponseType:
		if adapter.virtual {
			adapter.mu.Lock()
			adapter.statistics.downlinks++
			adapter.mu.Unlock()
		}
		packet, gpsTimed, err := adapter.decodePullResponse(data[4:])
		ackError := "NONE"
		switch {
		case errors.Is(err, errGatewayGPSUnlocked):
			ackError = "GPS_UNLOCKED"
		case err != nil:
			ackError = "INVALID_DATA"
		case gpsTimed:
			ackError, err = adapter.gpsDownlinkStatus(*packet)
		}
		if ackErr := adapter.sendTXAck(data[1:3], datagram.remote, ackError); ackErr != nil {
			if err == nil {
				return ackErr
			}
			return fmt.Errorf("decode downlink: %v; send TX_ACK: %w", err, ackErr)
		}
		if err != nil {
			return err
		}
		if gpsTimed {
			adapter.gpsDownlinks.add(*packet)
			return nil
		}
		return adapter.deliverDownlink(ctx, *packet)
	case txAckType:
		if len(data) < 12 {
			return errors.New("invalid TX_ACK packet")
		}
		if err := adapter.validateGatewayEUI(data[4:12]); err != nil {
			return err
		}
		// The Semtech protocol reserves bytes 4..11 for the gateway EUI.
		// The JSON acknowledgement is optional and starts at byte 12.
		packet, err := adapter.handleTXAck(data[1:3], data[12:])
		if err != nil && !adapter.emitErrorForPacket(ctx, err, false, packet) {
			return ctx.Err()
		}
		return nil
	default:
		return fmt.Errorf("unsupported Semtech UDP packet type 0x%02x", data[3])
	}
}

func (adapter *udpAdapter) decodePullResponse(body []byte) (*types.GatewayPacket, bool, error) {
	var payload struct {
		TXPK txPacket `json:"txpk"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false, fmt.Errorf("decode PULL_RESP payload: %w", err)
	}
	gpsTimed := payload.TXPK.GPSMilliseconds != nil && !payload.TXPK.Immediate && payload.TXPK.Timestamp == nil && payload.TXPK.Time == ""
	if gpsTimed && !adapter.virtual {
		return nil, false, errGatewayGPSUnlocked
	}
	if payload.TXPK.Modulation != "" && !strings.EqualFold(payload.TXPK.Modulation, "LORA") {
		return nil, false, fmt.Errorf("unsupported downlink modulation %q", payload.TXPK.Modulation)
	}
	if payload.TXPK.Data == "" {
		return nil, false, errors.New("downlink payload is empty")
	}
	decoded, err := base64.StdEncoding.DecodeString(payload.TXPK.Data)
	if err != nil {
		return nil, false, fmt.Errorf("decode downlink payload: %w", err)
	}
	if payload.TXPK.Size > 0 && payload.TXPK.Size != len(decoded) {
		return nil, false, fmt.Errorf("downlink payload size mismatch: declared %d, decoded %d", payload.TXPK.Size, len(decoded))
	}
	receivedAt := time.Now()
	var transmitAt time.Time
	if gpsTimed {
		var valid bool
		transmitAt, valid = timeFromGPSMilliseconds(*payload.TXPK.GPSMilliseconds)
		if !valid {
			return nil, false, errors.New("invalid GPS transmission time")
		}
	}
	if adapter.virtual && payload.TXPK.Timestamp != nil && !payload.TXPK.Immediate {
		nowTimestamp := uint32(receivedAt.UnixMicro())
		delta := int32(*payload.TXPK.Timestamp - nowTimestamp)
		transmitAt = receivedAt.Add(time.Duration(delta) * time.Microsecond)
	}
	if payload.TXPK.Time != "" {
		transmitAt, err = time.Parse(time.RFC3339Nano, payload.TXPK.Time)
		if err != nil {
			return nil, false, fmt.Errorf("parse downlink transmission time: %w", err)
		}
	}
	return &types.GatewayPacket{
		GatewayID:       adapter.gatewayID,
		Kind:            types.GatewayPacketDownlink,
		Payload:         decoded,
		Frequency:       int64(math.Round(payload.TXPK.Frequency * 1_000_000)),
		Bandwidth:       parseBandwidth(payload.TXPK.DataRate),
		SpreadingFactor: parseSpreadingFactor(payload.TXPK.DataRate),
		Power:           payload.TXPK.Power,
		DataRate:        payload.TXPK.DataRate,
		TransmitAt:      transmitAt,
		ReceivedAt:      receivedAt,
	}, gpsTimed, nil
}

func (adapter *udpAdapter) sendTXAck(token []byte, destination *net.UDPAddr, errorCode string) error {
	if len(token) != 2 {
		return errors.New("invalid PULL_RESP token")
	}
	eui, err := hex.DecodeString(adapter.gatewayEUI)
	if err != nil {
		return fmt.Errorf("decode gateway EUI: %w", err)
	}
	body := txAckPayload{}
	body.TXPKAck.Error = errorCode
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode TX_ACK payload: %w", err)
	}
	frame := append(packetHeader([2]byte{token[0], token[1]}, txAckType), eui...)
	frame = append(frame, encoded...)
	return adapter.send(frame, destination)
}

func (adapter *udpAdapter) handlePushData(ctx context.Context, body []byte) error {
	var payload struct {
		RXPK []json.RawMessage `json:"rxpk"`
	}
	if len(body) == 0 {
		return adapter.emitHeartbeat(ctx)
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode PUSH_DATA payload: %w", err)
	}
	var packetErrors []error
	for index, rawPacket := range payload.RXPK {
		if err := adapter.handleRXPacket(ctx, rawPacket); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			packetErrors = append(packetErrors, fmt.Errorf("rxpk[%d]: %w", index, err))
		}
	}
	return errors.Join(append(packetErrors, adapter.emitHeartbeat(ctx))...)
}

func (adapter *udpAdapter) handleRXPacket(ctx context.Context, rawPacket json.RawMessage) error {
	var received *rxPacket
	if err := json.Unmarshal(rawPacket, &received); err != nil {
		return fmt.Errorf("decode gateway RXPK: %w", err)
	}
	if received == nil {
		return errors.New("gateway RXPK is null")
	}
	if received.Status == -1 {
		return nil
	}
	if received.Modulation != "" && !strings.EqualFold(received.Modulation, "LORA") {
		return fmt.Errorf("unsupported gateway modulation %q", received.Modulation)
	}
	decoded, err := base64.StdEncoding.DecodeString(received.Data)
	if err != nil {
		return fmt.Errorf("decode gateway payload: %w", err)
	}
	if received.Size > 0 && received.Size != len(decoded) {
		return fmt.Errorf("gateway payload size mismatch: declared %d, decoded %d", received.Size, len(decoded))
	}
	receivedAt := time.Now()
	if received.Timestamp != nil {
		adapter.mu.Lock()
		adapter.clockTimestamp = *received.Timestamp
		adapter.clockReceivedAt = receivedAt
		adapter.mu.Unlock()
	}
	packet := types.GatewayPacket{
		ConcentratorTimestamp: received.Timestamp,
		GatewayID:             adapter.gatewayID,
		Payload:               decoded,
		Frequency:             int64(math.Round(received.Frequency * 1_000_000)),
		Bandwidth:             parseBandwidth(received.DataRate),
		SpreadingFactor:       parseSpreadingFactor(received.DataRate),
		RSSI:                  float64(received.RSSI),
		SNR:                   received.SNR,
		DataRate:              received.DataRate,
		ReceivedAt:            receivedAt,
	}
	select {
	case adapter.packets <- packet:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func parseSpreadingFactor(dataRate string) int {
	if !strings.HasPrefix(dataRate, "SF") {
		return 0
	}
	value, err := strconv.Atoi(strings.TrimPrefix(strings.Split(dataRate, "BW")[0], "SF"))
	if err != nil || value < 7 || value > 12 {
		return 0
	}
	return value
}

func parseBandwidth(dataRate string) int64 {
	parts := strings.Split(dataRate, "BW")
	if len(parts) != 2 {
		return 0
	}
	value, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || value <= 0 {
		return 0
	}
	return value * 1000
}

func dataRateFromTransmission(spreadingFactor int, bandwidth int64) string {
	if spreadingFactor < 7 || spreadingFactor > 12 || bandwidth <= 0 {
		return ""
	}
	return fmt.Sprintf("SF%dBW%d", spreadingFactor, bandwidth/1000)
}

func (adapter *udpAdapter) handleTXAck(token, body []byte) (*types.GatewayPacket, error) {
	tokenKey := hex.EncodeToString(token)
	adapter.mu.Lock()
	pending, hasPending := adapter.pending[tokenKey]
	delete(adapter.pending, tokenKey)
	adapter.mu.Unlock()
	if len(body) == 0 {
		return nil, nil
	}
	var ack txAckPayload
	if err := json.Unmarshal(body, &ack); err != nil {
		if hasPending {
			return &pending, fmt.Errorf("decode TX_ACK payload: %w", err)
		}
		return nil, fmt.Errorf("decode TX_ACK payload: %w", err)
	}
	if ack.TXPKAck.Error != "" && ack.TXPKAck.Error != "NONE" {
		if hasPending {
			return &pending, fmt.Errorf("gateway TX_ACK: %s", ack.TXPKAck.Error)
		}
		return nil, fmt.Errorf("gateway TX_ACK: %s", ack.TXPKAck.Error)
	}
	return nil, nil
}

func (adapter *udpAdapter) validateGatewayEUI(value []byte) error {
	expected, err := hex.DecodeString(adapter.gatewayEUI)
	if err != nil || len(expected) != len(value) || !equalBytes(expected, value) {
		return errors.New("gateway EUI in PUSH_DATA does not match configured gateway")
	}
	return nil
}

func (adapter *udpAdapter) emitState(ctx context.Context, state contracts.GatewayConnectionState) bool {
	return adapter.emitEvent(ctx, types.GatewayAdapterEvent{GatewayID: adapter.gatewayID, State: state, At: time.Now()})
}

func (adapter *udpAdapter) emitHeartbeat(ctx context.Context) error {
	if !adapter.emitEvent(ctx, types.GatewayAdapterEvent{GatewayID: adapter.gatewayID, State: contracts.Connected, Heartbeat: true, At: time.Now()}) {
		return ctx.Err()
	}
	return nil
}

func (adapter *udpAdapter) emitError(ctx context.Context, err error, timeout bool) bool {
	return adapter.emitErrorForPacket(ctx, err, timeout, nil)
}

// emitPendingFailures hands packets without a TX_ACK back to the engine. The
// engine keeps them queued and dispatches them after the adapter reconnects.
// Clearing the transport-side map before emitting the events makes this
// operation idempotent when a socket failure and a heartbeat timeout overlap.
func (adapter *udpAdapter) emitPendingFailures(ctx context.Context, err error, timeout bool) bool {
	adapter.mu.Lock()
	packets := make([]types.GatewayPacket, 0, len(adapter.pending))
	for _, packet := range adapter.pending {
		packet.Payload = append([]byte(nil), packet.Payload...)
		packets = append(packets, packet)
	}
	adapter.pending = make(map[string]types.GatewayPacket)
	adapter.mu.Unlock()

	for index := range packets {
		if !adapter.emitErrorForPacket(ctx, err, timeout, &packets[index]) {
			return false
		}
	}
	return true
}

func (adapter *udpAdapter) emitErrorForPacket(ctx context.Context, err error, timeout bool, packet *types.GatewayPacket) bool {
	return adapter.emitEvent(ctx, types.GatewayAdapterEvent{GatewayID: adapter.gatewayID, State: contracts.Error, Error: err.Error(), Timeout: timeout, At: time.Now(), Packet: packet})
}

func (adapter *udpAdapter) emitEvent(ctx context.Context, event types.GatewayAdapterEvent) bool {
	select {
	case adapter.events <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

type pushDataPayload struct {
	RXPK []rxPacket         `json:"rxpk,omitempty"`
	Stat *gatewayStatistics `json:"stat,omitempty"`
}

type rxPacket struct {
	GPSMilliseconds *uint64 `json:"tmms,omitempty"`
	Data            string  `json:"data"`
	Frequency       float64 `json:"freq"`
	Timestamp       *uint32 `json:"tmst,omitempty"`
	Channel         int     `json:"chan"`
	RFChannel       int     `json:"rfch"`
	Status          int     `json:"stat"`
	DataRate        string  `json:"datr"`
	Modulation      string  `json:"modu,omitempty"`
	CodingRate      string  `json:"codr,omitempty"`
	RSSI            int     `json:"rssi"`
	SNR             float64 `json:"lsnr"`
	Size            int     `json:"size,omitempty"`
}

type txPacket struct {
	InvertPolarity  bool    `json:"ipol"`
	Immediate       bool    `json:"imme"`
	Timestamp       *uint32 `json:"tmst,omitempty"`
	GPSMilliseconds *uint64 `json:"tmms,omitempty"`
	Time            string  `json:"time,omitempty"`
	Frequency       float64 `json:"freq"`
	Modulation      string  `json:"modu"`
	DataRate        string  `json:"datr"`
	CodingRate      string  `json:"codr"`
	Power           int     `json:"powe,omitempty"`
	Size            int     `json:"size"`
	Data            string  `json:"data"`
}

type txAckPayload struct {
	TXPKAck struct {
		Error string `json:"error"`
	} `json:"txpk_ack"`
}

func packetHeader(token [2]byte, packetType byte) []byte {
	return []byte{protocolVersion, token[0], token[1], packetType}
}

func randomToken() ([2]byte, error) {
	var token [2]byte
	_, err := rand.Read(token[:])
	return token, err
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}

func equalBytes(left, right []byte) bool {
	return len(left) == len(right) && strings.EqualFold(hex.EncodeToString(left), hex.EncodeToString(right))
}

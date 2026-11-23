package types

import (
	"context"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type EventSink func(contracts.SimulationEvent)

type GatewayPacket struct {
	GatewayID       string
	DeviceID        string
	Kind            GatewayPacketKind
	Protocol        contracts.RadioProtocol
	Payload         []byte
	Frequency       int64
	Bandwidth       int64
	SpreadingFactor int
	RSSI            float64
	SNR             float64
	Power           int
	DataRate        string
	// ConcentratorTimestamp is the Semtech 32-bit microsecond counter at RX end or scheduled TX.
	ConcentratorTimestamp *uint32
	// TransmitAt requests a timed downlink. A zero value keeps the existing
	// immediate-transmission behavior used by ordinary gateway packets.
	TransmitAt time.Time
	ReceivedAt time.Time
}

type GatewayPacketKind string

const (
	GatewayPacketGeneric        GatewayPacketKind = "generic"
	GatewayPacketUplink         GatewayPacketKind = "uplink"
	GatewayPacketClassBBeacon   GatewayPacketKind = "class-b-beacon"
	GatewayPacketClassBDownlink GatewayPacketKind = "class-b-downlink"
	GatewayPacketDownlink       GatewayPacketKind = "downlink"
)

type GatewayAdapterEvent struct {
	GatewayID string
	State     contracts.GatewayConnectionState
	Error     string
	Timeout   bool
	Heartbeat bool
	At        time.Time
	Packet    *GatewayPacket
}

type GatewayAdapter interface {
	Start(context.Context) error
	Send(context.Context, GatewayPacket) error
	Packets() <-chan GatewayPacket
	Events() <-chan GatewayAdapterEvent
	Close() error
}

// GatewayUplinkAdapter is implemented by transports that can expose the
// simulator as a gateway to an external network server. Downlinks continue to
// use GatewayAdapter.Send; uplinks use this separate method so the transport
// can encode the Semtech PUSH_DATA envelope correctly.
type GatewayUplinkAdapter interface {
	SendUplink(context.Context, GatewayPacket) error
}

// VirtualGatewayAdapterFactory identifies factories that can transport
// in-process virtual gateways to an external Gateway Bridge.
type VirtualGatewayAdapterFactory interface {
	SupportsVirtualGateways() bool
}

type GatewayAdapterFactory interface {
	NewGatewayAdapter(contracts.Gateway) (GatewayAdapter, error)
}

type GatewayAdapterFactoryCloser interface {
	Close() error
}

// GatewayBridgeConfigurator allows the backend to apply the bridge settings
// selected for a simulation before the engine starts its gateway adapters.
// Implementations must not start or stop transports while configuring them.
type GatewayBridgeConfigurator interface {
	ConfigureGatewayBridge(contracts.GatewayBridgeConfig) error
}

type GatewayRuntime struct {
	State              contracts.GatewayConnectionState
	LastHeartbeat      time.Duration
	ConnectionAttempts int64
	LastNetworkError   string
	IngressPackets     int64
	EgressPackets      int64
}

type Options struct {
	Clock                 Clock
	EventSink             EventSink
	EventBuffer           int
	GatewayAdapterFactory GatewayAdapterFactory
	Checkpoint            *EngineCheckpoint
}

// EngineCheckpoint is the durable simulation state. It intentionally keeps
// the contract snapshot and the engine runtime state together so a backend
// restart can continue from the same simulation timeline.
type EngineCheckpoint struct {
	RunID          string                       `json:",omitzero"`
	Config         contracts.SimulationConfig   `json:",omitzero"`
	Devices        []contracts.Device           `json:",omitzero"`
	Gateways       []contracts.Gateway          `json:",omitzero"`
	State          contracts.SimulationState    `json:",omitzero"`
	Metrics        contracts.SimulationMetrics  `json:",omitzero"`
	Sessions       map[string]DeviceSession     `json:",omitzero"`
	Radio          map[string]RadioTransmission `json:",omitzero"`
	Scheduled      []ScheduledEvent             `json:",omitzero"`
	GatewayPackets []GatewayPacket              `json:",omitzero"`
	EventLog       []contracts.SimulationEvent  `json:",omitzero"`
}

type ScheduledEvent struct {
	ID               string                        `json:",omitzero"`
	At               time.Duration                 `json:",omitzero"`
	Type             contracts.SimulationEventType `json:",omitzero"`
	Message          string                        `json:",omitzero"`
	DeviceID         string                        `json:",omitzero"`
	GatewayID        string                        `json:",omitzero"`
	Kind             ScheduledEventKind            `json:",omitzero"`
	Attempt          int                           `json:",omitzero"`
	PacketID         string                        `json:",omitzero"`
	FrameCounter     int64                         `json:",omitzero"`
	Confirmed        bool                          `json:",omitzero"`
	Window           contracts.SimulationRxWindow  `json:",omitzero"`
	WindowDuration   time.Duration                 `json:",omitzero"`
	DataRate         int                           `json:",omitzero"`
	ChannelFrequency int64                         `json:",omitzero"`
	WindowBaseAt     time.Duration                 `json:",omitzero"`
	FragmentIndex    int                           `json:",omitzero"`
	FragmentCount    int                           `json:",omitzero"`
	FPendingPoll     bool                          `json:",omitzero"`
}

type ScheduledEventKind string

const (
	ScheduledEventGeneric             ScheduledEventKind = "generic"
	ScheduledEventDeviceUplink        ScheduledEventKind = "device-uplink"
	ScheduledEventJoinRequest         ScheduledEventKind = "join-request"
	ScheduledEventRX1Window           ScheduledEventKind = "rx1-window"
	ScheduledEventRX2Window           ScheduledEventKind = "rx2-window"
	ScheduledEventRadioComplete       ScheduledEventKind = "radio-transmission-complete"
	ScheduledEventGatewayHeartbeat    ScheduledEventKind = "virtual-gateway-heartbeat"
	ScheduledEventGatewayBeacon       ScheduledEventKind = "gateway-beacon"
	ScheduledEventClassBBeaconTimeout ScheduledEventKind = "class-b-beacon-timeout"
	ScheduledEventClassBPingSlot      ScheduledEventKind = "class-b-ping-slot"
	ScheduledEventClassCDownlink      ScheduledEventKind = "class-c-downlink"
)

// Class B follows the LoRaWAN beacon cadence. The one-second ping-slot period
// is the default periodicity used by the simulator when a device does not
// expose a network-specific ping-slot periodicity.
const (
	ClassBBeaconPeriod   = 128 * time.Second
	ClassBPingSlotPeriod = time.Second
)

// NetworkChannel is an indexed channel installed by a LoRaWAN network server.
type NetworkChannel struct {
	// Frequency is the uplink frequency in hertz; zero removes the channel.
	Frequency int64
	// MinimumDataRate and MaximumDataRate bound the permitted uplink data rates.
	MinimumDataRate, MaximumDataRate int
	// DownlinkFrequency overrides RX1 when a DLChannelReq is accepted.
	DownlinkFrequency int64
}

type DeviceSession struct {
	// PendingMACResponses contains complete uplink MAC responses waiting for transmission.
	PendingMACResponses [][]byte `json:",omitzero"`
	// NetworkChannels overrides the regional channel table by channel index.
	NetworkChannels map[int]NetworkChannel `json:",omitzero"`
	// DisabledChannels records channels disabled by a network channel mask.
	DisabledChannels map[int]bool `json:",omitzero"`
	// PendingDownlinkACK acknowledges a confirmed downlink on the next new uplink.
	PendingDownlinkACK bool `json:",omitzero"`
	// ExternalADR leaves ADR decisions to the connected network server.
	ExternalADR                      bool                `json:",omitzero"`
	Joined                           bool                `json:",omitzero"`
	FrameCounterUp                   int64               `json:",omitzero"`
	LastExternalFrameCounter         uint32              `json:",omitzero"`
	FrameCounterDown                 int64               `json:",omitzero"`
	LastExternalDownlinkFrameCounter uint32              `json:",omitzero"`
	ExternalDownlinkFrameCounterSet  bool                `json:",omitzero"`
	CurrentDataRate                  int                 `json:",omitzero"`
	CurrentSpreadingFactor           int                 `json:",omitzero"`
	DeviceAddress                    string              `json:",omitzero"`
	JoinEUI                          string              `json:",omitzero"`
	SecurityFingerprint              string              `json:",omitzero"`
	PendingUplink                    *PendingUplink      `json:",omitzero"`
	PendingJoinRequest               *PendingJoinRequest `json:",omitzero"`
	LastRSSI                         float64             `json:",omitzero"`
	LastSNR                          float64             `json:",omitzero"`
	LastAirtime                      time.Duration       `json:",omitzero"`
	LastChannel                      int64               `json:",omitzero"`
	LastPayloadSize                  int                 `json:",omitzero"`
	LastFPort                        int                 `json:",omitzero"`
	ClassBSynchronized               bool                `json:",omitzero"`
	LastBeaconAt                     time.Duration       `json:",omitzero"`
	NextPingSlotAt                   time.Duration       `json:",omitzero"`
	ClassBMissedBeacons              int64               `json:",omitzero"`
	ClassBGatewayID                  string              `json:",omitzero"`
	ClassBNextPingEventID            string              `json:",omitzero"`
	ClassBBeaconTimeoutID            string              `json:",omitzero"`
	PendingClassBDownlinks           []ClassBDownlink    `json:",omitzero"`
	PendingClassADownlinks           []Downlink          `json:",omitzero"`
	PendingClassCDownlinks           []Downlink          `json:",omitzero"`
	ClassCNextDownlinkEventID        string              `json:",omitzero"`
	NwkSKey                          []byte              `json:",omitzero"`
	AppSKey                          []byte              `json:",omitzero"`
	DevNonce                         uint16              `json:",omitzero"`
	CurrentTxPower                   int                 `json:",omitzero"`
	UnconfirmedRepetitions           int                 `json:",omitzero"`
	MaximumDutyCycle                 float64             `json:",omitzero"`
	RX1DataRateOffset                int                 `json:",omitzero"`
	RX2DataRate                      int                 `json:",omitzero"`
	RX2Frequency                     int64               `json:",omitzero"`
	ReceiveDelay                     time.Duration       `json:",omitzero"`
	UplinkDwellTime                  bool                `json:",omitzero"`
	DownlinkDwellTime                bool                `json:",omitzero"`
	MaximumEIRP                      int                 `json:",omitzero"`
	PingSlotDataRate                 int                 `json:",omitzero"`
	PingSlotFrequency                int64               `json:",omitzero"`
	PingSlotPeriodicity              int                 `json:",omitzero"`
	BeaconFrequency                  int64               `json:",omitzero"`
	AdditionalChannels               []RadioChannel      `json:",omitzero"`
}

// Downlink is queued for delivery according to the device class. An empty ID
// is replaced by the engine with a generated identifier.
type Downlink struct {
	ID          string
	DeviceID    string
	Payload     []byte
	FPort       int
	DataRate    int
	Confirmed   bool
	FPending    bool
	ACK         bool
	MACCommands []MACCommand
}

type MACCommandType string

const (
	MACLinkCheckAns       MACCommandType = "link-check-ans"
	MACLinkADRReq         MACCommandType = "link-adr-req"
	MACDutyCycleReq       MACCommandType = "duty-cycle-req"
	MACRXParamSetupReq    MACCommandType = "rx-param-setup-req"
	MACDevStatusReq       MACCommandType = "dev-status-req"
	MACNewChannelReq      MACCommandType = "new-channel-req"
	MACRXTimingSetupReq   MACCommandType = "rx-timing-setup-req"
	MACTXParamSetupReq    MACCommandType = "tx-param-setup-req"
	MACDLChannelReq       MACCommandType = "dl-channel-req"
	MACDeviceTimeAns      MACCommandType = "device-time-ans"
	MACPingSlotInfoAns    MACCommandType = "ping-slot-info-ans"
	MACPingSlotChannelReq MACCommandType = "ping-slot-channel-req"
	MACBeaconFreqReq      MACCommandType = "beacon-freq-req"
)

// MACCommand contains the semantic values of a LoRaWAN 1.0.x MAC command.
// Only fields used by the selected command are considered. Pointer fields
// distinguish an explicit zero from an omitted value.
type MACCommand struct {
	Type                 MACCommandType
	DataRate             *int
	TxPower              *int
	NbTrans              *int
	ChannelMask          *uint16
	ChannelMaskControl   *int
	MaxDutyCycleExponent *int
	RX1DataRateOffset    *int
	Frequency            *int64
	ChannelIndex         *int
	MinimumDataRate      *int
	MaximumDataRate      *int
	Delay                *time.Duration
	UplinkDwellTime      *bool
	DownlinkDwellTime    *bool
	MaximumEIRP          *int
	Margin               *int
	GatewayCount         *int
	BatteryLevel         *int
	PingSlotPeriodicity  *int
	DeviceTime           *time.Time
}

// ClassBDownlink is kept as a semantic alias for callers that use the
// Class-B-specific API.
type ClassBDownlink = Downlink

type PendingUplink struct {
	// WirePayload preserves the entire frame, including MAC responses, across retries.
	WirePayload []byte
	// WireFragmentIndex identifies the fragment represented by WirePayload.
	WireFragmentIndex int
	// ReservedFOpts freezes application fragment capacity while a message is in flight.
	ReservedFOpts int
	// PayloadConfig freezes the application data for retries and remaining fragments.
	PayloadConfig *contracts.PayloadConfig
	// SentAt is the wall-clock completion time of the latest radio attempt.
	SentAt         time.Time
	PacketID       string
	FrameCounter   int64
	Attempt        int
	Confirmed      bool
	GatewayIDs     []string
	TransmissionAt time.Duration
	FPort          int
	PayloadSize    int
	DataRate       int
	FragmentIndex  int
	FragmentCount  int
	AnySuccessful  bool
}

type PendingJoinRequest struct {
	// RequestedAt identifies the wall-clock attempt for timed external Join-Accepts.
	RequestedAt    time.Time
	PacketID       string
	JoinEUI        string
	GatewayIDs     []string
	TransmissionAt time.Duration
}

type RadioChannel struct {
	Frequency       int64
	Bandwidth       int64
	SpreadingFactor int
	DataRate        int
}

// DataRateProfile describes the radio modulation used by a regional data
// rate. Keeping this in the public types package lets backend and frontend
// adapters expose the same engine vocabulary without depending on runtime
// internals.
type DataRateProfile struct {
	DataRate        int
	SpreadingFactor int
	Bandwidth       int64
	MaximumPayload  int
}

type RegionalChannelGroup struct {
	InitialFrequency int64
	FrequencyStep    int64
	ChannelCount     int
	MinimumDataRate  int
	MaximumDataRate  int
}

type RegionalPlan struct {
	Region            contracts.DeviceRegion
	DefaultDataRate   int
	MinimumDataRate   int
	MaximumDataRate   int
	RX2Frequency      int64
	RX2DataRate       int
	BeaconFrequency   int64
	BeaconDataRate    int
	PingSlotFrequency int64
	PingSlotDataRate  int
	UplinkChannels    []RegionalChannelGroup
	DataRates         map[int]DataRateProfile
}

// RadioFrame contains the radio-level data shared by every modulation
// protocol. Protocol-specific encoders populate the payload and protocol,
// while the radio medium uses the modulation parameters to model airtime,
// range, and collisions.
type RadioFrame struct {
	Protocol          contracts.RadioProtocol
	Payload           []byte
	ChannelFrequency  int64
	Bandwidth         int64
	SpreadingFactor   int
	DataRate          int
	TransmissionPower float64
}

// RadioTransmission tracks a frame while it is in flight through the radio
// medium. RadioFrame is embedded so the existing runtime can access common
// radio properties without coupling them to LoRaWAN session state.
type RadioTransmission struct {
	RadioFrame
	PacketID      string
	DeviceID      string
	FrameCounter  int64
	Attempt       int
	Confirmed     bool
	StartAt       time.Duration
	EndAt         time.Duration
	FPort         int
	PayloadSize   int
	FragmentIndex int
	FragmentCount int
	Airtime       time.Duration
	RSSI          float64
	SNR           float64
	GatewayIDs    []string
	Collision     bool
	FPendingPoll  bool
}

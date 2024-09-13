package socket

import "github.com/brocaar/lorawan"

// ConsoleLog represents a log entry with a name and a message.
type ConsoleLog struct {
	Name string `json:"name"`    // Name specifies the source or identifier of the log entry.
	Msg  string `json:"message"` // Msg contains the actual log message.
}

// NewStatusDev represents the status of a device in the network, including identifiers and counters.
type NewStatusDev struct {
	DevEUI   lorawan.EUI64   `json:"devEUI"`   // DevEUI is the unique identifier of the device.
	DevAddr  lorawan.DevAddr `json:"devAddr"`  // DevAddr is the device address.
	NwkSKey  string          `json:"nwkSKey"`  // NwkSKey is the network session key.
	AppSKey  string          `json:"appSKey"`  // AppSKey is the application session key.
	FCntDown uint32          `json:"fcntDown"` // FCntDown is the downlink frame counter.
	FCnt     uint32          `json:"fcnt"`     // FCnt is the uplink frame counter.
}

// NewPayload represents a structure for handling payload changes with ID, message type, and payload data.
type NewPayload struct {
	Id      int    `json:"id"`      // Id is the unique identifier of the payload.
	MType   string `json:"mtype"`   // MType is the message type.
	Payload string `json:"payload"` // Payload is the actual payload data.
}

// NewLocation represents the geographical location of a device.
type NewLocation struct {
	Id        int     `json:"id"`        // Id is the unique identifier of the location.
	Latitude  float64 `json:"latitude"`  // Latitude is the geographical latitude.
	Longitude float64 `json:"longitude"` // Longitude is the geographical longitude.
	Altitude  int32   `json:"altitude"`  // Altitude is the height above sea level.
}

// MacCommand represents a MAC command to be sent to a device in the network.
type MacCommand struct {
	Id          int    `json:"id"`          // Id is the unique identifier of the MAC command.
	CID         string `json:"cid"`         // CID is the command identifier.
	Periodicity uint8  `json:"periodicity"` // Periodicity is the interval at which the command is sent.
}

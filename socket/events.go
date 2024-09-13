package socket

// Event is a type representing an event that is emitted to the console.
const (
	// EventLog represents the event type for console simulation logs.
	EventLog = "console-sim"
	// EventError represents an error log event that is emitted to the console.
	EventError = "console-error"
	// EventDev is a constant representing the "log-dev" event type used for device logging.
	EventDev = "log-dev"
	// EventGw is a string constant representing the "log-gw" event type, typically used for logging gateway-specific messages.
	EventGw = "log-gw"
	// EventToggleStateDevice represents an event used to toggle the state of a device.
	EventToggleStateDevice = "toggleState-dev"
	// EventToggleStateGateway is a constant representing the event for toggling the state of a gateway.
	EventToggleStateGateway = "toggleState-gw"
	// EventSaveStatus represents the event name used to indicate the saving of a status in the system.
	EventSaveStatus = "save-status"
	// EventMacCommand represents the event identifier for sending a MAC command to a device.
	EventMacCommand = "send-MACCommand"
	// EventResponseCommand is used to indicate a response to a command issued to a device or gateway.
	EventResponseCommand = "response-command"
	// EventChangePayload is an event type for signaling a payload change in the system.
	EventChangePayload = "change-payload"
	// EventSendUplink represents the event for sending an uplink message in the system.
	EventSendUplink = "send-uplink"
	// EventChangeLocation is the constant for the "change-location" event.
	EventChangeLocation = "change-location"
	// EventGetParameters is the event name used for requesting regional parameters in the simulation environment.
	EventGetParameters = "get-regional-parameters"
)

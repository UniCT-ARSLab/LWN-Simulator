package socket

import (
	socketio "github.com/googollee/go-socket.io"
)

const (
	EventLog             = "console-sim"
	EventError           = "console-error"
	EventDev             = "log-dev"
	EventGw              = "log-gw"
	EventTurnOnDevice    = "Turn-ON-dev"
	EventTurnOffDevice   = "Turn-OFF-dev"
	EventTurnOnGateway   = "Turn-ON-gw"
	EventTurnOffGateway  = "Turn-OFF-gw"
	EventSaveStatus      = "save-status"
	EventMacCommand      = "send-MACCommand"
	EventResponseCommand = "response-command"
	EventChangePayload   = "change-payload"
	EventSendUplink      = "send-uplink"
	EventChangeLocation  = "change-location"
	EventGetParameters   = "get-regional-parameters"
)

func EmitResponse(conn socketio.Conn, msg string) {
	conn.Emit(EventResponseCommand, msg)
}

package console

import (
	"log"

	socketio "github.com/googollee/go-socket.io"
)

type Console struct {
	WebSocket socketio.Conn
}

func (c *Console) PrintLog(message string) {
	log.Println(message)
}

func (c *Console) PrintSocket(eventName string, data ...interface{}) {
	if c.WebSocket != nil {
		c.WebSocket.Emit(eventName, data...)
	}
}

func (c *Console) SetupWebSocket(WebSocket *socketio.Conn) {
	c.WebSocket = *WebSocket
}

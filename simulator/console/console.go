package console

import (
	"log"

	socketio "github.com/googollee/go-socket.io"
)

// Console represents a socket connection to send messages to the web terminal
type Console struct {
	WebSocket socketio.Conn
}

// PrintLog prints a message to the command line stdout
func (c *Console) PrintLog(message string) {
	log.Println(message)
}

// PrintSocket prints a message to the web terminal via a socket connection
func (c *Console) PrintSocket(eventName string, data ...interface{}) {
	if c.WebSocket != nil {
		c.WebSocket.Emit(eventName, data...)
	}
}

// SetupWebSocket sets the socket connection for the Console
func (c *Console) SetupWebSocket(WebSocket *socketio.Conn) {
	c.WebSocket = *WebSocket
}

package resources

import (
	"sync"

	socketio "github.com/googollee/go-socket.io"
)

type Resources struct {
	ExitGroup sync.WaitGroup `json:"-"`
	WebSocket socketio.Conn  `json:"-"`
}

func (r *Resources) AddWebSocket(WebSocket *socketio.Conn) {
	r.WebSocket = *WebSocket
}

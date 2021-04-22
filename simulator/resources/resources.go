package resources

import (
	"sync"

	socketio "github.com/googollee/go-socket.io"
)

type Resources struct {
	Mutex     sync.Mutex     `json:"-"`
	ExitGroup sync.WaitGroup `json:"-"`
	WebSocket socketio.Conn  `json:"-"`
}

func NewResource(WebSocket socketio.Conn) *Resources {
	var r Resources

	r = Resources{
		WebSocket: WebSocket,
	}

	return &r
}

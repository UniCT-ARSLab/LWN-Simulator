package websocket

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	gorilla "github.com/gorilla/websocket"
)

var (
	ErrConnectionClosed = errors.New("websocket connection is closed")
	ErrSlowConsumer     = errors.New("websocket client is not consuming messages fast enough")
)

const (
	readLimit      int64 = 64 * 1024
	readTimeout          = 60 * time.Second
	writeTimeout         = 10 * time.Second
	pingInterval         = 45 * time.Second
	outboundBuffer       = 256
)

// Server owns the transport-level configuration shared by all websocket
// endpoints. Domain handlers decide what messages are sent over a connection.
type Server struct {
	upgrader gorilla.Upgrader
}

// NewServer creates a websocket transport with a caller-provided origin
// policy. Keeping the policy outside the connection makes the transport
// reusable while allowing the HTTP application to enforce its own policy.
func NewServer(checkOrigin func(*http.Request) bool) *Server {
	if checkOrigin == nil {
		checkOrigin = func(*http.Request) bool { return true }
	}

	return &Server{
		upgrader: gorilla.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 8192,
			CheckOrigin:     checkOrigin,
		},
	}
}

// Upgrade accepts an HTTP request and starts the read and write pumps. The
// read pump consumes client frames so that close and pong frames are handled
// promptly; application messages are sent through SendJSON.
func (s *Server) Upgrade(writer http.ResponseWriter, request *http.Request) (*Connection, error) {
	socket, err := s.upgrader.Upgrade(writer, request, nil)
	if err != nil {
		return nil, err
	}

	connection := &Connection{
		socket: socket,
		send:   make(chan outboundMessage, outboundBuffer),
		done:   make(chan struct{}),
	}
	go connection.readPump()
	go connection.writePump()

	return connection, nil
}

// Connection is a single bidirectional websocket transport. Only the write
// pump writes to the underlying socket, which keeps concurrent event sources
// safe when several parts of the application publish updates at once.
type Connection struct {
	socket *gorilla.Conn
	send   chan outboundMessage
	done   chan struct{}
	once   sync.Once
}

type outboundMessage struct {
	payload []byte
	flushed chan error
}

// Done closes when the peer disconnects or the application closes the
// connection.
func (c *Connection) Done() <-chan struct{} {
	return c.done
}

// SendJSON queues a JSON message without blocking the simulation loop. A slow
// client is disconnected instead of being allowed to grow unbounded memory.
func (c *Connection) SendJSON(value any) error {
	message, err := json.Marshal(value)
	if err != nil {
		return err
	}

	select {
	case <-c.done:
		return ErrConnectionClosed
	default:
	}

	select {
	case c.send <- outboundMessage{payload: message}:
		return nil
	case <-c.done:
		return ErrConnectionClosed
	default:
		c.Close()
		return ErrSlowConsumer
	}
}

// SendJSONAndClose delivers one final message through the write pump before
// closing the connection. It is useful for protocol errors that must reach a
// client before the handler returns.
func (c *Connection) SendJSONAndClose(value any) error {
	message, err := json.Marshal(value)
	if err != nil {
		return err
	}
	flushed := make(chan error, 1)
	select {
	case <-c.done:
		return ErrConnectionClosed
	case c.send <- outboundMessage{payload: message, flushed: flushed}:
	}

	select {
	case err := <-flushed:
		c.Close()
		return err
	case <-c.done:
		return ErrConnectionClosed
	}
}

// Close is safe to call from any goroutine and is idempotent.
func (c *Connection) Close() {
	c.once.Do(func() {
		close(c.done)
		_ = c.socket.Close()
	})
}

func (c *Connection) readPump() {
	defer c.Close()
	c.socket.SetReadLimit(readLimit)
	_ = c.socket.SetReadDeadline(time.Now().Add(readTimeout))
	c.socket.SetPongHandler(func(string) error {
		return c.socket.SetReadDeadline(time.Now().Add(readTimeout))
	})

	for {
		if _, _, err := c.socket.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Connection) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	defer c.Close()

	for {
		select {
		case message := <-c.send:
			_ = c.socket.SetWriteDeadline(time.Now().Add(writeTimeout))
			err := c.socket.WriteMessage(gorilla.TextMessage, message.payload)
			if message.flushed != nil {
				message.flushed <- err
			}
			if err != nil {
				return
			}
		case <-ticker.C:
			_ = c.socket.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.socket.WriteMessage(gorilla.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

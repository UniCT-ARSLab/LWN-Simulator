package websocket

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gorilla "github.com/gorilla/websocket"
)

func TestConnectionSendsJSONAndClosesIdempotently(t *testing.T) {
	server := NewServer(nil)
	messageReady := make(chan struct{})
	var accepted *Connection

	httpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := server.Upgrade(writer, request)
		if err != nil {
			return
		}
		accepted = connection
		close(messageReady)
		<-connection.Done()
	}))
	defer httpServer.Close()

	connection, _, err := gorilla.DefaultDialer.Dial("ws"+httpServer.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer connection.Close()

	select {
	case <-messageReady:
	case <-time.After(time.Second):
		t.Fatal("websocket server did not accept the connection")
	}

	if err := accepted.SendJSON(map[string]string{"type": "event"}); err != nil {
		t.Fatalf("send websocket message: %v", err)
	}

	_ = connection.SetReadDeadline(time.Now().Add(time.Second))
	var message map[string]string
	if err := connection.ReadJSON(&message); err != nil {
		t.Fatalf("read websocket message: %v", err)
	}
	if message["type"] != "event" {
		t.Fatalf("unexpected websocket message: %#v", message)
	}

	accepted.Close()
	accepted.Close()
	select {
	case <-accepted.Done():
	case <-time.After(time.Second):
		t.Fatal("websocket connection did not close")
	}
	if err := accepted.SendJSON(map[string]string{"type": "after-close"}); err != ErrConnectionClosed {
		t.Fatalf("send after close error = %v, want %v", err, ErrConnectionClosed)
	}
}

func TestServerAppliesOriginPolicy(t *testing.T) {
	server := NewServer(func(request *http.Request) bool {
		return request.Header.Get("Origin") == "https://allowed.example"
	})
	httpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		connection, err := server.Upgrade(writer, request)
		if err == nil {
			defer connection.Close()
		}
	}))
	defer httpServer.Close()

	dialer := gorilla.Dialer{}
	badHeaders := http.Header{"Origin": []string{"https://blocked.example"}}
	if _, _, err := dialer.Dial("ws"+httpServer.URL[len("http"):], badHeaders); err == nil {
		t.Fatal("blocked websocket origin was accepted")
	}
}

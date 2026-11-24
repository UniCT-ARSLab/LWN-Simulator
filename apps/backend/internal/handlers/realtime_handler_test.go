package handlers

import (
	"net/http/httptest"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"lwn-simulator-backend/internal/realtime"
)

func TestRealtimeHandlerBroadcastsSharedMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hub := realtime.NewHub()
	router := gin.New()
	router.GET("/ws", NewRealtimeHandler(hub).WebSocket)
	server := httptest.NewServer(router)
	defer server.Close()

	connection, _, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):]+"/ws", nil)
	if err != nil {
		t.Fatalf("dial realtime websocket: %v", err)
	}
	defer connection.Close()

	hub.Publish(contracts.RealtimeWebSocketMessage{
		Type: contracts.RealtimeDeviceCreatedMessage,
	})

	var message contracts.RealtimeWebSocketMessage
	if err := connection.ReadJSON(&message); err != nil {
		t.Fatalf("read realtime message: %v", err)
	}
	if message.Type != contracts.RealtimeDeviceCreatedMessage {
		t.Fatalf("unexpected realtime message type: %s", message.Type)
	}
}

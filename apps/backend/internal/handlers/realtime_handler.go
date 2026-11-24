package handlers

import (
	"time"

	"github.com/gin-gonic/gin"
	"lwn-simulator-backend/internal/realtime"
	ws "lwn-simulator-backend/internal/websocket"
)

type RealtimeHandler struct {
	hub       *realtime.Hub
	websocket *ws.Server
}

func NewRealtimeHandler(hub *realtime.Hub) *RealtimeHandler {
	return &RealtimeHandler{hub: hub, websocket: ws.NewServer(allowWebSocketOrigin)}
}

func (h *RealtimeHandler) WebSocket(c *gin.Context) {
	subscription := h.hub.Subscribe(c.Query("profileID"))
	defer subscription.Close()

	connection, err := h.websocket.Upgrade(c.Writer, c.Request)
	if err != nil {
		return
	}
	defer connection.Close()

	for {
		select {
		case <-connection.Done():
			return
		case message, ok := <-subscription.Updates:
			if !ok {
				return
			}
			message.TimestampMilliseconds = time.Now().UnixMilli()
			if err := connection.SendJSON(message); err != nil {
				return
			}
		}
	}
}

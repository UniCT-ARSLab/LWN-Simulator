package realtime

import (
	"sync"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

const subscriberBufferSize = 4096

// Subscription is a best-effort realtime stream. Consumers can recover a
// missed update through the corresponding HTTP endpoint after reconnecting.
type Subscription struct {
	Updates <-chan contracts.RealtimeWebSocketMessage
	Close   func()
}

type subscriber struct {
	profileID string
	updates   chan contracts.RealtimeWebSocketMessage
}

// Hub fans application updates out to every connected frontend. A single hub
// is shared by hardware, simulation and log producers so multiple browser
// tabs observe the same state transitions.
type Hub struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[uint64]subscriber
}

func NewHub() *Hub {
	return &Hub{subscribers: make(map[uint64]subscriber)}
}

func (h *Hub) Subscribe(profileID ...string) Subscription {
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	updates := make(chan contracts.RealtimeWebSocketMessage, subscriberBufferSize)
	selectedProfile := ""
	if len(profileID) > 0 {
		selectedProfile = profileID[0]
	}
	h.subscribers[id] = subscriber{profileID: selectedProfile, updates: updates}
	h.mu.Unlock()

	return Subscription{
		Updates: updates,
		Close: func() {
			h.mu.Lock()
			if subscriber, ok := h.subscribers[id]; ok {
				delete(h.subscribers, id)
				close(subscriber.updates)
			}
			h.mu.Unlock()
		},
	}
}

func (h *Hub) Publish(message contracts.RealtimeWebSocketMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, subscriber := range h.subscribers {
		if subscriber.profileID != "" && message.ProfileID != nil && *message.ProfileID != subscriber.profileID {
			continue
		}
		select {
		case subscriber.updates <- message:
		default:
			// The websocket transport will reconnect and consumers can recover
			// durable state through HTTP. Never block a simulation or CRUD call
			// on a slow browser tab.
		}
	}
}

package realtime

import (
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestHubPublishesToEverySubscriber(t *testing.T) {
	hub := NewHub()
	first := hub.Subscribe()
	second := hub.Subscribe()
	defer first.Close()
	defer second.Close()

	want := contracts.RealtimeWebSocketMessage{
		Type:                  contracts.RealtimeDeviceCreatedMessage,
		TimestampMilliseconds: 1,
		ResourceID:            stringPointer("device-1"),
	}
	hub.Publish(want)

	for name, subscription := range map[string]Subscription{"first": first, "second": second} {
		t.Run(name, func(t *testing.T) {
			select {
			case got := <-subscription.Updates:
				if got.Type != want.Type || got.ResourceID == nil || *got.ResourceID != "device-1" {
					t.Fatalf("unexpected realtime message: %+v", got)
				}
			case <-time.After(time.Second):
				t.Fatal("subscriber did not receive realtime message")
			}
		})
	}
}

func TestHubCloseIsIdempotentAndClosesStream(t *testing.T) {
	hub := NewHub()
	subscription := hub.Subscribe()
	subscription.Close()
	subscription.Close()

	select {
	case _, ok := <-subscription.Updates:
		if ok {
			t.Fatal("closed subscription still returned a message")
		}
	case <-time.After(time.Second):
		t.Fatal("subscription was not closed")
	}
}

func TestHubFiltersProfileSubscriptions(t *testing.T) {
	hub := NewHub()
	profileID := "550e8400-e29b-41d4-a716-446655440000"
	subscription := hub.Subscribe(profileID)
	defer subscription.Close()

	otherProfile := "6ba7b810-9dad-41d1-80b4-00c04fd430c8"
	hub.Publish(contracts.RealtimeWebSocketMessage{ProfileID: &otherProfile})
	hub.Publish(contracts.RealtimeWebSocketMessage{ProfileID: &profileID})

	select {
	case message := <-subscription.Updates:
		if message.ProfileID == nil || *message.ProfileID != profileID {
			t.Fatalf("received update for the wrong profile: %+v", message)
		}
	case <-time.After(time.Second):
		t.Fatal("profile update was not delivered")
	}
}

func stringPointer(value string) *string { return &value }

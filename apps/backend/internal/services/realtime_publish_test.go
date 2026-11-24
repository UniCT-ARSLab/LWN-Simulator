package services

import (
	"testing"
	"time"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

type recordingRealtimePublisher struct {
	messages chan contracts.RealtimeWebSocketMessage
}

func (publisher *recordingRealtimePublisher) Publish(message contracts.RealtimeWebSocketMessage) {
	publisher.messages <- message
}

func TestDeviceMutationsPublishRealtimeMessages(t *testing.T) {
	repository := &mockDeviceRepository{}
	publisher := &recordingRealtimePublisher{messages: make(chan contracts.RealtimeWebSocketMessage, 3)}
	service := NewDeviceService(repository)
	service.SetRealtimePublisher(publisher)

	created, err := service.CreateDevice(contracts.CreateDeviceRequest{DevEUI: "0102030405060708", Name: "Device"})
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	if message := receiveRealtimeMessage(t, publisher.messages); message.Type != contracts.RealtimeDeviceCreatedMessage || message.Device == nil {
		t.Fatalf("unexpected create message: %+v", message)
	}

	updated := created.Device
	updated.Name = "Updated device"
	if _, err := service.UpdateDevice(contracts.UpdateDeviceRequest{ID: updated.ID, Device: updated}); err != nil {
		t.Fatalf("update device: %v", err)
	}
	if message := receiveRealtimeMessage(t, publisher.messages); message.Type != contracts.RealtimeDeviceUpdatedMessage || message.Device == nil || message.Device.Name != updated.Name {
		t.Fatalf("unexpected update message: %+v", message)
	}

	if _, err := service.DeleteDevice(contracts.DeleteDeviceRequest{ID: updated.ID}); err != nil {
		t.Fatalf("delete device: %v", err)
	}
	if message := receiveRealtimeMessage(t, publisher.messages); message.Type != contracts.RealtimeDeviceDeletedMessage || message.ResourceID == nil || *message.ResourceID != updated.ID {
		t.Fatalf("unexpected delete message: %+v", message)
	}
}

func TestGatewayMutationsPublishRealtimeMessages(t *testing.T) {
	repository := &mockGatewayRepository{}
	publisher := &recordingRealtimePublisher{messages: make(chan contracts.RealtimeWebSocketMessage, 3)}
	service := NewGatewayService(repository)
	service.SetRealtimePublisher(publisher)

	created, err := service.CreateGateway(validCreateGatewayRequest())
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	if message := receiveRealtimeMessage(t, publisher.messages); message.Type != contracts.RealtimeGatewayCreatedMessage || message.Gateway == nil {
		t.Fatalf("unexpected create message: %+v", message)
	}

	updated := created
	updated.Name = "Updated gateway"
	if _, err := service.UpdateGateway(contracts.UpdateGatewayRequest{ID: updated.ID, Gateway: updated}); err != nil {
		t.Fatalf("update gateway: %v", err)
	}
	if message := receiveRealtimeMessage(t, publisher.messages); message.Type != contracts.RealtimeGatewayUpdatedMessage || message.Gateway == nil || message.Gateway.Name != updated.Name {
		t.Fatalf("unexpected update message: %+v", message)
	}

	if _, err := service.DeleteGateway(contracts.DeleteGatewayRequest{ID: updated.ID}); err != nil {
		t.Fatalf("delete gateway: %v", err)
	}
	if message := receiveRealtimeMessage(t, publisher.messages); message.Type != contracts.RealtimeGatewayDeletedMessage || message.ResourceID == nil || *message.ResourceID != updated.ID {
		t.Fatalf("unexpected delete message: %+v", message)
	}
}

func receiveRealtimeMessage(t *testing.T, messages <-chan contracts.RealtimeWebSocketMessage) contracts.RealtimeWebSocketMessage {
	t.Helper()
	select {
	case message := <-messages:
		return message
	case <-time.After(time.Second):
		t.Fatal("realtime message was not published")
		return contracts.RealtimeWebSocketMessage{}
	}
}

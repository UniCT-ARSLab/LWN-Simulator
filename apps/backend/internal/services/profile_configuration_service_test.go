package services

import (
	"os"
	"path/filepath"
	"testing"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
)

func TestProfileConfigurationServiceReturnsDefaultsWhenConfigurationIsMissing(t *testing.T) {
	service := NewProfileConfigurationService(filepath.Join(t.TempDir(), "gateway-bridge.json"), "")

	config, err := service.GetGatewayBridge()
	if err != nil {
		t.Fatalf("get missing gateway bridge configuration: %v", err)
	}
	if config != defaultGatewayBridgeConfig {
		t.Fatalf("unexpected default gateway bridge configuration: %+v", config)
	}
}

func TestProfileConfigurationServicePersistsConfigurationAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "gateway-bridge.json")
	service := NewProfileConfigurationService(path, "")
	wanted := contracts.GatewayBridgeConfig{
		Enabled: true,
		Address: "127.0.0.1",
		Port:    1701,
	}

	updated, err := service.UpdateGatewayBridge(wanted)
	if err != nil {
		t.Fatalf("update gateway bridge configuration: %v", err)
	}
	if updated != wanted {
		t.Fatalf("unexpected updated configuration: %+v", updated)
	}

	loaded, err := service.GetGatewayBridge()
	if err != nil {
		t.Fatalf("reload gateway bridge configuration: %v", err)
	}
	if loaded != wanted {
		t.Fatalf("configuration was not persisted: %+v", loaded)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("configuration file was not created: %v", err)
	}
}

func TestProfileConfigurationServiceNormalizesUnsafeValues(t *testing.T) {
	service := NewProfileConfigurationService(filepath.Join(t.TempDir(), "gateway-bridge.json"), "")

	updated, err := service.UpdateGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: true,
		Port:    0,
	})
	if err != nil {
		t.Fatalf("update invalid gateway bridge configuration: %v", err)
	}
	if updated.Address != defaultGatewayBridgeConfig.Address || updated.Port != defaultGatewayBridgeConfig.Port {
		t.Fatalf("configuration was not normalized: %+v", updated)
	}
}

func TestProfileConfigurationServiceRejectsMalformedConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway-bridge.json")
	if err := os.WriteFile(path, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := NewProfileConfigurationService(path, "").GetGatewayBridge()
	if err == nil {
		t.Fatal("expected malformed configuration to fail")
	}
}

func TestProfileConfigurationServiceRejectsUnresolvableEnabledAddress(t *testing.T) {
	service := NewProfileConfigurationService(filepath.Join(t.TempDir(), "gateway-bridge.json"), "")

	_, err := service.UpdateGatewayBridge(contracts.GatewayBridgeConfig{
		Enabled: true,
		Address: "not/an-address",
		Port:    1700,
	})
	if err == nil {
		t.Fatal("unresolvable gateway bridge address was accepted")
	}
}

func TestProfileConfigurationServicePublishesProfileScopedUpdates(t *testing.T) {
	publisher := &recordingRealtimePublisher{messages: make(chan contracts.RealtimeWebSocketMessage, 1)}
	service := NewProfileConfigurationService(filepath.Join(t.TempDir(), "gateway-bridge.json"), "550e8400-e29b-41d4-a716-446655440000")
	service.SetRealtimePublisher(publisher)

	config := contracts.GatewayBridgeConfig{Enabled: true, Address: "127.0.0.1", Port: 1701}
	if _, err := service.UpdateGatewayBridge(config); err != nil {
		t.Fatalf("update gateway bridge configuration: %v", err)
	}

	message := receiveRealtimeMessage(t, publisher.messages)
	if message.Type != contracts.RealtimeProfileGatewayBridgeUpdatedMessage {
		t.Fatalf("unexpected realtime message type: %+v", message)
	}
	if message.ProfileID == nil || *message.ProfileID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("unexpected profile id: %+v", message.ProfileID)
	}
	if message.GatewayBridge == nil || *message.GatewayBridge != config {
		t.Fatalf("unexpected gateway bridge configuration: %+v", message.GatewayBridge)
	}
}

package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	contracts "github.com/Giuseppe-Compagnone/lwn-contracts/generated"
	engineValidation "github.com/Giuseppe-Compagnone/lwn-engine/validation"
	"lwn-simulator-backend/internal/apperrors"
)

var defaultGatewayBridgeConfig = contracts.GatewayBridgeConfig{
	Enabled: false,
	Address: "0.0.0.0",
	Port:    1700,
}

// ProfileConfigurationService stores configuration that belongs to one
// profile and must be shared by every frontend connected to the backend.
type ProfileConfigurationService struct {
	gatewayBridgePath string
	profileID         string
	realtime          RealtimePublisher
	mu                sync.Mutex
}

func NewProfileConfigurationService(gatewayBridgePath string, profileID string) *ProfileConfigurationService {
	return &ProfileConfigurationService{
		gatewayBridgePath: gatewayBridgePath,
		profileID:         profileID,
	}
}

func (s *ProfileConfigurationService) SetRealtimePublisher(publisher RealtimePublisher) {
	s.mu.Lock()
	s.realtime = publisher
	s.mu.Unlock()
}

func (s *ProfileConfigurationService) GetGatewayBridge() (contracts.GatewayBridgeConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.gatewayBridgePath)
	if errors.Is(err, os.ErrNotExist) {
		return defaultGatewayBridgeConfig, nil
	}
	if err != nil {
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("read gateway bridge configuration: %w", err)
	}
	var config contracts.GatewayBridgeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("decode gateway bridge configuration: %w", err)
	}
	return normalizeGatewayBridgeConfig(config), nil
}

func (s *ProfileConfigurationService) UpdateGatewayBridge(config contracts.GatewayBridgeConfig) (contracts.GatewayBridgeConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	config = normalizeGatewayBridgeConfig(config)
	if config.Enabled {
		if err := engineValidation.ValidateSimulationConfig(contracts.SimulationConfig{
			Speed:         1,
			GatewayBridge: &config,
		}); err != nil {
			return contracts.GatewayBridgeConfig{}, apperrors.Invalid("invalid gateway bridge configuration: %s", err)
		}
	}
	data, err := json.Marshal(config)
	if err != nil {
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("encode gateway bridge configuration: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.gatewayBridgePath), 0700); err != nil {
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("create profile configuration directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.gatewayBridgePath), ".gateway-bridge-*.tmp")
	if err != nil {
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("create gateway bridge configuration: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0600); err != nil {
		_ = temporary.Close()
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("set gateway bridge configuration permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("write gateway bridge configuration: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("sync gateway bridge configuration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("close gateway bridge configuration: %w", err)
	}
	if err := os.Rename(temporaryName, s.gatewayBridgePath); err != nil {
		return contracts.GatewayBridgeConfig{}, fmt.Errorf("replace gateway bridge configuration: %w", err)
	}
	if s.realtime != nil {
		profileID := s.profileID
		s.realtime.Publish(contracts.RealtimeWebSocketMessage{
			Type:          contracts.RealtimeProfileGatewayBridgeUpdatedMessage,
			ProfileID:     &profileID,
			GatewayBridge: &config,
		})
	}
	return config, nil
}

func normalizeGatewayBridgeConfig(config contracts.GatewayBridgeConfig) contracts.GatewayBridgeConfig {
	if config.Address == "" {
		config.Address = defaultGatewayBridgeConfig.Address
	}
	if config.Port < 1 || config.Port > 65535 {
		config.Port = defaultGatewayBridgeConfig.Port
	}
	return config
}

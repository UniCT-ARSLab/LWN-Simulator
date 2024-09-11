package models

import (
	"encoding/json"
	"fmt"
	"os"
)

// ServerConfig holds the configuration for the server including address, ports, and other settings.
type ServerConfig struct {
	Address       string `json:"address"`
	Port          int    `json:"port"`
	MetricsPort   int    `json:"metricsPort"`
	ConfigDirname string `json:"configDirname"`
	AutoStart     bool   `json:"autoStart"`
}

// GetConfigFile loads the configuration from the specified file path, parses it as JSON,
// and returns a ServerConfig instance. It returns an error if the file cannot be read or parsed.
func GetConfigFile(path string) (*ServerConfig, error) {
	config := &ServerConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return config, nil
}

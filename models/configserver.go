package models

import (
	"encoding/json"
	"io/ioutil"
)

type ServerConfig struct {
	Address       string `json:"address"`
	Port          int    `json:"port"`
	ConfigDirname string `json:"configDirname"`
}

func GetConfigFile(path string) (*ServerConfig, error) {

	config := ServerConfig{}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

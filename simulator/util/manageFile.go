package util

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/arslab/lwnsimulator/models"
)

// GetPath returns the path of the config directory path, and creates it if it does not exist
func GetPath() (string, error) {
	path := GetConfigDirname()
	err := CreateConfigDir(path)
	if err != nil {
		return "", err
	}
	return path, nil
}

// GetConfigDirname returns the field configDirname from the config file
func GetConfigDirname() string {
	// Read the config.json file located alongside the executable
	info, err := models.GetConfigFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	return info.ConfigDirname
}

// CreateConfigDir creates the config directory in the provided path
func CreateConfigDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// CreateConfigFiles creates the config files in the config directory
func CreateConfigFiles() {
	var paths [3]string
	pathDir, err := GetPath()
	if err != nil {
		log.Fatal(err)
	}
	paths[0] = pathDir + "/simulator.json"
	paths[1] = pathDir + "/gateways.json"
	paths[2] = pathDir + "/devices.json"
	emptyData := "{}"
	for i, _ := range paths {
		_, err = os.Create(paths[i])
		if err != nil {
			log.Fatal(err)
		}
		err = WriteConfigFile(paths[i], []byte(emptyData))
		if err != nil {
			log.Fatal(err)
		}
	}
}

// RecoverConfigFile reads the data from the file in the path and stores it in the provided interface
func RecoverConfigFile(path string, v interface{}) error {
	// In case of first execution, create the config files
	if _, err := os.Stat(path); os.IsNotExist(err) {
		CreateConfigFiles()
	}
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(fileBytes, &v)
}

// WriteConfigFile writes the data to the file in the path
func WriteConfigFile(path string, data []byte) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_, err = os.Create(path)
		if err != nil {
			log.Fatal(fmt.Sprintf("Error creating file: %v", err))
		}
	}
	return os.WriteFile(path, data, os.ModePerm)
}

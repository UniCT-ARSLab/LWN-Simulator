package main

import (
	"log"

	"github.com/arslab/lwnsimulator/models"
	ws "github.com/arslab/lwnsimulator/webserver"
)

func main() {

	var info *models.ServerConfig
	var err error

	info, err = models.GetConfigFile("config.json")
	if err != nil {

		log.Fatal(err)

	}

	WebServer := ws.NewWebServer(info)

	WebServer.Run()

}

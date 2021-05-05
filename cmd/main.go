package main

import (
	"log"

	cnt "github.com/arslab/lwnsimulator/controllers"
	"github.com/arslab/lwnsimulator/models"
	repo "github.com/arslab/lwnsimulator/repositories"
	ws "github.com/arslab/lwnsimulator/webserver"
)

func main() {

	var info *models.ServerConfig
	var err error

	simulatorRepository := repo.NewSimulatorRepository()
	simulatorController := cnt.NewSimulatorController(simulatorRepository)
	simulatorController.GetIstance()

	log.Println("LWN Simulator is online...")

	info, err = models.GetConfigFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	WebServer := ws.NewWebServer(info, simulatorController)
	WebServer.Run()

}

package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus/promhttp"

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

	go startMetrics(info)

	WebServer := ws.NewWebServer(info, simulatorController)
	WebServer.Run()
}

func startMetrics(info *models.ServerConfig) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(info.Address+":"+strconv.Itoa(info.MetricsPort), nil)
	if err != nil {
		log.Println("[Metrics] [ERROR]:", err.Error())
	}
}

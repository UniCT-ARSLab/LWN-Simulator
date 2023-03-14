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

	var cfg *models.ServerConfig
	var err error

	simulatorRepository := repo.NewSimulatorRepository()
	simulatorController := cnt.NewSimulatorController(simulatorRepository)
	simulatorController.GetIstance()

	log.Println("LWN Simulator is online...")

	cfg, err = models.GetConfigFile("config.json")
	if err != nil {
		log.Fatal(err)
	}

	go startMetrics(cfg)

	if cfg.AutoStart == true {
		// start the simulator automatically
		log.Println("Autostart of the simulator")
		simulatorController.Run()
	} else {
		log.Println("Autostart not enabled")
	}

	WebServer := ws.NewWebServer(cfg, simulatorController)
	WebServer.Run()
}

func startMetrics(cfg *models.ServerConfig) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(cfg.Address+":"+strconv.Itoa(cfg.MetricsPort), nil)
	if err != nil {
		log.Println("[Metrics] [ERROR]:", err.Error())
	}
}

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

// Entry point of the program.
func main() {
	var cfg *models.ServerConfig
	var err error
	// Load the configuration file, and if there is an error, log it and terminate the program.
	cfg, err = models.GetConfigFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	// Create a new simulator controller and repository.
	simulatorRepository := repo.NewSimulatorRepository()
	simulatorController := cnt.NewSimulatorController(simulatorRepository)
	simulatorController.GetIstance()
	log.Println("LWN Simulator is ready to start...")
	// Start the metrics server.
	go startMetrics(cfg)
	// If the autoStart flag is set to true, start the simulator automatically.
	if cfg.AutoStart {
		log.Println("Autostarting the simulation")
		simulatorController.Run()
	} else {
		log.Println("Autostart not enabled")
	}
	// Start the web server and serve WebUI
	WebServer := ws.NewWebServer(cfg, simulatorController)
	WebServer.Run()
	log.Println("webUI online")
}

// Prometheus metrics server
func startMetrics(cfg *models.ServerConfig) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(cfg.Address+":"+strconv.Itoa(cfg.MetricsPort), nil)
	if err != nil {
		log.Println("[Metrics] [ERROR]:", err.Error())
	}
}

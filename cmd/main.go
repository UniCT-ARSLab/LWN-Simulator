package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"strconv"

	cnt "github.com/arslab/lwnsimulator/controllers"
	"github.com/arslab/lwnsimulator/models"
	repo "github.com/arslab/lwnsimulator/repositories"
	"github.com/arslab/lwnsimulator/shared"
	ws "github.com/arslab/lwnsimulator/webserver"
)

// Entry point of the program.
func main() {
	// Load the configuration file, and if there is an error, log it and terminate the program.
	cfg, err := models.GetConfigFile("config.json")
	if err != nil {
		log.Fatal(err)
	}
	// Check if the verbose flag is set to true, and if so, enable verbose logging.
	if cfg.Verbose {
		shared.Verbose = true
		shared.DebugPrint("Verbose mode enabled")
	}
	// Create a new simulator controller and repository.
	simulatorRepository := repo.NewSimulatorRepository()
	simulatorController := cnt.NewSimulatorController(simulatorRepository)
	simulatorController.GetInstance()
	log.Printf("LWN Simulator (%s) is ready to start...\n", shared.Version)
	// Start the metrics server.
	go startMetrics(cfg)
	// If the autoStart flag is set to true, start the simulator automatically.
	if cfg.AutoStart {
		log.Println("Auto-starting the simulation")
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

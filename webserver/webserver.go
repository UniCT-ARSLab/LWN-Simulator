package webserver

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	cnt "github.com/arslab/lwnsimulator/controllers"
	"github.com/arslab/lwnsimulator/models"
	dev "github.com/arslab/lwnsimulator/simulator/components/device"
	rp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters"
	mrp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters/models_rp"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	"github.com/arslab/lwnsimulator/socket"
	_ "github.com/arslab/lwnsimulator/webserver/statik"
	"github.com/brocaar/lorawan"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/rakyll/statik/fs"
)

// WebServer represents a web server configuration including address, port, router setup, and server socket.
type WebServer struct {
	Address      string           // Address of the web server
	Port         int              // Port of the web server
	Router       *gin.Engine      // Router of the web server
	ServerSocket *socketio.Server // ServerSocket of the web server
}

// Global variables
var (
	simulatorController cnt.SimulatorController // simulatorController is an instance of the cSimulatorController interface for managing simulator operations.
	configuration       *models.ServerConfig    // configuration is a pointer to models.ServerConfig struct which holds the server's configuration settings.
)

// NewWebServer creates a new web server instance with the given configuration and simulator controller.
func NewWebServer(config *models.ServerConfig, controller cnt.SimulatorController) *WebServer {
	// Storing the configuration and controller instances in the global variables.
	configuration = config
	simulatorController = controller
	serverSocket := newServerSocket()
	// Start the server socket in a separate goroutine due to its blocking nature.
	// If an error occurs, log it and terminate the program.
	go func() {
		err := serverSocket.Serve()
		if err != nil {
			log.Fatal(fmt.Errorf("[WS] [ERROR] [SERVERSOCKET]: %w", err))
		}
	}()
	// Initialize the Gin router and setting up the CORS configuration.
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	configCors := cors.DefaultConfig()
	configCors.AllowAllOrigins = true
	configCors.AllowHeaders = []string{"Origin", "Access-Control-Allow-Origin",
		"Access-Control-Allow-Headers", "Content-type"}
	configCors.AllowMethods = []string{"GET", "POST", "DELETE", "OPTIONS"}
	configCors.AllowCredentials = true
	router.Use(cors.New(configCors))
	// Recovery middleware recovers from any panics and writes a 500 if there was one.
	router.Use(gin.Recovery())
	// Create a new WebServer instance with the given configuration and router.
	ws := WebServer{
		Address:      configuration.Address,
		Port:         configuration.Port,
		Router:       router,
		ServerSocket: serverSocket,
	}
	// Serve the static files using the statik file system.
	staticFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}
	staticGroup := router.Group("/dashboard")
	staticGroup.StaticFS("/", staticFS)
	// Set up the API routes.
	apiRoutes := router.Group("/api")
	{
		apiRoutes.GET("/start", startSimulator)        // Start the simulator
		apiRoutes.GET("/stop", stopSimulator)          // Stop the simulator
		apiRoutes.GET("/status", simulatorStatus)      // Get the simulator status (running or stopped)
		apiRoutes.GET("/bridge", getRemoteAddress)     // Get the remote address of the bridge
		apiRoutes.GET("/gateways", getGateways)        // Get the list of gateways
		apiRoutes.GET("/devices", getDevices)          // Get the list of devices
		apiRoutes.POST("/add-device", addDevice)       // Add a new device
		apiRoutes.POST("/up-device", updateDevice)     // Update a device
		apiRoutes.POST("/del-device", deleteDevice)    // Delete a device
		apiRoutes.POST("/del-gateway", deleteGateway)  // Delete a gateway
		apiRoutes.POST("/add-gateway", addGateway)     // Add a new gateway
		apiRoutes.POST("/up-gateway", updateGateway)   // Update a gateway
		apiRoutes.POST("/bridge/save", saveInfoBridge) // Save the remote address of the bridge
	}
	// Set up the WebSocket routes.
	router.GET("/socket.io/*any", gin.WrapH(serverSocket))
	router.POST("/socket.io/*any", gin.WrapH(serverSocket))
	// Redirect the root path to the dashboard.
	router.GET("/", func(context *gin.Context) { context.Redirect(http.StatusMovedPermanently, "/dashboard") })
	return &ws
}

// newServerSocket creates a new server socket instance and sets up the socket events.
func newServerSocket() *socketio.Server {
	serverSocket := socketio.NewServer(nil)
	serverSocket.OnConnect("/", func(s socketio.Conn) error {
		log.Println("[WS]: Socket connected")
		s.SetContext("")
		simulatorController.AddWebSocket(&s)
		return nil
	})
	serverSocket.OnDisconnect("/", func(s socketio.Conn, reason string) {
		// Remove the socket from the list of connected sockets
		serverSocket.Remove(s.ID())
		_ = s.Close()
	})
	serverSocket.OnEvent("/", socket.EventToggleStateDevice, func(s socketio.Conn, Id int) {
		simulatorController.ToggleStateDevice(Id)
	})
	serverSocket.OnEvent("/", socket.EventToggleStateGateway, func(s socketio.Conn, Id int) {
		simulatorController.ToggleStateGateway(Id)
	})
	serverSocket.OnEvent("/", socket.EventMacCommand, func(s socketio.Conn, data socket.MacCommand) {

		switch data.CID {
		case "DeviceTimeReq":
			simulatorController.SendMACCommand(lorawan.DeviceTimeReq, data)
		case "LinkCheckReq":
			simulatorController.SendMACCommand(lorawan.LinkCheckReq, data)
		case "PingSlotInfoReq":
			simulatorController.SendMACCommand(lorawan.PingSlotInfoReq, data)
		}

	})
	serverSocket.OnEvent("/", socket.EventChangePayload, func(s socketio.Conn, data socket.NewPayload) (string, bool) {
		return simulatorController.ChangePayload(data)
	})
	serverSocket.OnEvent("/", socket.EventSendUplink, func(s socketio.Conn, data socket.NewPayload) {
		simulatorController.SendUplink(data)
	})
	serverSocket.OnEvent("/", socket.EventGetParameters, func(s socketio.Conn, code int) mrp.Informations {
		return rp.GetInfo(code)
	})
	serverSocket.OnEvent("/", socket.EventChangeLocation, func(s socketio.Conn, info socket.NewLocation) bool {
		return simulatorController.ChangeLocation(info)
	})
	return serverSocket
}

// Run starts the web server and listens on the given address and port.
func (ws *WebServer) Run() {
	fullAddress := ws.Address + ":" + strconv.Itoa(ws.Port)
	log.Printf("[WS]: Listen [%s]", fullAddress)
	err := ws.Router.Run(fullAddress)
	// If an error occurs, log it and terminate the program.
	if err != nil {
		log.Fatal(fmt.Errorf("[WS] [ERROR]: %w", err))
	}
}

// --- API Handlers ---
// startSimulator starts the simulator
func startSimulator(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.Run())
}

// stopSimulator stops the simulator
func stopSimulator(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.Stop())
}

// simulatorStatus returns the status of the simulator
func simulatorStatus(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.Status())
}

// saveInfoBridge saves the remote address of the bridge
func saveInfoBridge(c *gin.Context) {
	var ns models.AddressIP
	err := c.BindJSON(&ns)
	// If an error occurs, return a bad request status.
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": simulatorController.SaveBridgeAddress(ns)})
}

// getRemoteAddress returns the remote address of the bridge
func getRemoteAddress(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.GetBridgeAddress())
}

// getGateways returns the list of gateways
func getGateways(c *gin.Context) {
	gws := simulatorController.GetGateways()
	c.JSON(http.StatusOK, gws)
}

// addGateway adds a new gateway
func addGateway(c *gin.Context) {
	var g gw.Gateway
	err := c.BindJSON(&g)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid request"})
		return
	}
	code, id, err := simulatorController.AddGateway(&g)
	errString := fmt.Sprintf("%v", err)
	c.JSON(http.StatusOK, gin.H{"status": errString, "code": code, "id": id})
}

// updateGateway updates a gateway
func updateGateway(c *gin.Context) {
	var g gw.Gateway
	err := c.BindJSON(&g)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid request"})
		return
	}
	code, err := simulatorController.UpdateGateway(&g)
	errString := fmt.Sprintf("%v", err)
	c.JSON(http.StatusOK, gin.H{"status": errString, "code": code})
}

// deleteGateway deletes a gateway
func deleteGateway(c *gin.Context) {
	Identifier := struct {
		Id int `json:"id"`
	}{}
	err := c.BindJSON(&Identifier)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": simulatorController.DeleteGateway(Identifier.Id)})
}

// getDevices returns the list of devices
func getDevices(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.GetDevices())
}

// addDevice adds a new device
func addDevice(c *gin.Context) {
	var device dev.Device
	err := c.BindJSON(&device)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid request"})
		return
	}
	code, id, err := simulatorController.AddDevice(&device)
	errString := fmt.Sprintf("%v", err)
	c.JSON(http.StatusOK, gin.H{"status": errString, "code": code, "id": id})
}

// updateDevice updates a device
func updateDevice(c *gin.Context) {
	var device dev.Device
	err := c.BindJSON(&device)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid request"})
		return
	}
	code, err := simulatorController.UpdateDevice(&device)
	errString := fmt.Sprintf("%v", err)
	c.JSON(http.StatusOK, gin.H{"status": errString, "code": code})
}

// deleteDevice deletes a device
func deleteDevice(c *gin.Context) {
	Identifier := struct {
		Id int `json:"id"`
	}{}
	err := c.BindJSON(&Identifier)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "Invalid request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": simulatorController.DeleteDevice(Identifier.Id)})
}

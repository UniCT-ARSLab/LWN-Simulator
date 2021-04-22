package webserver

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/brocaar/lorawan"

	cnt "github.com/arslab/lwnsimulator/controllers"
	"github.com/arslab/lwnsimulator/models"
	repo "github.com/arslab/lwnsimulator/repositories"
	rp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters"
	mrp "github.com/arslab/lwnsimulator/simulator/components/device/regional_parameters/models_rp"
	gw "github.com/arslab/lwnsimulator/simulator/components/gateway"
	"github.com/arslab/lwnsimulator/socket"
	"github.com/arslab/lwnsimulator/types"
	_ "github.com/arslab/lwnsimulator/webserver/statik"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/rakyll/statik/fs"
)

//WebServer type
type WebServer struct {
	Address      string
	Port         int
	Router       *gin.Engine
	ServerSocket *socketio.Server
}

var (
	simulatorRepository repo.SimulatorRepository = repo.NewSimulatorRepository()
	simulatorController cnt.SimulatorController  = cnt.NewSimulatorController(simulatorRepository)
	configuration       *models.ServerConfig
)

func NewWebServer(config *models.ServerConfig) *WebServer {

	serverSocket := newServerSocket()

	configuration = config

	go func() {

		err := serverSocket.Serve()

		if err != nil {
			log.Fatal(err)
		}

	}()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	configCors := cors.DefaultConfig()
	configCors.AllowAllOrigins = true
	configCors.AllowHeaders = []string{"Origin", "Access-Control-Allow-Origin",
		"Access-Control-Allow-Headers", "Content-type"}
	configCors.AllowMethods = []string{"GET", "POST", "DELETE", "OPTIONS"}
	configCors.AllowCredentials = true
	router.Use(cors.New(configCors))

	router.Use(gin.Recovery())

	ws := WebServer{
		Address:      configuration.Address,
		Port:         configuration.Port,
		Router:       router,
		ServerSocket: serverSocket,
	}

	staticFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	staticGroup := router.Group("/dashboard")
	staticGroup.StaticFS("/", staticFS)
	//router.Use(static.Serve("/", staticFS))

	apiRoutes := router.Group("/api")
	{
		apiRoutes.GET("/start", startSimulator)
		apiRoutes.GET("/stop", stopSimulator)
		apiRoutes.GET("/bridge", getRemoteAddress)
		apiRoutes.GET("/gateways", getGateways)
		apiRoutes.GET("/devices", getDevices)
		apiRoutes.POST("/add-device", addDevice)
		apiRoutes.POST("/up-device", updateDevice)
		apiRoutes.POST("/del-device", deleteDevice)
		apiRoutes.POST("/del-gateway", deleteGateway)
		apiRoutes.POST("/add-gateway", addGateway)
		apiRoutes.POST("/up-gateway", updateGateway)
		apiRoutes.POST("/bridge/save", saveInfoBridge)

	}

	router.GET("/socket.io/*any", gin.WrapH(serverSocket))
	router.POST("/socket.io/*any", gin.WrapH(serverSocket))

	router.GET("/", func(context *gin.Context) { context.Redirect(http.StatusMovedPermanently, "/dashboard") })

	return &ws
}

func startSimulator(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.Run())
}

func stopSimulator(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.Stop())
}

func saveInfoBridge(c *gin.Context) {

	var ns types.AddressIP
	c.BindJSON(&ns)

	c.JSON(http.StatusOK, gin.H{"status": simulatorController.SaveBridgeAddress(ns)})
}

func getRemoteAddress(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.GetBridgeAddress())
}

func getGateways(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.GetGateways())
}

func addGateway(c *gin.Context) {

	var g types.Gateway
	c.BindJSON(&g)

	code, err := simulatorController.AddGateway(g.Gw)
	errString := fmt.Sprintf("%v", err)

	c.JSON(http.StatusOK, gin.H{"status": errString, "code": code})

}

func updateGateway(c *gin.Context) {

	var g types.Gateway
	c.BindJSON(&g)

	code, err := simulatorController.UpdateGateway(g.Gw, g.Index)
	errString := fmt.Sprintf("%v", err)
	c.JSON(http.StatusOK, gin.H{"status": errString, "code": code})

}

func deleteGateway(c *gin.Context) {
	var g gw.Gateway

	c.BindJSON(&g)

	c.JSON(http.StatusOK, gin.H{"status": simulatorController.DeleteGateway(g.Info.MACAddress)})

}

func getDevices(c *gin.Context) {
	c.JSON(http.StatusOK, simulatorController.GetDevices())
}

func addDevice(c *gin.Context) {

	var d types.Device
	c.BindJSON(&d)

	code, err := simulatorController.AddDevice(d.Dev)
	errString := fmt.Sprintf("%v", err)

	c.JSON(http.StatusOK, gin.H{"status": errString, "code": code})

}

func updateDevice(c *gin.Context) {

	var g types.Device
	c.BindJSON(&g)

	code, err := simulatorController.UpdateDevice(g.Dev, g.Index)
	errString := fmt.Sprintf("%v", err)

	c.JSON(http.StatusOK, gin.H{"status": errString, "code": code})

}

func deleteDevice(c *gin.Context) {

	g := struct {
		DevEUI lorawan.EUI64
	}{}

	c.BindJSON(&g)

	c.JSON(http.StatusOK, gin.H{"status": simulatorController.DeleteDevice(g.DevEUI)})
}

func newServerSocket() *socketio.Server {

	serverSocket, _ := socketio.NewServer(nil)

	serverSocket.OnConnect("/", func(s socketio.Conn) error {

		s.SetContext("")
		simulatorController.Setup(s)

		log.Println("[WS]: Socket connected")

		return nil
	})

	serverSocket.OnDisconnect("/", func(s socketio.Conn, reason string) {
		s.Close()
	})

	serverSocket.OnEvent("/", socket.EventTurnOnDevice, func(s socketio.Conn, DevEUI string) (string, bool) {

		var paramDevEUI lorawan.EUI64
		DevEUITmp, _ := hex.DecodeString(DevEUI)
		copy(paramDevEUI[:8], DevEUITmp)

		return DevEUI, simulatorController.TurnONDevice(paramDevEUI)

	})

	serverSocket.OnEvent("/", socket.EventTurnOffDevice, func(s socketio.Conn, DevEUI string) (string, bool) {

		var paramDevEUI lorawan.EUI64
		DevEUITmp, _ := hex.DecodeString(DevEUI)
		copy(paramDevEUI[:8], DevEUITmp)

		return DevEUI, simulatorController.TurnOFFDevice(paramDevEUI)

	})

	serverSocket.OnEvent("/", socket.EventTurnOnGateway, func(s socketio.Conn, MACaddress string) (string, bool) {
		var Mac lorawan.EUI64
		MACtmp, _ := hex.DecodeString(MACaddress)
		copy(Mac[:8], MACtmp)

		return MACaddress, simulatorController.TurnONGateway(Mac)

	})

	serverSocket.OnEvent("/", socket.EventTurnOffGateway, func(s socketio.Conn, MACaddress string) (string, bool) {

		var Mac lorawan.EUI64
		MACtmp, _ := hex.DecodeString(MACaddress)
		copy(Mac[:8], MACtmp)

		return MACaddress, simulatorController.TurnOFFGateway(Mac)

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

	serverSocket.OnEvent("/", socket.EventChangePayload, func(s socketio.Conn, data socket.NewPayload) {
		simulatorController.ChangePayload(data)
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

//Run a webserver
func (ws *WebServer) Run() {

	log.Println("[WS]: Listen [", ws.Address+":"+strconv.Itoa(ws.Port), "]")
	err := ws.Router.Run(ws.Address + ":" + strconv.Itoa(ws.Port))
	if err != nil {
		log.Println("[WS] [ERROR]:", err.Error())
	}

}

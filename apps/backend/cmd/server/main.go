package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"lwn-simulator-backend/internal/mcpserver"
	"lwn-simulator-backend/internal/server"
	"lwn-simulator-backend/internal/utils"
)

func main() {
	var port string
	var serveMCP, writableMCP bool

	flag.StringVar(&port, "port", "8080", "Port to start the server on")
	flag.StringVar(&port, "p", "8080", "Port to start the server on (short)")
	flag.BoolVar(&serveMCP, "mcp", false, "Run a local MCP server over standard input and output")
	flag.BoolVar(&writableMCP, "mcp-write", false, "Enable simulation control and traffic tools in MCP mode")
	flag.Parse()

	if serveMCP {
		agentServer, err := mcpserver.New(port, writableMCP)
		if err != nil {
			log.Fatal(err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := agentServer.Run(ctx, &mcp.StdioTransport{MaxLineLength: 64 << 10}); err != nil && ctx.Err() == nil {
			log.Fatal(err)
		}
		return
	}
	if writableMCP {
		log.Fatal("-mcp-write requires -mcp")
	}

	app, err := server.New(port)
	if err != nil {
		log.Fatal(err)
	}

	privateIP := utils.GetPrivateIP()

	address := fmt.Sprintf(":%s", port)

	log.Println("Server running:")
	log.Printf("  Local:   http://localhost:%s\n", port)
	log.Printf("  Network: http://%s:%s\n", privateIP, port)

	log.Fatal(app.Run(address))
}

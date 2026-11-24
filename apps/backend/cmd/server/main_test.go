package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPCommandOverStdio(t *testing.T) {
	const childMarker = "LWN_TEST_MCP_STDIO_CHILD"
	if os.Getenv(childMarker) == "1" {
		for index, argument := range os.Args {
			if argument == "--" {
				os.Args = append(os.Args[:1], os.Args[index+1:]...)
				main()
				os.Exit(0)
			}
		}
		os.Exit(1)
	}

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/profile/get-profiles" {
			http.NotFound(w, request)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"profiles":[{"id":"00000000-0000-4000-8000-000000000001","name":"Stdio network"}],"total":1}`)
	}))
	t.Cleanup(backend.Close)
	_, port, err := net.SplitHostPort(backend.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMCPCommandOverStdio$", "--", "-mcp", "-port", port)
	cmd.Env = append(os.Environ(), childMarker+"=1")
	cmd.Stderr = os.Stderr
	client := mcp.NewClient(&mcp.Implementation{Name: "lwn-stdio-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_profiles"})
	if err != nil || result == nil || result.IsError || result.StructuredContent == nil {
		t.Errorf("list_profiles over stdio failed: %+v (%v)", result, err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("close stdio session: %v", err)
	}
}

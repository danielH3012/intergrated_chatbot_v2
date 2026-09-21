package main

import (
	"github.com/mark3labs/mcp-go/server"

	"mcp_server/client"
	"mcp_server/controller"
)

func main() {
	client.InitLogger()

	s := server.NewMCPServer(
		"qtera-inventory-mcp",
		"2.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	// Register modular tool groups
	controller.RegisterAssetTools(s)
	controller.RegisterDocumentTools(s)
	controller.RegisterScheduleTools(s)
	controller.RegisterUtilityTools(s)

	if err := server.ServeStdio(s); err != nil {
		if client.Logger != nil {
			client.Logger.Fatalf("server error: %v", err)
		}
	}
}

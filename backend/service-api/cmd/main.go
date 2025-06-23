package main

import (
	"watch-party/pkg/config"
	"watch-party/pkg/logger"
	"watch-party/service-api/internal/app"
)

func main() {
	// Initialize configuration
	cfg := config.NewConfig()

	// Initialize logger
	logger.InitLogger(cfg)

	// Create and start the application server
	server := app.NewAppServer(cfg)
	server.Serve()
}

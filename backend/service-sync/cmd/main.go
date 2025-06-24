package main

import (
	"watch-party/pkg/config"
	"watch-party/pkg/logger"
	"watch-party/service-sync/internal/app"
)

func main() {
	// initialize configuration
	cfg := config.NewConfig()

	// initialize logger
	logger.InitLogger(cfg)

	// create and start the sync service
	server := app.NewSyncServer(cfg)
	server.Serve()
}

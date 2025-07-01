package main

import (
	"context"

	"watch-party/pkg/config"
	api "watch-party/service-api"
)

func startAPIService(ctx context.Context, cfg *config.Config) {
	app := api.NewAppServer(cfg)
	app.Serve()
}

package main

import (
	"context"
	"watch-party/pkg/config"
	sync "watch-party/service-sync"
)

func startSyncService(ctx context.Context, cfg *config.Config) {
	cp := *cfg
	cp.Port = "8081"
	app := sync.NewSyncServer(&cp)
	app.Serve()
}

package helper

import (
	"watch-party/pkg/config"
	"watch-party/service-sync/internal/app"
)

func NewSyncServer(
	cfg *config.Config,
) *app.AppServer {
	return app.NewAppServer(cfg)
}

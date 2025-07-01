package helper

import (
	"watch-party/pkg/config"
	"watch-party/service-api/internal/app"
)

func NewAppServer(
	cfg *config.Config,
) *app.AppServer {
	return app.NewAppServer(cfg)
}

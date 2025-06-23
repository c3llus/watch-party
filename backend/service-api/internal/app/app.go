package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"watch-party/pkg/config"
	"watch-party/pkg/database"
	"watch-party/pkg/logger"
	mdw "watch-party/service-api/internal/app/middleware"
	ctl "watch-party/service-api/internal/controller"
	authRepo "watch-party/service-api/internal/repository/auth"
	userRepo "watch-party/service-api/internal/repository/user"
	authService "watch-party/service-api/internal/service/auth"
	userService "watch-party/service-api/internal/service/user"
)

type appServer struct {
	config     *config.Config
	middleware mdw.MiddlewareProvider
	controller ctl.ControllerProvider
}

// NewAppServer creates a new instance of appServer with the provided configuration, middleware, and controller.
func NewAppServer(cfg *config.Config) *appServer {
	// initialize database
	db, err := database.NewPgDB(cfg)
	if err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}

	// initialize repositories
	userRepository := userRepo.NewRepository(db)
	authRepository := authRepo.NewRepository(db)

	// initialize services
	userSvc := userService.NewUserService(userRepository)
	authSvc := authService.NewAuthService(cfg, userSvc, authRepository)

	// initialize controller
	controller := ctl.NewController(authSvc)

	// initialize middleware
	middleware := mdw.NewMiddleware()

	return &appServer{
		config:     cfg,
		middleware: middleware,
		controller: controller,
	}
}

func (a *appServer) Serve() {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", a.config.Port),
		Handler: a.RegisterHandlers(),
	}

	// serve the server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed to start: %v", err)
		}
	}()

	logger.Infof("server started on port %s", a.config.Port)

	a.gracefulShutdown(server)

	logger.Info("server shutdown complete")
}

func (a *appServer) gracefulShutdown(server *http.Server) {
	ctx, stopCtx := context.WithCancel(context.Background())

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP) // wait for the sigterm
		<-signals

		// we received an os signal, shut down.
		if err := server.Shutdown(ctx); err != nil {
			logger.Error(err, "server shutdown error")
		} else {
			logger.Info("server graceful shutdown")
		}

		stopCtx()
	}()

	<-ctx.Done()
}

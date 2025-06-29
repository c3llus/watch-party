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
	"watch-party/pkg/email"
	"watch-party/pkg/events"
	"watch-party/pkg/logger"
	"watch-party/pkg/storage"
	"watch-party/pkg/video"
	mdw "watch-party/service-api/internal/app/middleware"
	ctl "watch-party/service-api/internal/controller"
	authRepo "watch-party/service-api/internal/repository/auth"
	movieRepo "watch-party/service-api/internal/repository/movie"
	roomRepo "watch-party/service-api/internal/repository/room"
	userRepo "watch-party/service-api/internal/repository/user"
	authService "watch-party/service-api/internal/service/auth"
	movieService "watch-party/service-api/internal/service/movie"
	roomService "watch-party/service-api/internal/service/room"
	userService "watch-party/service-api/internal/service/user"
)

type appServer struct {
	config                *config.Config
	middleware            mdw.MiddlewareProvider
	controller            ctl.ControllerProvider
	movieController       *ctl.MovieController
	roomController        *ctl.RoomController
	webhookController     *ctl.WebhookController
	streamingController   *ctl.StreamingController
	videoAccessController *ctl.VideoAccessController
	roomService           *roomService.Service
}

// NewAppServer creates a new instance of appServer with the provided configuration, middleware, and controller.
func NewAppServer(cfg *config.Config) *appServer {
	// initialize database
	db, err := database.NewPgDB(cfg)
	if err != nil {
		logger.Fatalf("failed to initialize database: %v", err)
	}

	// initialize storage provider
	storageProvider, err := storage.NewStorageProvider(context.Background(), &cfg.Storage)
	if err != nil {
		logger.Fatalf("failed to initialize storage provider: %v", err)
	}

	// initialize repositories
	userRepository := userRepo.NewRepository(db)
	authRepository := authRepo.NewRepository(db)
	movieRepository := movieRepo.NewRepository(db)
	roomRepository := roomRepo.NewRepository(db)

	// shared pkgs
	emailService, err := email.NewEmailProvider(context.Background(), &cfg.Email)
	if err != nil {
		logger.Fatalf("failed to initialize email provider: %v", err)
	}

	// initialize services
	userSvc := userService.NewUserService(userRepository)
	authSvc := authService.NewAuthService(cfg, userSvc, authRepository)
	movieSvc := movieService.NewMovieService(movieRepository, storageProvider)
	roomSvc := roomService.NewService(roomRepository, userRepository, emailService, cfg)

	// initialize event handler dependencies
	tempDir := cfg.Storage.VideoProcessing.TempDir
	hlsBaseURL := cfg.Storage.VideoProcessing.HLSBaseURL

	// create video processor
	videoProcessor := video.NewProcessor(storageProvider, tempDir)

	// create upload event handler
	uploadHandler := events.NewHandler(movieRepository, storageProvider, videoProcessor, hlsBaseURL, tempDir)

	// initialize controllers
	controller := ctl.NewController(authSvc)
	movieController := ctl.NewMovieController(movieSvc)
	roomController := ctl.NewRoomController(roomSvc)
	webhookController := ctl.NewWebhookController(uploadHandler)
	streamingController := ctl.NewStreamingController(storageProvider, movieSvc, roomSvc)
	videoAccessController := ctl.NewVideoAccessController(storageProvider, movieSvc, roomSvc)

	// initialize middleware
	middleware := mdw.NewMiddleware()

	return &appServer{
		config:                cfg,
		middleware:            middleware,
		controller:            controller,
		movieController:       movieController,
		roomController:        roomController,
		webhookController:     webhookController,
		streamingController:   streamingController,
		videoAccessController: videoAccessController,
		roomService:           roomSvc,
	}
}

func (a *appServer) Serve() {
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", a.config.Port),
		Handler: a.RegisterHandlers(),
	}

	// serve the server
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
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
		err := server.Shutdown(ctx)
		if err != nil {
			logger.Error(err, "server shutdown error")
		} else {
			logger.Info("server graceful shutdown")
		}

		stopCtx()
	}()

	<-ctx.Done()
}

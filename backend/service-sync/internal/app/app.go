package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"watch-party/pkg/auth"
	"watch-party/pkg/config"
	"watch-party/pkg/logger"
	"watch-party/pkg/redis"
	"watch-party/service-sync/internal/handler"
	"watch-party/service-sync/internal/repository"
	"watch-party/service-sync/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type AppServer struct {
	config      *config.Config
	handler     *handler.SyncHandler
	redisClient *redis.Client
}

// NewAppServer creates a new sync server instance
func NewAppServer(cfg *config.Config) *AppServer {
	// service-sync only needs Redis for real-time state management
	// room validation will be done via HTTP calls to service-api

	// initialize Redis client
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		logger.Fatalf("failed to initialize Redis client: %v", err)
	}

	// initialize sync repository (Redis-based for real-time sync state)
	syncRepo := repository.NewSyncRepository(redisClient)

	// initialize service
	syncService := service.NewSyncService(syncRepo, redisClient)

	// initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret)

	// initialize handler
	syncHandler := handler.NewSyncHandler(syncService, jwtManager)

	return &AppServer{
		config:      cfg,
		handler:     syncHandler,
		redisClient: redisClient,
	}
}

// Serve starts the sync server
func (s *AppServer) Serve() {
	// setup gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// cors middleware
	corsConfig := cors.Config{
		AllowOrigins:     s.config.CORS.AllowedOrigins,
		AllowMethods:     s.config.CORS.AllowedMethods,
		AllowHeaders:     s.config.CORS.AllowedHeaders,
		AllowCredentials: true,
	}
	router.Use(cors.New(corsConfig))

	// setup routes
	s.setupRoutes(router)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", s.config.Port),
		Handler: router,
	}

	sslEnabled := os.Getenv("SSL_ENABLED") == "true"
	certPath := os.Getenv("SSL_CERT_PATH")
	keyPath := os.Getenv("SSL_KEY_PATH")

	// start server
	go func() {
		var err error
		if sslEnabled && certPath != "" && keyPath != "" {
			logger.Infof("Starting SSL server on port %s", s.config.Port)
			err = server.ListenAndServeTLS(certPath, keyPath)
		} else {
			logger.Infof("Starting HTTP server on port %s", s.config.Port)
			err = server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("sync server failed to start: %v", err)
		}
	}()

	if sslEnabled {
		httpRouter := gin.New()
		httpRouter.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "sync"})
		})

		httpServer := &http.Server{
			Addr:    ":8080",
			Handler: httpRouter,
		}

		go func() {
			logger.Info("Starting HTTP health check server on port 8080")
			err := httpServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				logger.Error(err, "HTTP health server failed")
			}
		}()
	}

	logger.Infof("sync server started on port %s (SSL: %v)", s.getSyncPort(), sslEnabled)

	s.gracefulShutdown(server)

	logger.Info("sync server shutdown complete")
}

// setupRoutes configures the server routes
func (s *AppServer) setupRoutes(router *gin.Engine) {
	// websocket endpoint for room synchronization
	router.GET("/ws/room/:roomID", s.handler.HandleWebSocket)

	// read-only endpoints for sync state (Redis-based)
	api := router.Group("/api/v1")
	{
		// room sync state queries (read-only, from Redis)
		api.GET("/rooms/:roomID/state", s.handler.GetRoomState)
		api.GET("/rooms/:roomID/participants", s.handler.GetRoomParticipants)
	}

	// health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "sync"})
	})
}

// getSyncPort returns the port for the sync service
func (s *AppServer) getSyncPort() string {
	// use different port for sync service, default to 8081
	if port := os.Getenv("SYNC_PORT"); port != "" {
		return port
	}
	return "8081"
}

// gracefulShutdown handles graceful server shutdown
func (s *AppServer) gracefulShutdown(server *http.Server) {
	ctx, stopCtx := context.WithCancel(context.Background())

	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		<-signals

		// close Redis connection
		if s.redisClient != nil {
			s.redisClient.Close()
		}

		// shutdown server
		err := server.Shutdown(ctx)
		if err != nil {
			logger.Error(err, "sync server shutdown error")
		} else {
			logger.Info("sync server graceful shutdown")
		}

		stopCtx()
	}()

	<-ctx.Done()
}

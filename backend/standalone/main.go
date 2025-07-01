package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"watch-party/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {

	// Create centralized configuration
	cfg := createEmbeddedConfig()

	logger.InitLogger(cfg)

	logger.Info("🚀 Starting Watch Party Standalone Application...")
	logger.Info("This includes: PostgreSQL 17, Redis 7, MinIO, API Service, Sync Service, and React Frontend")

	// Initialize embedded services
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// Start embedded services first and wait for them to be ready
	logger.Info("🔧 Starting embedded services...")

	// Start embedded PostgreSQL
	wg.Add(1)
	go func() {
		defer wg.Done()
		startEmbeddedDB(ctx)
	}()

	// Start embedded Redis
	wg.Add(1)
	go func() {
		defer wg.Done()
		startEmbeddedRedis(ctx)
	}()

	// Start embedded MinIO
	wg.Add(1)
	go func() {
		defer wg.Done()
		startEmbeddedMinio(ctx)
	}()

	// Wait for embedded services to be ready
	logger.Info("⏳ Waiting for all embedded services to be ready...")

	// Wait for services to be ready with proper health checks
	waitForEmbeddedServicesToBeReady()

	// Update config with actual embedded service addresses
	updateConfigWithEmbeddedServices(cfg)

	logger.Info("🚀 Starting application services...")

	// Start API service with config
	wg.Add(1)
	go func() {
		defer wg.Done()
		startAPIService(ctx, cfg)
	}()

	// Start Sync service with config
	wg.Add(1)
	go func() {
		defer wg.Done()
		startSyncService(ctx, cfg)
	}()

	// Start frontend server
	wg.Add(1)
	go func() {
		defer wg.Done()
		startFrontendServer(ctx)
	}()

	// Setup graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	logger.Info("Shutting down...")
	cancel()
	wg.Wait()
	logger.Info("Shutdown complete")
}

func startFrontendServer(ctx context.Context) {
	router := gin.Default()

	// Setup frontend routes using the embedded frontend
	err := setupFrontendRoutes(router)
	if err != nil {
		logger.Error(err, "Failed to setup frontend routes")
		return
	}

	server := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	logger.Info("Frontend server running on http://localhost:3000")
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		logger.Error(err, "Frontend server error")
	}
}

// waitForEmbeddedServicesToBeReady waits for all embedded services to be ready
func waitForEmbeddedServicesToBeReady() {
	waitForPostgreSQLReady()
	waitForRedisReady()
	waitForMinIOServiceReady()
	logger.Info("✅ All embedded services are ready!")
}

// waitForPostgreSQLReady waits for PostgreSQL to be ready
func waitForPostgreSQLReady() {
	logger.Info("Waiting for PostgreSQL to be ready...")
	for i := 0; i < 60; i++ { // wait up to 60 seconds
		if GetDBConnection() != nil {
			err := GetDBConnection().Ping()
			if err == nil {
				logger.Info("✅ PostgreSQL is ready")
				break
			}
		}
		if i == 59 {
			logger.Error(nil, "PostgreSQL failed to become ready within 60 seconds")
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// waitForRedisReady waits for Redis to be ready
func waitForRedisReady() {
	logger.Info("Waiting for Redis to be ready...")
	for i := 0; i < 30; i++ { // wait up to 30 seconds
		if GetRedisAddr() != "" {
			logger.Info("✅ Redis is ready")
			break
		}
		if i == 29 {
			logger.Error(nil, "Redis failed to become ready within 30 seconds")
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// waitForMinIOServiceReady waits for MinIO to be ready
func waitForMinIOServiceReady() {
	logger.Info("Waiting for MinIO to be ready...")
	for i := 0; i < 30; i++ { // wait up to 30 seconds
		if GetMinioEndpoint() != "" {
			logger.Info("✅ MinIO is ready")
			break
		}
		if i == 29 {
			logger.Error(nil, "MinIO failed to become ready within 30 seconds")
			return
		}
		time.Sleep(1 * time.Second)
	}
}

package main

import (
	"context"
	"fmt"
	"log"

	"watch-party/pkg/logger"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

var (
	embeddedRedis *miniredis.Miniredis
	redisClient   *redis.Client
)

func startEmbeddedRedis(ctx context.Context) {
	logger.Info("Starting embedded Redis 7...")

	// create embedded Redis instance
	var err error
	embeddedRedis, err = miniredis.Run()
	if err != nil {
		log.Fatalf("Failed to start embedded Redis: %v", err)
	}

	// create Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr: embeddedRedis.Addr(),
		DB:   0,
	})

	// test connection
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to ping embedded Redis: %v", err)
	}

	logger.Info(fmt.Sprintf("✅ Embedded Redis started successfully on %s", embeddedRedis.Addr()))

	// wait for context cancellation
	<-ctx.Done()

	// shutdown
	logger.Info("Shutting down embedded Redis...")
	if redisClient != nil {
		redisClient.Close()
	}
	if embeddedRedis != nil {
		embeddedRedis.Close()
	}
}

// GetRedisClient returns the Redis client for use by services
func GetRedisClient() *redis.Client {
	return redisClient
}

// GetRedisAddr returns the address of the embedded Redis instance
func GetRedisAddr() string {
	if embeddedRedis != nil {
		return embeddedRedis.Addr()
	}
	return ""
}

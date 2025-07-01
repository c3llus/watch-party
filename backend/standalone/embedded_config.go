package main

import (
	"fmt"
	"strings"
	"time"
	"watch-party/pkg/config"
)

// createEmbeddedConfig creates a hardcoded configuration for the standalone application
func createEmbeddedConfig() *config.Config {
	return &config.Config{
		Port:      "8080",
		JWTSecret: "embedded-jwt-secret-key-change-in-production",
		Database: config.DatabaseConfig{
			Name:            "watchparty",
			Host:            "localhost",
			Port:            "15432", // Updated to match embedded DB port
			Username:        "postgres",
			Password:        "postgres",
			Database:        "watchparty",
			MaxOpenConns:    25,
			MaxIdleConns:    25,
			ConnMaxLifetime: config.Duration(5 * time.Minute),
			SSLMode:         "disable",
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     "6379",
			Password: "",
			DB:       0,
		},
		Storage: config.StorageConfig{
			Provider: "minio",
			MinIO: config.MinIOConfig{
				Endpoint:       "localhost:19000", // Updated to avoid conflicts
				AccessKey:      "minioadmin",
				SecretKey:      "minioadmin",
				Bucket:         "watch-party-videos",
				UseSSL:         false,
				PublicEndpoint: "", // Will be set dynamically
			},
			VideoProcessing: config.VideoConfig{
				TempDir:     "./temp",
				HLSBaseURL:  "http://localhost:8080/api/v1/files",
				FFmpegPath:  "ffmpeg",
				FFprobePath: "ffprobe",
			},
		},
		Email: config.EmailConfig{
			Provider: "noop",
			SMTP: config.SMTPConfig{
				Host:     "localhost",
				Port:     587,
				Username: "gg",
				Password: "gg",
				UseTLS:   false,
			},
			SendGrid: config.SendGridConfig{
				FromEmail: "noreply@watch-party.local",
				FromName:  "Watch Party",
			},
			Templates: config.EmailTemplateConfig{
				BaseURL: "http://localhost:3000",
				AppName: "Watch Party",
			},
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080", "*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"*"},
		},
	}
}

// updateConfigWithEmbeddedServices updates the config with actual embedded service addresses
func updateConfigWithEmbeddedServices(cfg *config.Config) {
	// update Redis address if embedded Redis is running
	redisAddr := GetRedisAddr()
	if redisAddr != "" {
		// extract host and port from address like "127.0.0.1:12345"
		parts := strings.Split(redisAddr, ":")
		if len(parts) == 2 {
			cfg.Redis.Host = parts[0]
			cfg.Redis.Port = parts[1]
		}
	}

	// ensure database connection is available
	if GetDBConnection() != nil {
		cfg.Database.Host = "localhost"
		cfg.Database.Port = fmt.Sprintf("%d", GetDBPort())
	}
}

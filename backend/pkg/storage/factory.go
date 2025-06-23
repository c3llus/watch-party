package storage

import (
	"context"
	"fmt"
	"watch-party/pkg/config"
)

// Storage provider constants
const (
	StorageProviderLocal = "local"
	StorageProviderGCS   = "gcs"
)

// NewStorageProvider creates a storage provider based on configuration
func NewStorageProvider(ctx context.Context, cfg *config.StorageConfig) (Provider, error) {
	switch cfg.Provider {
	case StorageProviderLocal:
		baseURL := "http://localhost:8080/api/v1/files" // Default base URL for serving files
		return NewLocalProvider(cfg.LocalPath, baseURL), nil

	case StorageProviderGCS:
		if cfg.GCSBucket == "" {
			return nil, fmt.Errorf("GCS bucket name is required")
		}
		return NewGCSProvider(ctx, cfg.GCSBucket)

	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.Provider)
	}
}

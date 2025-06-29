package storage

import (
	"context"
	"fmt"
	"watch-party/pkg/config"
)

// Storage provider constants
const (
	StorageProviderGCS   = "gcs"
	StorageProviderMinIO = "minio"
)

// NewStorageProvider creates a storage provider based on configuration
func NewStorageProvider(ctx context.Context, cfg *config.StorageConfig) (Provider, error) {
	switch cfg.Provider {
	case StorageProviderGCS:
		if cfg.GCSBucket == "" {
			return nil, fmt.Errorf("GCS bucket name is required")
		}
		return NewGCSProvider(ctx, cfg.GCSBucket)

	case StorageProviderMinIO:
		if cfg.MinIO.Endpoint == "" {
			return nil, fmt.Errorf("MinIO endpoint is required")
		}
		if cfg.MinIO.Bucket == "" {
			return nil, fmt.Errorf("MinIO bucket name is required")
		}
		return NewMinIOProvider(
			cfg.MinIO.Endpoint,
			cfg.MinIO.AccessKey,
			cfg.MinIO.SecretKey,
			cfg.MinIO.Bucket,
			cfg.MinIO.UseSSL,
			cfg.MinIO.PublicEndpoint,
		)

	}

	return nil, fmt.Errorf("unsupported storage provider: %s. Supported providers: gcs, minio", cfg.Provider)

}

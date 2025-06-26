package storage

import (
	"context"
	"mime/multipart"
	"time"
)

// Provider defines the interface for storage providers
type Provider interface {
	Upload(ctx context.Context, file *multipart.FileHeader, filename string) (string, error)
	UploadFromPath(ctx context.Context, localPath, storagePath string) error
	GenerateSignedUploadURL(ctx context.Context, filename string, opts *UploadOptions) (*SignedURL, error)
	GetSignedURL(ctx context.Context, path string) (string, error)
	Download(ctx context.Context, storagePath, localPath string) error
	Delete(ctx context.Context, path string) error
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
	GetPublicURL(ctx context.Context, path string) (string, error)
	ListObjects(ctx context.Context, prefix string) ([]string, error)
	GenerateSignedURLs(ctx context.Context, paths []string, opts *CDNSignedURLOptions) (map[string]string, error)
	GenerateCDNSignedURL(ctx context.Context, path string, opts *CDNSignedURLOptions) (string, error)
}

// SignedURL represents a signed URL for upload
type SignedURL struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`                // http method (PUT, POST)
	Headers    map[string]string `json:"headers"`               // required headers
	FormFields map[string]string `json:"form_fields,omitempty"` // for POST-based uploads
	ExpiresAt  time.Time         `json:"expires_at"`
}

// FileInfo represents basic file information
type FileInfo struct {
	Name         string
	Size         int64
	ContentType  string
	LastModified string
	URL          string
}

// UploadOptions represents options for file upload
type UploadOptions struct {
	ContentType string
	MaxFileSize int64
	ExpiresIn   time.Duration // Duration for signed URL validity
	Public      bool
}

// CDNSignedURLOptions represents options for CDN-friendly signed URLs
type CDNSignedURLOptions struct {
	ExpiresIn    time.Duration // Duration for URL validity
	CacheControl string        // Cache-Control header value
	Organization string        // Organization scope for multi-tenant access
	ContentType  string        // Override content type
}

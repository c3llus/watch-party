package storage

import (
	"context"
	"mime/multipart"
	"time"
)

// Provider defines the interface for storage providers
type Provider interface {
	// Upload uploads a file and returns the storage path
	Upload(ctx context.Context, file *multipart.FileHeader, filename string) (string, error)

	// UploadFromPath uploads a file from local filesystem to storage
	UploadFromPath(ctx context.Context, localPath, storagePath string) error

	// GenerateSignedUploadURL generates a signed URL for client-side upload
	GenerateSignedUploadURL(ctx context.Context, filename string, opts *UploadOptions) (*SignedURL, error)

	// GetSignedURL returns a signed URL for accessing the file
	GetSignedURL(ctx context.Context, path string) (string, error)

	// Download downloads a file from storage to a local file
	Download(ctx context.Context, storagePath, localPath string) error

	// Delete deletes a file from storage
	Delete(ctx context.Context, path string) error

	// GetFileInfo returns basic information about a file
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)

	// GetPublicURL returns a public URL for the file (for HLS playlists)
	GetPublicURL(ctx context.Context, path string) (string, error)

	// ListObjects lists objects in a directory/prefix
	ListObjects(ctx context.Context, prefix string) ([]string, error)
}

// SignedURL represents a signed URL for upload
type SignedURL struct {
	URL        string            `json:"url"`
	Method     string            `json:"method"`                // HTTP method (PUT, POST)
	Headers    map[string]string `json:"headers"`               // Required headers
	FormFields map[string]string `json:"form_fields,omitempty"` // For POST-based uploads
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

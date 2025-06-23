package storage

import (
	"context"
	"mime/multipart"
)

// Provider defines the interface for storage providers
type Provider interface {
	// Upload uploads a file and returns the storage path
	Upload(ctx context.Context, file *multipart.FileHeader, filename string) (string, error)

	// GetSignedURL returns a signed URL for accessing the file
	GetSignedURL(ctx context.Context, path string) (string, error)

	// Delete deletes a file from storage
	Delete(ctx context.Context, path string) error

	// GetFileInfo returns basic information about a file
	GetFileInfo(ctx context.Context, path string) (*FileInfo, error)
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
	Public      bool
}

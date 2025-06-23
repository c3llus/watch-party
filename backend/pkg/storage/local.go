package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// LocalProvider implements storage for local filesystem
type LocalProvider struct {
	basePath string
	baseURL  string // For serving files via HTTP
}

// NewLocalProvider creates a new local storage provider
func NewLocalProvider(basePath, baseURL string) *LocalProvider {
	// Ensure the base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create storage directory: %v", err))
	}

	return &LocalProvider{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// Upload uploads a file to the local filesystem
func (l *LocalProvider) Upload(ctx context.Context, file *multipart.FileHeader, filename string) (string, error) {
	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Create the full path
	fullPath := filepath.Join(l.basePath, filename)

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Create the destination file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// Copy the file content
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Return the relative path
	return filename, nil
}

// GetSignedURL returns a URL for accessing the file
func (l *LocalProvider) GetSignedURL(ctx context.Context, path string) (string, error) {
	// For local storage, we return a direct URL
	return fmt.Sprintf("%s/%s", l.baseURL, path), nil
}

// Delete deletes a file from the local filesystem
func (l *LocalProvider) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(l.basePath, path)
	return os.Remove(fullPath)
}

// GetFileInfo returns information about a file
func (l *LocalProvider) GetFileInfo(ctx context.Context, path string) (*FileInfo, error) {
	fullPath := filepath.Join(l.basePath, path)

	stat, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	url, _ := l.GetSignedURL(ctx, path)

	return &FileInfo{
		Name:         stat.Name(),
		Size:         stat.Size(),
		ContentType:  getContentType(path),
		LastModified: stat.ModTime().Format(time.RFC3339),
		URL:          url,
	}, nil
}

// getContentType returns the MIME type based on file extension
func getContentType(filename string) string {
	ext := filepath.Ext(filename)
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".avi":
		return "video/x-msvideo"
	case ".mkv":
		return "video/x-matroska"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

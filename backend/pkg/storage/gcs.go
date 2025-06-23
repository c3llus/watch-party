package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"cloud.google.com/go/storage"
)

// GCSProvider implements storage for Google Cloud Storage
type GCSProvider struct {
	client *storage.Client
	bucket string
}

// NewGCSProvider creates a new GCS storage provider
func NewGCSProvider(ctx context.Context, bucketName string) (*GCSProvider, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return &GCSProvider{
		client: client,
		bucket: bucketName,
	}, nil
}

// Upload uploads a file to Google Cloud Storage
func (g *GCSProvider) Upload(ctx context.Context, file *multipart.FileHeader, filename string) (string, error) {
	// open the uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// get a reference to the GCS object
	obj := g.client.Bucket(g.bucket).Object(filename)

	// create a writer to the GCS object
	writer := obj.NewWriter(ctx)
	writer.ContentType = file.Header.Get("Content-Type")
	if writer.ContentType == "" {
		writer.ContentType = getContentType(filename)
	}

	// copy the file to GCS
	_, err = io.Copy(writer, src)
	if err != nil {
		writer.Close()
		return "", fmt.Errorf("failed to copy file to GCS: %w", err)
	}

	// close the writer to finalize the upload
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close GCS writer: %w", err)
	}

	return filename, nil
}

// GetSignedURL returns a signed URL for accessing the file
func (g *GCSProvider) GetSignedURL(ctx context.Context, path string) (string, error) {
	// generate a signed URL valid for 1 hour
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(time.Hour),
	}

	url, err := storage.SignedURL(g.bucket, path, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// Delete deletes a file from Google Cloud Storage
func (g *GCSProvider) Delete(ctx context.Context, path string) error {
	obj := g.client.Bucket(g.bucket).Object(path)
	err := obj.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete object from GCS: %w", err)
	}
	return nil
}

// GetFileInfo returns information about a file in GCS
func (g *GCSProvider) GetFileInfo(ctx context.Context, path string) (*FileInfo, error) {
	obj := g.client.Bucket(g.bucket).Object(path)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	url, _ := g.GetSignedURL(ctx, path)

	return &FileInfo{
		Name:         attrs.Name,
		Size:         attrs.Size,
		ContentType:  attrs.ContentType,
		LastModified: attrs.Updated.Format(time.RFC3339),
		URL:          url,
	}, nil
}

// Close closes the GCS client
func (g *GCSProvider) Close() error {
	return g.client.Close()
}

package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
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

// UploadFromPath uploads a file from local filesystem to GCS
func (g *GCSProvider) UploadFromPath(ctx context.Context, localPath, storagePath string) error {
	// Open the local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get a reference to the GCS object
	obj := g.client.Bucket(g.bucket).Object(storagePath)

	// Create a writer to the GCS object
	writer := obj.NewWriter(ctx)
	writer.ContentType = getContentType(localPath)

	// Copy the file to GCS
	_, err = io.Copy(writer, file)
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to copy file to GCS: %w", err)
	}

	// Close the writer to finalize the upload
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close GCS writer: %w", err)
	}

	return nil
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

// GenerateSignedUploadURL generates a signed URL for uploading to GCS
func (g *GCSProvider) GenerateSignedUploadURL(ctx context.Context, filename string, opts *UploadOptions) (*SignedURL, error) {
	if opts == nil {
		opts = &UploadOptions{
			ExpiresIn: time.Hour,
		}
	}

	// set default expiration if not provided
	if opts.ExpiresIn == 0 {
		opts.ExpiresIn = time.Hour
	}

	// Get the bucket handle
	bucket := g.client.Bucket(g.bucket)

	// Generate signed URL for PUT method
	opts2 := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		Expires:        time.Now().Add(opts.ExpiresIn),
		GoogleAccessID: "", // This would need to be configured based on service account
	}

	// Set content type if provided
	if opts.ContentType != "" {
		opts2.ContentType = opts.ContentType
	}

	url, err := bucket.SignedURL(filename, opts2)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GCS signed URL: %w", err)
	}

	headers := make(map[string]string)
	if opts.ContentType != "" {
		headers["Content-Type"] = opts.ContentType
	}

	return &SignedURL{
		URL:       url,
		Method:    "PUT",
		Headers:   headers,
		ExpiresAt: time.Now().Add(opts.ExpiresIn),
	}, nil
}

// GetPublicURL returns a public URL for the file in GCS
func (g *GCSProvider) GetPublicURL(ctx context.Context, path string) (string, error) {
	// for public files in GCS, the URL format is:
	// https://storage.googleapis.com/{bucket}/{object}
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", g.bucket, path), nil
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

// ListObjects lists objects with a given prefix in GCS
func (g *GCSProvider) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	var objects []string

	it := g.client.Bucket(g.bucket).Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	for {
		attrs, err := it.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		objects = append(objects, attrs.Name)
	}

	return objects, nil
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

// Download downloads a file from GCS to local filesystem
func (g *GCSProvider) Download(ctx context.Context, storagePath, localPath string) error {
	// Get object from GCS
	obj := g.client.Bucket(g.bucket).Object(storagePath)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to get object reader from GCS: %w", err)
	}
	defer reader.Close()

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Copy data from GCS reader to local file
	_, err = io.Copy(localFile, reader)
	if err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	return nil
}

// Close closes the GCS client
func (g *GCSProvider) Close() error {
	return g.client.Close()
}

// getContentType returns the MIME type based on file name
func getContentType(filename string) string {
	if strings.HasSuffix(filename, ".m3u8") {
		return "application/x-mpegURL"
	}
	if strings.HasSuffix(filename, ".ts") {
		return "video/MP2T"
	}
	if strings.HasSuffix(filename, ".mp4") {
		return "video/mp4"
	}
	if strings.HasSuffix(filename, ".webm") {
		return "video/webm"
	}
	return "application/octet-stream"
}

// GenerateCDNSignedURL generates a CDN-friendly signed URL with custom options
func (g *GCSProvider) GenerateCDNSignedURL(ctx context.Context, path string, opts *CDNSignedURLOptions) (string, error) {
	// TODO: real CDN support
	if opts == nil {
		opts = &CDNSignedURLOptions{
			ExpiresIn: time.Hour * 2,
		}
	}
	// Set default expiration if not provided
	expiration := opts.ExpiresIn
	if expiration == 0 {
		expiration = time.Hour * 2
	}

	// Set up signed URL options
	signOpts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	}

	// Add response headers for CDN optimization
	if opts.CacheControl != "" || opts.ContentType != "" {
		signOpts.QueryParameters = make(map[string][]string)
		if opts.CacheControl != "" {
			signOpts.QueryParameters["response-cache-control"] = []string{opts.CacheControl}
		}
		if opts.ContentType != "" {
			signOpts.QueryParameters["response-content-type"] = []string{opts.ContentType}
		}
	}

	// Generate signed URL using storage.SignedURL function
	signedURL, err := storage.SignedURL(g.bucket, path, signOpts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return signedURL, nil
}

// GenerateSignedURLs generates signed URLs for multiple files (CDN-friendly)
func (g *GCSProvider) GenerateSignedURLs(ctx context.Context, paths []string, opts *CDNSignedURLOptions) (map[string]string, error) {
	if opts == nil {
		opts = &CDNSignedURLOptions{
			ExpiresIn: time.Hour * 2,
		}
	}

	result := make(map[string]string)

	// Generate URLs for each path
	for _, path := range paths {
		signedURL, err := g.GenerateCDNSignedURL(ctx, path, opts)
		if err != nil {
			// Continue with other URLs but log the error
			continue
		}
		result[path] = signedURL
	}

	// Return error if no URLs were generated successfully
	if len(result) == 0 {
		return nil, fmt.Errorf("failed to generate any signed URLs")
	}

	return result, nil
}

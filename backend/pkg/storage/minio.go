package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"
	"watch-party/pkg/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// minioProvider implements the Provider interface using MinIO
type minioProvider struct {
	client         *minio.Client
	bucket         string
	endpoint       string
	publicClient   *minio.Client // Client configured with public endpoint for signing URLs
	publicEndpoint string        // Public endpoint for generating URLs accessible from browser
	useSSL         bool
}

// NewMinIOProvider creates a new MinIO storage provider
func NewMinIOProvider(endpoint, accessKey, secretKey, bucket string, useSSL bool, publicEndpoint string) (Provider, error) {
	logger.Info(fmt.Sprintf("Creating MinIO provider with endpoint: %s, publicEndpoint: %s, useSSL: %v", endpoint, publicEndpoint, useSSL))

	// If publicEndpoint is empty, use the same as endpoint
	if publicEndpoint == "" {
		publicEndpoint = endpoint
		logger.Info(fmt.Sprintf("PublicEndpoint was empty, setting to: %s", publicEndpoint))
	}

	// create MinIO client for internal operations
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}
	logger.Info(fmt.Sprintf("MinIO client created successfully for endpoint: %s", endpoint))

	if client.IsOffline() {
		return nil, fmt.Errorf("MinIO client is not online at %s", endpoint)
	}

	publicClient, err := minio.New(publicEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create public MinIO client: %w", err)
	}
	logger.Info(fmt.Sprintf("MinIO public client created successfully for endpoint: %s", publicEndpoint))

	if publicClient.IsOffline() {
		return nil, fmt.Errorf("public MinIO client is not online at %s", publicEndpoint)
	}

	provider := &minioProvider{
		client:         client,
		bucket:         bucket,
		endpoint:       endpoint,
		publicClient:   publicClient,
		publicEndpoint: publicEndpoint,
		useSSL:         useSSL,
	}

	logger.Info(fmt.Sprintf("MinIO provider created, checking bucket: %s", bucket))
	// ensure bucket exists
	err = provider.ensureBucket(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	logger.Info("MinIO provider initialized successfully")
	return provider, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (m *minioProvider) ensureBucket(ctx context.Context) error {
	logger.Info(fmt.Sprintf("Checking if bucket exists: %s", m.bucket))
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil {
		logger.Info(fmt.Sprintf("Error checking bucket existence: %v", err))
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	logger.Info(fmt.Sprintf("Bucket exists: %v", exists))
	if !exists {
		logger.Info(fmt.Sprintf("Creating bucket: %s", m.bucket))
		err = m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{})
		if err != nil {
			logger.Info(fmt.Sprintf("Error creating bucket: %v", err))
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info(fmt.Sprintf("Bucket created successfully: %s", m.bucket))
	}

	return nil
}

// Upload uploads a file to MinIO
func (m *minioProvider) Upload(ctx context.Context, file *multipart.FileHeader, filename string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// determine content type
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// upload file
	_, err = m.client.PutObject(ctx, m.bucket, filename, src, file.Size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to MinIO: %w", err)
	}

	return filename, nil
}

// GenerateSignedUploadURL generates a presigned URL for uploading
func (m *minioProvider) GenerateSignedUploadURL(ctx context.Context, filename string, opts *UploadOptions) (*SignedURL, error) {
	if opts == nil {
		opts = &UploadOptions{
			ExpiresIn: time.Hour,
		}
	}

	// set default expiration if not provided
	if opts.ExpiresIn == 0 {
		opts.ExpiresIn = time.Hour
	}

	presignedURL, err := m.publicClient.PresignedPutObject(ctx, m.bucket, filename, opts.ExpiresIn)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	headers := make(map[string]string)
	if opts.ContentType != "" {
		headers["Content-Type"] = opts.ContentType
	}

	return &SignedURL{
		URL:       presignedURL.String(),
		Method:    "PUT",
		Headers:   headers,
		ExpiresAt: time.Now().Add(opts.ExpiresIn),
	}, nil
}

// GetSignedURL returns a presigned URL for accessing a file
func (m *minioProvider) GetSignedURL(ctx context.Context, path string) (string, error) {
	presignedURL, err := m.publicClient.PresignedGetObject(ctx, m.bucket, path, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return presignedURL.String(), nil
}

// Delete deletes a file from MinIO
func (m *minioProvider) Delete(ctx context.Context, path string) error {
	err := m.client.RemoveObject(ctx, m.bucket, path, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file from MinIO: %w", err)
	}

	return nil
}

// GetFileInfo returns information about a file
func (m *minioProvider) GetFileInfo(ctx context.Context, path string) (*FileInfo, error) {
	stat, err := m.client.StatObject(ctx, m.bucket, path, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// generate a temporary URL for the file
	url, err := m.GetSignedURL(ctx, path)
	if err != nil {
		url = "" // not critical if we can't generate URL
	}

	return &FileInfo{
		Name:         stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		LastModified: stat.LastModified.Format(time.RFC3339),
		URL:          url,
	}, nil
}

// GetPublicURL returns a public URL for the file (for HLS playlists)
func (m *minioProvider) GetPublicURL(ctx context.Context, path string) (string, error) {
	// for MinIO, we'll use the direct endpoint URL
	protocol := "http"
	if m.useSSL {
		protocol = "https"
	}

	// construct public URL
	publicURL := fmt.Sprintf("%s://%s/%s/%s", protocol, m.endpoint, m.bucket, path)
	return publicURL, nil
}

// ListObjects lists objects with a given prefix
func (m *minioProvider) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	var objects []string

	// list objects with prefix
	objectCh := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("error listing objects: %w", object.Err)
		}
		objects = append(objects, object.Key)
	}

	return objects, nil
}

// UploadFile uploads a file from a local path (helper method for transcoded files)
func (m *minioProvider) UploadFile(ctx context.Context, localPath, remotePath string, contentType string) error {
	if contentType == "" {
		contentType = getContentTypeFromPath(remotePath)
	}

	_, err := m.client.FPutObject(ctx, m.bucket, remotePath, localPath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// UploadReader uploads content from an io.Reader (helper method)
func (m *minioProvider) UploadReader(ctx context.Context, reader io.Reader, remotePath string, size int64, contentType string) error {
	if contentType == "" {
		contentType = getContentTypeFromPath(remotePath)
	}

	_, err := m.client.PutObject(ctx, m.bucket, remotePath, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload from reader: %w", err)
	}

	return nil
}

// Download downloads a file from MinIO to local filesystem
func (m *minioProvider) Download(ctx context.Context, storagePath, localPath string) error {
	// get object from MinIO
	obj, err := m.client.GetObject(ctx, m.bucket, storagePath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	defer obj.Close()

	// create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// copy data from MinIO object to local file
	_, err = io.Copy(localFile, obj)
	if err != nil {
		return fmt.Errorf("failed to copy file data: %w", err)
	}

	return nil
}

// UploadFromPath uploads a file from local filesystem to MinIO
func (m *minioProvider) UploadFromPath(ctx context.Context, localPath, storagePath string) error {
	// Open the local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Determine content type based on file extension
	contentType := getContentTypeFromPath(localPath)

	// Upload file to MinIO
	_, err = m.client.PutObject(ctx, m.bucket, storagePath, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to MinIO: %w", err)
	}

	return nil
}

// getContentTypeFromPath returns the MIME type based on file extension
func getContentTypeFromPath(path string) string {
	if strings.HasSuffix(path, ".m3u8") {
		return "application/x-mpegURL"
	}
	if strings.HasSuffix(path, ".ts") {
		return "video/MP2T"
	}
	if strings.HasSuffix(path, ".mp4") {
		return "video/mp4"
	}
	if strings.HasSuffix(path, ".webm") {
		return "video/webm"
	}
	return "application/octet-stream"
}

// GenerateCDNSignedURL generates a CDN-friendly signed URL with custom options
func (m *minioProvider) GenerateCDNSignedURL(ctx context.Context, path string, opts *CDNSignedURLOptions) (string, error) {
	if opts == nil {
		opts = &CDNSignedURLOptions{
			ExpiresIn: time.Hour * 2,
		}
	}

	// set default expiration if not provided
	expiration := opts.ExpiresIn
	if expiration == 0 {
		expiration = time.Hour * 2
	}

	// create request parameters with cache control headers
	reqParams := make(map[string][]string)
	if opts.CacheControl != "" {
		reqParams["response-cache-control"] = []string{opts.CacheControl}
	}
	if opts.ContentType != "" {
		reqParams["response-content-type"] = []string{opts.ContentType}
	}

	// generate presigned URL using the public client for correct signature
	presignedURL, err := m.publicClient.PresignedGetObject(ctx, m.bucket, path, expiration, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return presignedURL.String(), nil
}

// GenerateSignedURLs generates signed URLs for multiple files (CDN-friendly)
func (m *minioProvider) GenerateSignedURLs(ctx context.Context, paths []string, opts *CDNSignedURLOptions) (map[string]string, error) {
	if opts == nil {
		opts = &CDNSignedURLOptions{
			ExpiresIn: time.Hour * 2,
		}
	}

	result := make(map[string]string)

	// Generate URLs for each path
	for _, path := range paths {
		signedURL, err := m.GenerateCDNSignedURL(ctx, path, opts)
		if err != nil {
			// Continue with other URLs but log the error
			logger.Errorf(err, "failed to generate signed URL. path: %s", path)
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

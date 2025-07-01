package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
	"watch-party/pkg/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	minioEndpoint      = "localhost:19000"
	minioAccessKey     = "minioadmin"
	minioSecretKey     = "minioadmin"
	minioBucketName    = "watch-party-videos"
	minioDataDirectory = "./data"
)

// getMinioDownloadURL returns the download URL for MinIO binary based on OS and architecture
func getMinioDownloadURL() string {
	baseURL := "https://dl.min.io/server/minio/release"
	
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return baseURL + "/linux-amd64/minio"
		case "arm64":
			return baseURL + "/linux-arm64/minio"
		default:
			return baseURL + "/linux-amd64/minio"
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return baseURL + "/darwin-amd64/minio"
		case "arm64":
			return baseURL + "/darwin-arm64/minio"
		default:
			return baseURL + "/darwin-amd64/minio"
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			return baseURL + "/windows-amd64/minio.exe"
		default:
			return baseURL + "/windows-amd64/minio.exe"
		}
	default:
		// default to linux amd64
		return baseURL + "/linux-amd64/minio"
	}
}

// downloadMinIOBinary downloads the MinIO binary if it doesn't exist
func downloadMinIOBinary() (string, error) {
	// create a cache directory for binaries
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}
	
	cacheDir := filepath.Join(homeDir, ".watch-party", "binaries")
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create cache directory: %v", err)
	}
	
	// determine binary name based on OS
	binaryName := "minio"
	if runtime.GOOS == "windows" {
		binaryName = "minio.exe"
	}
	
	binaryPath := filepath.Join(cacheDir, binaryName)
	
	// check if binary already exists
	if _, err := os.Stat(binaryPath); err == nil {
		logger.Info("MinIO binary already exists, using cached version")
		return binaryPath, nil
	}
	
	// download the binary
	downloadURL := getMinioDownloadURL()
	logger.Info(fmt.Sprintf("Downloading MinIO binary from %s...", downloadURL))
	
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download MinIO binary: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download MinIO binary: HTTP %d", resp.StatusCode)
	}
	
	// create the binary file
	file, err := os.Create(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to create binary file: %v", err)
	}
	defer file.Close()
	
	// copy the downloaded content
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write binary file: %v", err)
	}
	
	// make executable
	err = os.Chmod(binaryPath, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to make binary executable: %v", err)
	}
	
	logger.Info("MinIO binary downloaded successfully")
	return binaryPath, nil
}

func startEmbeddedMinio(ctx context.Context) {
	logger.Info("Starting embedded MinIO...")
	
	// create data directory if not exists
	err := os.MkdirAll(minioDataDirectory, 0755)
	if err != nil {
		logger.Fatalf("Failed to create MinIO data directory: %v", err)
		return
	}

	// download MinIO binary if needed
	binaryPath, err := downloadMinIOBinary()
	if err != nil {
		logger.Fatalf("Failed to get MinIO binary: %v", err)
		return
	}

	// build CLI args
	args := []string{
		"server",
		minioDataDirectory,
		"--address", minioEndpoint,
		"--console-address", "localhost:19001",
	}

	// run MinIO
	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		err := cmd.Start()
		if err != nil {
			logger.Fatalf("Failed to start MinIO: %v", err)
		}
	}()

	// wait for health check
	err = waitForMinIOReady()
	if err != nil {
		logger.Fatalf("MinIO did not become ready: %v", err)
		return
	}

	// create bucket
	err = createMinioBucket()
	if err != nil {
		logger.Fatalf("Failed to create MinIO bucket: %v", err)
	}
}

func waitForMinIOReady() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	healthURL := fmt.Sprintf("http://%s/minio/health/live", minioEndpoint)

	for i := 0; i < 30; i++ { // wait up to 30 seconds
		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("MinIO failed to become ready within 30 seconds")
}

func createMinioBucket() error {
	// create MinIO client
	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioAccessKey, minioSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %v", err)
	}

	// retry logic for MinIO readiness and bucket creation
	maxRetries := 30
	retryDelay := time.Second

	for i := 0; i < maxRetries; i++ {
		// check if bucket exists (this will test if MinIO is ready)
		exists, err := minioClient.BucketExists(context.Background(), minioBucketName)
		if err != nil {
			logger.Infof("MinIO not ready yet (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(retryDelay)
			continue
		}

		// MinIO is ready, now handle bucket creation
		if !exists {
			// create bucket
			err = minioClient.MakeBucket(context.Background(), minioBucketName, minio.MakeBucketOptions{})
			if err != nil {
				return fmt.Errorf("failed to create bucket: %v", err)
			}
			logger.Infof("Created MinIO bucket: %s", minioBucketName)
		} else {
			logger.Infof("MinIO bucket already exists: %s", minioBucketName)
		}

		return nil
	}

	return fmt.Errorf("MinIO failed to become ready after %d attempts", maxRetries)
}

// GetMinioEndpoint returns the MinIO endpoint address
func GetMinioEndpoint() string {
	return minioEndpoint
}

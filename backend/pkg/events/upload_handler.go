package events

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"watch-party/pkg/logger"
	"watch-party/pkg/model"
	"watch-party/pkg/storage"
	"watch-party/pkg/video"

	"github.com/google/uuid"
)

// Handler handles storage events like file uploads
type Handler interface {
	HandleUploadComplete(ctx context.Context, event *UploadEvent) error
}

// UploadEvent represents a file upload completion event
type UploadEvent struct {
	MovieID  uuid.UUID `json:"movie_id"`
	FilePath string    `json:"file_path"`
	FileSize int64     `json:"file_size"`
	MimeType string    `json:"mime_type"`
}

// Repository defines the interface for updating movie records
type Repository interface {
	GetByID(id uuid.UUID) (*model.Movie, error)
	UpdateStatus(id uuid.UUID, status model.MovieStatus) error
	UpdateProcessingTimes(id uuid.UUID, startedAt, endedAt *time.Time) error
	UpdateHLSInfo(id uuid.UUID, hlsPlaylistURL, transcodedPath string) error
	Update(movie *model.Movie) error
}

// eventHandler implements the Handler interface
type eventHandler struct {
	movieRepo       Repository
	storageProvider storage.Provider
	videoProcessor  video.Processor
	hlsBaseURL      string // Base URL for accessing HLS files
	tempDir         string // Directory for temporary processing files
}

// NewHandler creates a new event handler
func NewHandler(
	movieRepo Repository,
	storageProvider storage.Provider,
	videoProcessor video.Processor,
	hlsBaseURL string,
	tempDir string,
) Handler {
	return &eventHandler{
		movieRepo:       movieRepo,
		storageProvider: storageProvider,
		videoProcessor:  videoProcessor,
		hlsBaseURL:      hlsBaseURL,
		tempDir:         tempDir,
	}
}

// HandleUploadComplete processes a completed file upload
func (h *eventHandler) HandleUploadComplete(ctx context.Context, event *UploadEvent) error {
	logger.Infof("processing upload completion for movie %s", event.MovieID)

	// get movie record
	movie, err := h.movieRepo.GetByID(event.MovieID)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to get movie %s", event.MovieID))
		return fmt.Errorf("failed to get movie: %w", err)
	}

	if movie == nil {
		return fmt.Errorf("movie not found: %s", event.MovieID)
	}

	// update movie with original file path
	movie.OriginalFilePath = event.FilePath
	movie.FileSize = event.FileSize
	if event.MimeType != "" {
		movie.MimeType = event.MimeType
	}

	err = h.movieRepo.Update(movie)
	if err != nil {
		logger.Error(err, "failed to update movie with file info")
		return fmt.Errorf("failed to update movie: %w", err)
	}

	// validate the uploaded file
	err = h.validateUploadedFile(ctx, event.FilePath)
	if err != nil {
		logger.Error(err, "file validation failed")
		// update status to failed
		updateErr := h.movieRepo.UpdateStatus(event.MovieID, model.StatusFailed)
		if updateErr != nil {
			logger.Error(updateErr, "failed to update movie status to failed")
		}
		return fmt.Errorf("file validation failed: %w", err)
	}

	// start transcoding process
	go h.processVideoAsync(context.Background(), movie)

	logger.Infof("upload processing initiated for movie %s", event.MovieID)
	return nil
}

// validateUploadedFile validates the uploaded file
func (h *eventHandler) validateUploadedFile(ctx context.Context, filePath string) error {
	// check if file exists
	fileInfo, err := h.storageProvider.GetFileInfo(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// basic file size check (max 5GB)
	const maxFileSize = 5 * 1024 * 1024 * 1024 // 5GB
	if fileInfo.Size > maxFileSize {
		return fmt.Errorf("file size too large: %d bytes (max: %d bytes)", fileInfo.Size, maxFileSize)
	}

	// validate file format using storage provider (if the file is accessible)
	// for now, we'll rely on extension-based validation
	ext := filepath.Ext(filePath)
	if !isValidVideoExtension(ext) {
		return fmt.Errorf("unsupported video format: %s", ext)
	}

	return nil
}

// processVideoAsync handles the transcoding process asynchronously
func (h *eventHandler) processVideoAsync(ctx context.Context, movie *model.Movie) {
	movieID := movie.ID
	startTime := time.Now()

	logger.Infof("starting video transcoding for movie %s", movieID)

	// update status to transcoding
	err := h.movieRepo.UpdateStatus(movieID, model.StatusTranscoding)
	if err != nil {
		logger.Error(err, "failed to update movie status to transcoding")
		return
	}

	err = h.movieRepo.UpdateProcessingTimes(movieID, &startTime, nil)
	if err != nil {
		logger.Error(err, "failed to update processing start time")
	}

	// create temporary directory for this movie
	movieTempDir := filepath.Join(h.tempDir, movieID.String())
	outputDir := filepath.Join(movieTempDir, "hls")

	defer func() {
		// cleanup temporary files
		// Note: in production, you might want to keep temp files for debugging
		// os.RemoveAll(movieTempDir)
	}()

	// download file to temporary location for processing
	inputFile := filepath.Join(movieTempDir, "input"+filepath.Ext(movie.OriginalFilePath))
	err = h.downloadFileForProcessing(ctx, movie.OriginalFilePath, inputFile)
	if err != nil {
		h.handleTranscodingError(movieID, fmt.Errorf("failed to download file: %w", err))
		return
	}

	// transcode to HLS
	hlsOutput, err := h.videoProcessor.TranscodeToHLS(ctx, inputFile, outputDir, video.DefaultQualities)
	if err != nil {
		h.handleTranscodingError(movieID, fmt.Errorf("transcoding failed: %w", err))
		return
	}

	// upload HLS files to storage
	transcodedPath, hlsURL, err := h.uploadHLSFiles(ctx, movieID.String(), hlsOutput)
	if err != nil {
		h.handleTranscodingError(movieID, fmt.Errorf("failed to upload HLS files: %w", err))
		return
	}

	// update movie record with completion info
	endTime := time.Now()
	err = h.movieRepo.UpdateProcessingTimes(movieID, &startTime, &endTime)
	if err != nil {
		logger.Error(err, "failed to update processing end time")
	}

	err = h.movieRepo.UpdateHLSInfo(movieID, hlsURL, transcodedPath)
	if err != nil {
		logger.Error(err, "failed to update HLS info")
		h.handleTranscodingError(movieID, fmt.Errorf("failed to update HLS info: %w", err))
		return
	}

	err = h.movieRepo.UpdateStatus(movieID, model.StatusAvailable)
	if err != nil {
		logger.Error(err, "failed to update movie status to available")
		return
	}

	logger.Infof("video transcoding completed successfully for movie %s in %v",
		movieID, endTime.Sub(startTime))
}

// downloadFileForProcessing downloads a file from storage to local temp directory
func (h *eventHandler) downloadFileForProcessing(ctx context.Context, storagePath, localPath string) error {
	// Ensure the directory for the local file exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for local file: %w", err)
	}

	// Use the storage provider's Download method
	err := h.storageProvider.Download(ctx, storagePath, localPath)
	if err != nil {
		return fmt.Errorf("failed to download file from storage: %w", err)
	}

	return nil
}

// uploadHLSFiles uploads all HLS files to storage and returns the transcoded path and public HLS URL
func (h *eventHandler) uploadHLSFiles(ctx context.Context, movieID string, hlsOutput *video.HLSOutput) (string, string, error) {
	// Create base path for transcoded files
	basePath := fmt.Sprintf("transcoded/%s", movieID)

	// Upload master playlist first
	masterPath := fmt.Sprintf("%s/master.m3u8", basePath)
	err := h.storageProvider.UploadFromPath(ctx, hlsOutput.MasterPlaylistPath, masterPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to upload master playlist: %w", err)
	}

	// Upload all quality-specific playlists and segments
	for quality, playlistPath := range hlsOutput.QualityPlaylists {
		// Upload quality playlist
		qualityPlaylistPath := fmt.Sprintf("%s/%s.m3u8", basePath, quality)
		err := h.storageProvider.UploadFromPath(ctx, playlistPath, qualityPlaylistPath)
		if err != nil {
			return "", "", fmt.Errorf("failed to upload %s playlist: %w", quality, err)
		}

		// Upload segments for this quality
		// Segments are typically in the same directory as the playlist
		segmentDir := filepath.Dir(playlistPath)
		segmentPattern := fmt.Sprintf("%s_*.ts", quality)

		// Find all segment files for this quality
		segmentFiles, err := filepath.Glob(filepath.Join(segmentDir, segmentPattern))
		if err != nil {
			return "", "", fmt.Errorf("failed to find segment files for %s: %w", quality, err)
		}

		// Upload each segment file
		for _, segmentFile := range segmentFiles {
			segmentName := filepath.Base(segmentFile)
			segmentStoragePath := fmt.Sprintf("%s/%s", basePath, segmentName)

			err := h.storageProvider.UploadFromPath(ctx, segmentFile, segmentStoragePath)
			if err != nil {
				return "", "", fmt.Errorf("failed to upload segment %s: %w", segmentName, err)
			}
		}
	}

	// Get public URL for the master playlist
	hlsURL, err := h.storageProvider.GetPublicURL(ctx, masterPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get public URL for master playlist: %w", err)
	}

	return basePath, hlsURL, nil
}

// Helper function to get content type from file path (keeping for compatibility)
func getContentTypeFromPath(path string) string {
	if strings.HasSuffix(path, ".m3u8") {
		return "application/x-mpegURL"
	}
	if strings.HasSuffix(path, ".ts") {
		return "video/MP2T"
	}
	return "application/octet-stream"
}

// handleTranscodingError handles transcoding errors
func (h *eventHandler) handleTranscodingError(movieID uuid.UUID, err error) {
	logger.Error(err, fmt.Sprintf("transcoding failed for movie %s", movieID))

	endTime := time.Now()
	updateErr := h.movieRepo.UpdateProcessingTimes(movieID, nil, &endTime)
	if updateErr != nil {
		logger.Error(updateErr, "failed to update processing end time after error")
	}

	updateErr = h.movieRepo.UpdateStatus(movieID, model.StatusFailed)
	if updateErr != nil {
		logger.Error(updateErr, "failed to update movie status to failed")
	}
}

// isValidVideoExtension checks if the file extension is supported
func isValidVideoExtension(ext string) bool {
	supportedFormats := map[string]bool{
		".mp4":  true,
		".avi":  true,
		".mkv":  true,
		".mov":  true,
		".webm": true,
		".m4v":  true,
	}
	return supportedFormats[ext]
}

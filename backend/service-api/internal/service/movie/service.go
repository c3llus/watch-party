package movie

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"watch-party/pkg/logger"
	"watch-party/pkg/model"
	"watch-party/pkg/storage"
	movieRepo "watch-party/service-api/internal/repository/movie"

	"github.com/google/uuid"
)

var (
	ErrMovieNotFound     = errors.New("movie not found")
	ErrUnsupportedFormat = errors.New("unsupported video format")
	ErrInvalidFile       = errors.New("invalid file")
)

// Supported video formats
var supportedFormats = map[string]bool{
	".mp4":  true,
	".avi":  true,
	".mkv":  true,
	".mov":  true,
	".webm": true,
	".m4v":  true,
}

// Service defines the movie service interface
type Service interface {
	InitiateUpload(ctx context.Context, req *model.UploadMovieRequest, uploaderID uuid.UUID) (*model.MovieUploadResponse, error)
	GetMovie(ctx context.Context, id uuid.UUID) (*model.Movie, error)
	GetMovies(ctx context.Context, page, pageSize int) (*model.MovieListResponse, error)
	GetMoviesByUploader(ctx context.Context, uploaderID uuid.UUID, page, pageSize int) (*model.MovieListResponse, error)
	UpdateMovie(ctx context.Context, id uuid.UUID, req *model.UploadMovieRequest) (*model.Movie, error)
	DeleteMovie(ctx context.Context, id uuid.UUID) error
	GetMovieStreamURL(ctx context.Context, id uuid.UUID) (string, error)
	GetMovieStatus(ctx context.Context, id uuid.UUID) (*model.MovieStatusResponse, error)
}

// movieService provides movie-related services.
type movieService struct {
	movieRepo       movieRepo.Repository
	storageProvider storage.Provider
}

// NewMovieService creates a new movie service instance.
func NewMovieService(movieRepo movieRepo.Repository, storageProvider storage.Provider) Service {
	return &movieService{
		movieRepo:       movieRepo,
		storageProvider: storageProvider,
	}
}

// InitiateUpload creates a movie record and returns signed URL for upload
func (s *movieService) InitiateUpload(ctx context.Context, req *model.UploadMovieRequest, uploaderID uuid.UUID) (*model.MovieUploadResponse, error) {
	// validate request
	err := s.validateUploadRequest(req)
	if err != nil {
		return nil, err
	}

	// generate unique filename
	ext := filepath.Ext(req.FileName)
	filename := fmt.Sprintf("uploads/%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)

	// create movie record with processing status
	movie := &model.Movie{
		ID:                  uuid.New(),
		Title:               req.Title,
		Description:         req.Description,
		OriginalFilePath:    filename, // will be the final path after upload
		TranscodedFilePath:  "",
		HLSPlaylistURL:      "",
		DurationSeconds:     0,
		FileSize:            req.FileSize,
		MimeType:            s.getMimeTypeFromFilename(req.FileName),
		Status:              model.StatusProcessing,
		UploadedBy:          uploaderID,
		CreatedAt:           time.Now(),
		ProcessingStartedAt: nil,
		ProcessingEndedAt:   nil,
	}

	// save movie record to database
	err = s.movieRepo.Create(movie)
	if err != nil {
		return nil, fmt.Errorf("failed to create movie record: %w", err)
	}

	// generate signed URL for upload
	uploadOpts := &storage.UploadOptions{
		ContentType: movie.MimeType,
		MaxFileSize: req.FileSize,
		ExpiresIn:   time.Hour, // URL expires in 1 hour
		Public:      false,
	}

	signedURL, err := s.storageProvider.GenerateSignedUploadURL(ctx, filename, uploadOpts)
	if err != nil {
		// cleanup movie record if signed URL generation fails
		deleteErr := s.movieRepo.Delete(movie.ID)
		if deleteErr != nil {
			logger.Error(deleteErr, "failed to cleanup movie record after signed URL generation failed")
		}
		return nil, fmt.Errorf("failed to generate signed upload URL: %w", err)
	}

	logger.Infof("upload initiated for movie %s: %s", movie.Title, movie.ID)

	return &model.MovieUploadResponse{
		MovieID:   movie.ID,
		SignedURL: signedURL.URL,
		FilePath:  filename,
		Message:   "Upload initiated successfully. Use the signed URL to upload your video file.",
	}, nil
}

// GetMovie retrieves a movie by ID
func (s *movieService) GetMovie(ctx context.Context, id uuid.UUID) (*model.Movie, error) {
	movie, err := s.movieRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if movie == nil {
		return nil, ErrMovieNotFound
	}
	return movie, nil
}

// GetMovies retrieves movies with pagination
func (s *movieService) GetMovies(ctx context.Context, page, pageSize int) (*model.MovieListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	movies, totalCount, err := s.movieRepo.GetAll(pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &model.MovieListResponse{
		Movies:     movies,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// GetMoviesByUploader retrieves movies uploaded by a specific user
func (s *movieService) GetMoviesByUploader(ctx context.Context, uploaderID uuid.UUID, page, pageSize int) (*model.MovieListResponse, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	movies, totalCount, err := s.movieRepo.GetByUploader(uploaderID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return &model.MovieListResponse{
		Movies:     movies,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

// UpdateMovie updates a movie's metadata
func (s *movieService) UpdateMovie(ctx context.Context, id uuid.UUID, req *model.UploadMovieRequest) (*model.Movie, error) {
	// check if movie exists
	movie, err := s.movieRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if movie == nil {
		return nil, ErrMovieNotFound
	}

	// update movie fields
	movie.Title = req.Title
	movie.Description = req.Description

	// save updates
	err = s.movieRepo.Update(movie)
	if err != nil {
		return nil, err
	}

	return movie, nil
}

// DeleteMovie deletes a movie and its associated file
func (s *movieService) DeleteMovie(ctx context.Context, id uuid.UUID) error {
	// get movie details
	movie, err := s.movieRepo.GetByID(id)
	if err != nil {
		return err
	}
	if movie == nil {
		return ErrMovieNotFound
	}

	// delete from database first
	err = s.movieRepo.Delete(id)
	if err != nil {
		return err
	}

	// delete original file from storage if it exists
	if movie.OriginalFilePath != "" {
		err = s.storageProvider.Delete(ctx, movie.OriginalFilePath)
		if err != nil {
			logger.Error(err, "failed to delete original movie file from storage")
		}
	}

	// delete transcoded files from storage if they exist
	if movie.TranscodedFilePath != "" {
		// delete all transcoded files (this would need implementation based on storage structure)
		err = s.deleteTranscodedFiles(ctx, movie.TranscodedFilePath)
		if err != nil {
			logger.Error(err, "failed to delete transcoded files from storage")
		}
	}

	logger.Infof("movie deleted successfully: %s (ID: %s)", movie.Title, id)
	return nil
}

// GetMovieStreamURL returns a signed URL for streaming the movie
func (s *movieService) GetMovieStreamURL(ctx context.Context, id uuid.UUID) (string, error) {
	movie, err := s.movieRepo.GetByID(id)
	if err != nil {
		return "", err
	}
	if movie == nil {
		return "", ErrMovieNotFound
	}

	// if movie is not available yet, return error
	if movie.Status != model.StatusAvailable {
		return "", fmt.Errorf("movie is not ready for streaming (status: %s)", movie.Status)
	}

	// return HLS playlist URL if available
	if movie.HLSPlaylistURL != "" {
		return movie.HLSPlaylistURL, nil
	}

	// fallback to original file (though this shouldn't happen in the new workflow)
	if movie.OriginalFilePath != "" {
		url, err := s.storageProvider.GetSignedURL(ctx, movie.OriginalFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to generate stream URL: %w", err)
		}
		return url, nil
	}

	return "", fmt.Errorf("no streamable content available for movie")
}

// GetMovieStatus returns the processing status of a movie
func (s *movieService) GetMovieStatus(ctx context.Context, id uuid.UUID) (*model.MovieStatusResponse, error) {
	movie, err := s.movieRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if movie == nil {
		return nil, ErrMovieNotFound
	}

	response := &model.MovieStatusResponse{
		MovieID:             movie.ID,
		Status:              movie.Status,
		Title:               movie.Title,
		ProcessingStartedAt: movie.ProcessingStartedAt,
		ProcessingEndedAt:   movie.ProcessingEndedAt,
	}

	// include HLS URL if available
	if movie.Status == model.StatusAvailable && movie.HLSPlaylistURL != "" {
		response.HLSPlaylistURL = movie.HLSPlaylistURL
	}

	// include error message for failed movies (you might want to add an error field to the Movie model)
	if movie.Status == model.StatusFailed {
		response.ErrorMessage = "Video processing failed"
	}

	return response, nil
}

// validateUploadRequest validates the upload request
func (s *movieService) validateUploadRequest(req *model.UploadMovieRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	// validate file extension
	ext := strings.ToLower(filepath.Ext(req.FileName))
	if !supportedFormats[ext] {
		return ErrUnsupportedFormat
	}

	// validate file size (max 5GB)
	const maxFileSize = 5 * 1024 * 1024 * 1024 // 5GB
	if req.FileSize > maxFileSize {
		return fmt.Errorf("file size too large: %d bytes (max: %d bytes)", req.FileSize, maxFileSize)
	}

	if req.FileSize <= 0 {
		return fmt.Errorf("invalid file size: %d", req.FileSize)
	}

	return nil
}

// getMimeTypeFromFilename returns the MIME type based on file extension
func (s *movieService) getMimeTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	return s.getMimeType(ext)
}

// deleteTranscodedFiles deletes all transcoded files for a movie
func (s *movieService) deleteTranscodedFiles(ctx context.Context, transcodedPath string) error {
	// list all files under the transcoded path
	files, err := s.storageProvider.ListObjects(ctx, transcodedPath)
	if err != nil {
		return fmt.Errorf("failed to list transcoded files: %w", err)
	}

	// delete each file
	for _, file := range files {
		err = s.storageProvider.Delete(ctx, file)
		if err != nil {
			logger.Error(err, fmt.Sprintf("failed to delete transcoded file: %s", file))
		}
	}

	return nil
}

// getMimeType returns the MIME type based on file extension
func (s *movieService) getMimeType(ext string) string {
	switch strings.ToLower(ext) {
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
	case ".m4v":
		return "video/mp4"
	default:
		return "application/octet-stream"
	}
}

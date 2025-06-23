package movie

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
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
	UploadMovie(ctx context.Context, req *model.UploadMovieRequest, file *multipart.FileHeader, uploaderID uuid.UUID) (*model.Movie, error)
	GetMovie(ctx context.Context, id uuid.UUID) (*model.Movie, error)
	GetMovies(ctx context.Context, page, pageSize int) (*model.MovieListResponse, error)
	GetMoviesByUploader(ctx context.Context, uploaderID uuid.UUID, page, pageSize int) (*model.MovieListResponse, error)
	UpdateMovie(ctx context.Context, id uuid.UUID, req *model.UploadMovieRequest) (*model.Movie, error)
	DeleteMovie(ctx context.Context, id uuid.UUID) error
	GetMovieStreamURL(ctx context.Context, id uuid.UUID) (string, error)
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

// UploadMovie uploads a movie file and stores its metadata
func (s *movieService) UploadMovie(ctx context.Context, req *model.UploadMovieRequest, file *multipart.FileHeader, uploaderID uuid.UUID) (*model.Movie, error) {
	// validate file
	err := s.validateFile(file)
	if err != nil {
		return nil, err
	}

	// generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)

	// upload file to storage
	storagePath, err := s.storageProvider.Upload(ctx, file, filename)
	if err != nil {
		logger.Error(err, "failed to upload movie file")
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	// create movie record
	movie := &model.Movie{
		ID:              uuid.New(),
		Title:           req.Title,
		Description:     req.Description,
		StorageProvider: s.getStorageProviderType(),
		StoragePath:     storagePath,
		DurationSeconds: 0, // Will be updated later if needed
		FileSize:        file.Size,
		MimeType:        s.getMimeType(ext),
		UploadedBy:      uploaderID,
		CreatedAt:       time.Now(),
	}

	// save to database
	err = s.movieRepo.Create(movie)
	if err != nil {
		// if database save fails, try to cleanup uploaded file
		deleteErr := s.storageProvider.Delete(ctx, storagePath)
		if deleteErr != nil {
			logger.Error(deleteErr, "failed to cleanup uploaded file after database error")
		}
		return nil, fmt.Errorf("failed to save movie metadata: %w", err)
	}

	logger.Infof("movie uploaded successfully: %s (ID: %s)", movie.Title, movie.ID)
	return movie, nil
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

	// delete file from storage
	err = s.storageProvider.Delete(ctx, movie.StoragePath)
	if err != nil {
		logger.Error(err, "failed to delete movie file from storage")
		// don't return error here as the database record is already deleted
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

	// get signed URL from storage provider
	url, err := s.storageProvider.GetSignedURL(ctx, movie.StoragePath)
	if err != nil {
		return "", fmt.Errorf("failed to generate stream URL: %w", err)
	}

	return url, nil
}

// validateFile validates the uploaded file
func (s *movieService) validateFile(file *multipart.FileHeader) error {
	if file == nil {
		return ErrInvalidFile
	}

	// check file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !supportedFormats[ext] {
		return ErrUnsupportedFormat
	}

	// check file size (max 5GB)
	const maxFileSize = 5 * 1024 * 1024 * 1024 // 5GB
	if file.Size > maxFileSize {
		return fmt.Errorf("file size too large: %d bytes (max: %d bytes)", file.Size, maxFileSize)
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

// getStorageProviderType returns the storage provider type
func (s *movieService) getStorageProviderType() string {
	// this is a simplified approach - in a real implementation,
	// you might want to pass this information to the service
	return model.StorageProviderLocal // Default
}

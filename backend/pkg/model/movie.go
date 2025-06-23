package model

import (
	"time"

	"github.com/google/uuid"
)

// Movie represents a movie in the system
type Movie struct {
	ID              uuid.UUID `json:"id" db:"id"`
	Title           string    `json:"title" db:"title"`
	Description     string    `json:"description" db:"description"`
	StorageProvider string    `json:"storage_provider" db:"storage_provider"` // "local", "gcs"
	StoragePath     string    `json:"storage_path" db:"storage_path"`         // Path/URL to the video file
	DurationSeconds int       `json:"duration_seconds" db:"duration_seconds"`
	FileSize        int64     `json:"file_size" db:"file_size"`     // File size in bytes
	MimeType        string    `json:"mime_type" db:"mime_type"`     // e.g., "video/mp4"
	UploadedBy      uuid.UUID `json:"uploaded_by" db:"uploaded_by"` // Admin who uploaded
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Storage provider constants
const (
	StorageProviderLocal = "local"
	StorageProviderGCS   = "gcs"
)

// UploadMovieRequest represents the request for uploading a movie
type UploadMovieRequest struct {
	Title       string `form:"title" binding:"required"`
	Description string `form:"description"`
}

// MovieListResponse represents a paginated list of movies
type MovieListResponse struct {
	Movies     []Movie `json:"movies"`
	TotalCount int     `json:"total_count"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
}

// MovieUploadResponse represents the response after successful movie upload
type MovieUploadResponse struct {
	Movie   Movie  `json:"movie"`
	Message string `json:"message"`
}

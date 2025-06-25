package model

import (
	"time"

	"github.com/google/uuid"
)

// MovieStatus defines the state of a movie in the processing pipeline.
type MovieStatus string

const (
	StatusProcessing  MovieStatus = "processing"
	StatusTranscoding MovieStatus = "transcoding"
	StatusAvailable   MovieStatus = "available"
	StatusFailed      MovieStatus = "failed"
)

type Movie struct {
	ID                  uuid.UUID   `json:"id" db:"id"`
	Title               string      `json:"title" db:"title"`
	Description         string      `json:"description" db:"description"`
	OriginalFilePath    string      `json:"original_file_path" db:"original_file_path"`     // Path to the original uploaded file
	TranscodedFilePath  string      `json:"transcoded_file_path" db:"transcoded_file_path"` // Path to transcoded output directory
	HLSPlaylistURL      string      `json:"hls_playlist_url" db:"hls_playlist_url"`         // Public URL to the .m3u8 file
	DurationSeconds     int         `json:"duration_seconds" db:"duration_seconds"`
	FileSize            int64       `json:"file_size" db:"file_size"` // Original file size
	MimeType            string      `json:"mime_type" db:"mime_type"` // Original mime type
	Status              MovieStatus `json:"status" db:"status"`
	UploadedBy          uuid.UUID   `json:"uploaded_by" db:"uploaded_by"`
	CreatedAt           time.Time   `json:"created_at" db:"created_at"`
	ProcessingStartedAt *time.Time  `json:"processing_started_at" db:"processing_started_at"` // When transcoding started
	ProcessingEndedAt   *time.Time  `json:"processing_ended_at" db:"processing_ended_at"`     // When transcoding completed
}

// Storage provider constants
const (
	StorageProviderGCS   = "gcs"
	StorageProviderMinIO = "minio"
)

// UploadMovieRequest represents the request for uploading a movie
type UploadMovieRequest struct {
	Title       string `form:"title" binding:"required"`
	Description string `form:"description"`
	FileName    string `form:"filename" binding:"required"` // Required for signed URL generation
	FileSize    int64  `form:"filesize" binding:"required"` // Required for validation
	MimeType    string `form:"mimetype"`                    // Optional, will be inferred if not provided
}

// MovieListResponse represents a paginated list of movies
type MovieListResponse struct {
	Movies     []Movie `json:"movies"`
	TotalCount int     `json:"total_count"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
}

// MovieUploadResponse represents the response after successful movie upload initiation
type MovieUploadResponse struct {
	MovieID   uuid.UUID `json:"movie_id"`
	SignedURL string    `json:"signed_url"`
	Message   string    `json:"message"`
}

// MovieStatusResponse represents the status of a movie processing
type MovieStatusResponse struct {
	MovieID             uuid.UUID   `json:"movie_id"`
	Status              MovieStatus `json:"status"`
	Title               string      `json:"title"`
	HLSPlaylistURL      string      `json:"hls_playlist_url,omitempty"`
	ProcessingStartedAt *time.Time  `json:"processing_started_at,omitempty"`
	ProcessingEndedAt   *time.Time  `json:"processing_ended_at,omitempty"`
	ErrorMessage        string      `json:"error_message,omitempty"`
}

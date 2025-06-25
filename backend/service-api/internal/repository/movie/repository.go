package movie

import (
	"database/sql"
	"fmt"
	"time"
	"watch-party/pkg/model"

	"github.com/google/uuid"
)

// Repository defines the movie repository interface
type Repository interface {
	Create(movie *model.Movie) error
	GetByID(id uuid.UUID) (*model.Movie, error)
	GetAll(limit, offset int) ([]model.Movie, int, error)
	Update(movie *model.Movie) error
	Delete(id uuid.UUID) error
	GetByUploader(uploaderID uuid.UUID, limit, offset int) ([]model.Movie, int, error)
	UpdateStatus(id uuid.UUID, status model.MovieStatus) error
	UpdateProcessingTimes(id uuid.UUID, startedAt, endedAt *time.Time) error
	UpdateHLSInfo(id uuid.UUID, hlsPlaylistURL, transcodedPath string) error
}

// repository implements the movie repository
type repository struct {
	db *sql.DB
}

// NewRepository creates a new movie repository
func NewRepository(db *sql.DB) Repository {
	return &repository{
		db: db,
	}
}

// Create creates a new movie in the database
func (r *repository) Create(movie *model.Movie) error {
	query := `
		INSERT INTO movies (id, title, description, original_file_path, transcoded_file_path, 
			hls_playlist_url, duration_seconds, file_size, mime_type, status, uploaded_by, 
			created_at, processing_started_at, processing_ended_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.db.Exec(query,
		movie.ID, movie.Title, movie.Description, movie.OriginalFilePath,
		movie.TranscodedFilePath, movie.HLSPlaylistURL, movie.DurationSeconds,
		movie.FileSize, movie.MimeType, movie.Status, movie.UploadedBy,
		movie.CreatedAt, movie.ProcessingStartedAt, movie.ProcessingEndedAt)
	return err
}

// GetByID retrieves a movie by ID
func (r *repository) GetByID(id uuid.UUID) (*model.Movie, error) {
	movie := &model.Movie{}
	query := `
		SELECT id, title, description, original_file_path, transcoded_file_path, 
			hls_playlist_url, duration_seconds, file_size, mime_type, status, 
			uploaded_by, created_at, processing_started_at, processing_ended_at
		FROM movies 
		WHERE id = $1`

	row := r.db.QueryRow(query, id)
	err := row.Scan(&movie.ID, &movie.Title, &movie.Description,
		&movie.OriginalFilePath, &movie.TranscodedFilePath, &movie.HLSPlaylistURL,
		&movie.DurationSeconds, &movie.FileSize, &movie.MimeType, &movie.Status,
		&movie.UploadedBy, &movie.CreatedAt, &movie.ProcessingStartedAt, &movie.ProcessingEndedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Movie not found
		}
		return nil, err
	}

	return movie, nil
}

// GetAll retrieves all movies with pagination
func (r *repository) GetAll(limit, offset int) ([]model.Movie, int, error) {
	// get total count
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM movies"
	err := r.db.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get movies count: %w", err)
	}

	// get movies with pagination
	query := `
		SELECT id, title, description, original_file_path, transcoded_file_path, 
			hls_playlist_url, duration_seconds, file_size, mime_type, status, 
			uploaded_by, created_at, processing_started_at, processing_ended_at
		FROM movies 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query movies: %w", err)
	}
	defer rows.Close()

	var movies []model.Movie
	for rows.Next() {
		var movie model.Movie
		err := rows.Scan(&movie.ID, &movie.Title, &movie.Description,
			&movie.OriginalFilePath, &movie.TranscodedFilePath, &movie.HLSPlaylistURL,
			&movie.DurationSeconds, &movie.FileSize, &movie.MimeType, &movie.Status,
			&movie.UploadedBy, &movie.CreatedAt, &movie.ProcessingStartedAt, &movie.ProcessingEndedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan movie: %w", err)
		}
		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return movies, totalCount, nil
}

// Update updates a movie in the database
func (r *repository) Update(movie *model.Movie) error {
	query := `
		UPDATE movies 
		SET title = $2, description = $3, original_file_path = $4, transcoded_file_path = $5,
			hls_playlist_url = $6, duration_seconds = $7, file_size = $8, mime_type = $9,
			status = $10, processing_started_at = $11, processing_ended_at = $12
		WHERE id = $1`

	result, err := r.db.Exec(query, movie.ID, movie.Title, movie.Description,
		movie.OriginalFilePath, movie.TranscodedFilePath, movie.HLSPlaylistURL,
		movie.DurationSeconds, movie.FileSize, movie.MimeType, movie.Status,
		movie.ProcessingStartedAt, movie.ProcessingEndedAt)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("movie not found")
	}

	return nil
}

// Delete deletes a movie from the database
func (r *repository) Delete(id uuid.UUID) error {
	query := "DELETE FROM movies WHERE id = $1"
	result, err := r.db.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("movie not found")
	}

	return nil
}

// GetByUploader retrieves movies uploaded by a specific user
func (r *repository) GetByUploader(uploaderID uuid.UUID, limit, offset int) ([]model.Movie, int, error) {
	// Get total count for the uploader
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM movies WHERE uploaded_by = $1"
	err := r.db.QueryRow(countQuery, uploaderID).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get movies count: %w", err)
	}

	// get movies with pagination
	query := `
		SELECT id, title, description, original_file_path, transcoded_file_path, 
			hls_playlist_url, duration_seconds, file_size, mime_type, status, 
			uploaded_by, created_at, processing_started_at, processing_ended_at
		FROM movies 
		WHERE uploaded_by = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(query, uploaderID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query movies: %w", err)
	}
	defer rows.Close()

	var movies []model.Movie
	for rows.Next() {
		var movie model.Movie
		err := rows.Scan(&movie.ID, &movie.Title, &movie.Description,
			&movie.OriginalFilePath, &movie.TranscodedFilePath, &movie.HLSPlaylistURL,
			&movie.DurationSeconds, &movie.FileSize, &movie.MimeType, &movie.Status,
			&movie.UploadedBy, &movie.CreatedAt, &movie.ProcessingStartedAt, &movie.ProcessingEndedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan movie: %w", err)
		}
		movies = append(movies, movie)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return movies, totalCount, nil
}

// UpdateStatus updates the status of a movie
func (r *repository) UpdateStatus(id uuid.UUID, status model.MovieStatus) error {
	query := `UPDATE movies SET status = $2 WHERE id = $1`

	result, err := r.db.Exec(query, id, status)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("movie not found")
	}

	return nil
}

// UpdateProcessingTimes updates the processing start and end times
func (r *repository) UpdateProcessingTimes(id uuid.UUID, startedAt, endedAt *time.Time) error {
	query := `UPDATE movies SET processing_started_at = $2, processing_ended_at = $3 WHERE id = $1`

	result, err := r.db.Exec(query, id, startedAt, endedAt)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("movie not found")
	}

	return nil
}

// UpdateHLSInfo updates the HLS playlist URL and transcoded file path
func (r *repository) UpdateHLSInfo(id uuid.UUID, hlsPlaylistURL, transcodedPath string) error {
	query := `UPDATE movies SET hls_playlist_url = $2, transcoded_file_path = $3 WHERE id = $1`

	result, err := r.db.Exec(query, id, hlsPlaylistURL, transcodedPath)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("movie not found")
	}

	return nil
}

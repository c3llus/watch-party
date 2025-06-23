package movie

import (
	"database/sql"
	"fmt"
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
		INSERT INTO movies (id, title, description, storage_provider, storage_path, 
			duration_seconds, file_size, mime_type, uploaded_by, created_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(query,
		movie.ID, movie.Title, movie.Description, movie.StorageProvider,
		movie.StoragePath, movie.DurationSeconds, movie.FileSize,
		movie.MimeType, movie.UploadedBy, movie.CreatedAt)
	return err
}

// GetByID retrieves a movie by ID
func (r *repository) GetByID(id uuid.UUID) (*model.Movie, error) {
	movie := &model.Movie{}
	query := `
		SELECT id, title, description, storage_provider, storage_path, 
			duration_seconds, file_size, mime_type, uploaded_by, created_at 
		FROM movies 
		WHERE id = $1`

	row := r.db.QueryRow(query, id)
	err := row.Scan(&movie.ID, &movie.Title, &movie.Description,
		&movie.StorageProvider, &movie.StoragePath, &movie.DurationSeconds,
		&movie.FileSize, &movie.MimeType, &movie.UploadedBy, &movie.CreatedAt)
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
	// Get total count
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM movies"
	err := r.db.QueryRow(countQuery).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get movies count: %w", err)
	}

	// Get movies with pagination
	query := `
		SELECT id, title, description, storage_provider, storage_path, 
			duration_seconds, file_size, mime_type, uploaded_by, created_at 
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
			&movie.StorageProvider, &movie.StoragePath, &movie.DurationSeconds,
			&movie.FileSize, &movie.MimeType, &movie.UploadedBy, &movie.CreatedAt)
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
		SET title = $2, description = $3, duration_seconds = $4
		WHERE id = $1`

	result, err := r.db.Exec(query, movie.ID, movie.Title, movie.Description, movie.DurationSeconds)
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

	// Get movies with pagination
	query := `
		SELECT id, title, description, storage_provider, storage_path, 
			duration_seconds, file_size, mime_type, uploaded_by, created_at 
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
			&movie.StorageProvider, &movie.StoragePath, &movie.DurationSeconds,
			&movie.FileSize, &movie.MimeType, &movie.UploadedBy, &movie.CreatedAt)
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

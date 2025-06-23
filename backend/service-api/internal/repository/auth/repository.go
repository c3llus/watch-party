package auth

import (
	"database/sql"
	"time"
	"watch-party/pkg/model"

	"github.com/google/uuid"
)

// Repository defines the auth repository interface
type Repository interface {
	StoreRefreshToken(userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	GetRefreshToken(tokenHash string) (*model.Token, error)
	DeleteRefreshToken(tokenHash string) error
	DeleteAllUserTokens(userID uuid.UUID) error
}

// repository implements the auth repository
type repository struct {
	db *sql.DB
}

// NewRepository creates a new auth repository
func NewRepository(db *sql.DB) Repository {
	return &repository{
		db: db,
	}
}

// StoreRefreshToken stores a refresh token hash in the database
func (r *repository) StoreRefreshToken(userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO tokens (id, user_id, value, created_at) 
		VALUES ($1, $2, $3, $4)`

	id := uuid.New()
	createdAt := time.Now()

	_, err := r.db.Exec(query, id, userID, tokenHash, createdAt)
	return err
}

// GetRefreshToken retrieves a refresh token by hash
func (r *repository) GetRefreshToken(tokenHash string) (*model.Token, error) {
	token := &model.Token{}
	query := `
		SELECT id, user_id, value, created_at 
		FROM tokens 
		WHERE value = $1`

	row := r.db.QueryRow(query, tokenHash)
	err := row.Scan(&token.ID, &token.UserID, &token.Value, &token.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Token not found or expired
		}
		return nil, err
	}

	return token, nil
}

// DeleteRefreshToken deletes a refresh token by hash
func (r *repository) DeleteRefreshToken(tokenHash string) error {
	query := `DELETE FROM tokens WHERE value = $1`
	_, err := r.db.Exec(query, tokenHash)
	return err
}

// DeleteAllUserTokens deletes all refresh tokens for a user
func (r *repository) DeleteAllUserTokens(userID uuid.UUID) error {
	query := `DELETE FROM tokens WHERE user_id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}

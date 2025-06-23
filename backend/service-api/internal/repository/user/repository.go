package user

import (
	"database/sql"
	"watch-party/pkg/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Repository defines the user repository interface
type Repository interface {
	Create(user *model.User) error
	GetByEmail(email string) (*model.User, error)
	GetByID(id uuid.UUID) (*model.User, error)
}

// repository implements the user repository
type repository struct {
	db *sql.DB
}

// NewRepository creates a new user repository
func NewRepository(db *sql.DB) Repository {
	return &repository{
		db: db,
	}
}

// Create creates a new user in the database
func (r *repository) Create(user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, role, created_at) 
		VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Exec(query, user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt)
	return err
}

// GetByEmail retrieves a user by email
func (r *repository) GetByEmail(email string) (*model.User, error) {
	user := &model.User{}
	query := `
		SELECT id, email, password_hash, role, created_at 
		FROM users 
		WHERE email = $1`

	row := r.db.QueryRow(query, email)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *repository) GetByID(id uuid.UUID) (*model.User, error) {
	user := &model.User{}
	query := `
		SELECT id, email, password_hash, role, created_at 
		FROM users 
		WHERE id = $1`

	row := r.db.QueryRow(query, id)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // User not found
		}
		return nil, err
	}

	return user, nil
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

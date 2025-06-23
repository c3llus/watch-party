package user

import (
	"errors"
	"time"
	"watch-party/pkg/model"
	userRepo "watch-party/service-api/internal/repository/user"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

// Service defines the user service interface
type Service interface {
	RegisterUser(req *model.RegisterRequest, role string) (*model.User, error)
	GetUserByEmail(email string) (*model.User, error)
	GetUserByID(id uuid.UUID) (*model.User, error)
}

// userService provides user-related services.
type userService struct {
	userRepo userRepo.Repository
}

// NewUserService creates a new user service instance.
func NewUserService(userRepo userRepo.Repository) Service {
	return &userService{
		userRepo: userRepo,
	}
}

// RegisterUser registers a new user
func (s *userService) RegisterUser(req *model.RegisterRequest, role string) (*model.User, error) {
	// check if user already exists
	existingUser, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// hash the password
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// create user
	user := &model.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         role,
		CreatedAt:    time.Now(),
	}

	// save user to database
	err = s.userRepo.Create(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *userService) GetUserByEmail(email string) (*model.User, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(id uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// HashPassword hashes a password using bcrypt
func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

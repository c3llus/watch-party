package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
	"watch-party/pkg/auth"
	"watch-party/pkg/config"
	"watch-party/pkg/model"
	authRepo "watch-party/service-api/internal/repository/auth"
	userRepo "watch-party/service-api/internal/repository/user"
	userService "watch-party/service-api/internal/service/user"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Service defines the auth service interface
type Service interface {
	Login(req *model.LoginRequest) (*model.LoginResponse, error)
	RegisterAdmin(req *model.RegisterRequest) (*model.User, error)
	RegisterUser(req *model.RegisterRequest) (*model.User, error)
	Logout(refreshToken string) error
}

// authService provides auth-related services.
type authService struct {
	jwtManager  *auth.JWTManager
	userService userService.Service
	authRepo    authRepo.Repository
}

// NewAuthService creates a new auth service instance.
func NewAuthService(
	cfg *config.Config,
	userService userService.Service,
	authRepo authRepo.Repository,
) Service {
	return &authService{
		jwtManager:  auth.NewJWTManager(cfg.JWTSecret),
		userService: userService,
		authRepo:    authRepo,
	}
}

// Login authenticates a user and returns tokens
func (s *authService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
	// Get user by email
	user, err := s.userService.GetUserByEmail(req.Email)
	if err != nil {
		if err == userService.ErrUserNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	// Verify password
	err = userRepo.VerifyPassword(user.PasswordHash, req.Password)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(user)
	if err != nil {
		return nil, err
	}

	// Store refresh token hash in database
	refreshTokenHash := hashToken(refreshToken)
	expiresAt := time.Now().Add(time.Hour * 24 * 7) // 7 days
	err = s.authRepo.StoreRefreshToken(user.ID, refreshTokenHash, expiresAt)
	if err != nil {
		return nil, err
	}

	return &model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         *user,
	}, nil
}

// RegisterAdmin registers a new admin user
func (s *authService) RegisterAdmin(req *model.RegisterRequest) (*model.User, error) {
	return s.userService.RegisterUser(req, model.RoleAdmin)
}

// RegisterUser registers a new regular user
func (s *authService) RegisterUser(req *model.RegisterRequest) (*model.User, error) {
	return s.userService.RegisterUser(req, model.RoleUser)
}

// Logout invalidates a refresh token
func (s *authService) Logout(refreshToken string) error {
	refreshTokenHash := hashToken(refreshToken)
	return s.authRepo.DeleteRefreshToken(refreshTokenHash)
}

// hashToken creates a SHA-256 hash of a token for storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

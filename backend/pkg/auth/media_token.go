package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MediaTokenClaims represents the claims in a media access token
type MediaTokenClaims struct {
	MovieID   string `json:"movie_id"`
	FilePath  string `json:"file_path"`
	UserID    string `json:"user_id,omitempty"`
	RoomID    string `json:"room_id,omitempty"`
	IsGuest   bool   `json:"is_guest"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// MediaTokenService handles media token generation and validation
type MediaTokenService struct {
	signingKey []byte
	tokenTTL   time.Duration
}

// NewMediaTokenService creates a new media token service
func NewMediaTokenService(signingKey string, tokenTTLSeconds int) *MediaTokenService {
	return &MediaTokenService{
		signingKey: []byte(signingKey),
		tokenTTL:   time.Duration(tokenTTLSeconds) * time.Second,
	}
}

// GenerateToken creates a short-lived media access token
func (mts *MediaTokenService) GenerateToken(movieID, filePath string, userID *uuid.UUID, roomID *uuid.UUID, isGuest bool) (string, error) {
	now := time.Now()
	claims := MediaTokenClaims{
		MovieID:   movieID,
		FilePath:  filePath,
		IsGuest:   isGuest,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(mts.tokenTTL).Unix(),
	}

	if userID != nil {
		claims.UserID = userID.String()
	}
	if roomID != nil {
		claims.RoomID = roomID.String()
	}

	// create header
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// create signature
	message := headerB64 + "." + claimsB64
	signature := mts.sign(message)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	token := message + "." + signatureB64
	return token, nil
}

// ValidateToken validates a media access token and returns the claims
func (mts *MediaTokenService) ValidateToken(token string) (*MediaTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// verify signature
	message := parts[0] + "." + parts[1]
	expectedSignature := mts.sign(message)

	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	if !hmac.Equal(expectedSignature, providedSignature) {
		return nil, fmt.Errorf("invalid signature")
	}

	// decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims MediaTokenClaims
	err = json.Unmarshal(claimsJSON, &claims)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// check expiration
	now := time.Now().Unix()
	if claims.ExpiresAt < now {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// sign creates HMAC-SHA256 signature
func (mts *MediaTokenService) sign(message string) []byte {
	h := hmac.New(sha256.New, mts.signingKey)
	h.Write([]byte(message))
	return h.Sum(nil)
}

// GenerateCDNURL creates a CDN URL with embedded media token
func (mts *MediaTokenService) GenerateCDNURL(baseURL, movieID, filePath string, userID *uuid.UUID, roomID *uuid.UUID, isGuest bool) (string, error) {
	token, err := mts.GenerateToken(movieID, filePath, userID, roomID, isGuest)
	if err != nil {
		return "", err
	}

	// construct URL with token
	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}

	return fmt.Sprintf("%s%smedia_token=%s", baseURL, separator, token), nil
}

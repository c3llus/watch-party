package middleware

import (
	"context"
	"net/http"
	"strings"
	"watch-party/pkg/auth"
	"watch-party/pkg/logger"
	roomService "watch-party/service-api/internal/service/room"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// StreamingAuthMiddleware creates middleware for streaming endpoints that validates
// user access to rooms containing the requested movie
func StreamingAuthMiddleware(jwtManager *auth.JWTManager, roomSvc *roomService.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		movieIDStr := c.Param("movieId")
		if movieIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "movie ID required"})
			c.Abort()
			return
		}

		movieID, err := uuid.Parse(movieIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid movie ID"})
			c.Abort()
			return
		}

		// try to authenticate via JWT token first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			if authenticateWithJWT(c, jwtManager, roomSvc, movieID) {
				c.Next()
				return
			}
		}

		// try guest session token as fallback
		guestToken := c.Query("token")
		if guestToken == "" {
			guestToken = c.GetHeader("X-Guest-Token")
		}

		if guestToken != "" {
			if authenticateWithGuestToken(c, roomSvc, movieID, guestToken) {
				c.Next()
				return
			}
		}

		logger.Warn("streaming access denied: no valid authentication provided")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required for streaming"})
		c.Abort()
	}
}

// authenticateWithJWT validates JWT token and checks room access
func authenticateWithJWT(c *gin.Context, jwtManager *auth.JWTManager, roomSvc *roomService.Service, movieID uuid.UUID) bool {
	authHeader := c.GetHeader("Authorization")
	bearerToken := strings.Split(authHeader, " ")
	if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
		return false
	}

	tokenString := bearerToken[1]
	claims, err := jwtManager.ValidateToken(tokenString)
	if err != nil {
		logger.Error(err, "invalid JWT token in streaming request")
		return false
	}

	// check if user has access to any room containing this movie
	hasAccess, err := roomSvc.CheckUserMovieAccess(context.Background(), claims.UserID, movieID)
	if err != nil {
		logger.Error(err, "failed to check user movie access")
		return false
	}

	if !hasAccess {
		logger.Warnf("user %s denied streaming access to movie %s - not in any authorized room",
			claims.UserID, movieID)
		return false
	}

	// store user info in context
	c.Set("user", claims)
	c.Set("user_id", claims.UserID)
	c.Set("user_email", claims.Email)
	c.Set("user_role", claims.Role)
	c.Set("auth_type", "jwt")

	return true
}

// authenticateWithGuestToken validates guest session and checks room access
func authenticateWithGuestToken(c *gin.Context, roomSvc *roomService.Service, movieID uuid.UUID, token string) bool {
	session, err := roomSvc.ValidateGuestSession(context.Background(), token)
	if err != nil {
		logger.Error(err, "invalid guest token in streaming request")
		return false
	}

	// check if the guest's room contains this movie
	hasAccess, err := roomSvc.CheckRoomContainsMovie(context.Background(), session.RoomID, movieID)
	if err != nil {
		logger.Error(err, "failed to check room movie access for guest")
		return false
	}

	if !hasAccess {
		logger.Warnf("guest denied streaming access to movie %s - movie not in authorized room %s",
			movieID, session.RoomID)
		return false
	}

	// store guest info in context
	c.Set("guest_session", session)
	c.Set("room_id", session.RoomID)
	c.Set("guest_name", session.GuestName)
	c.Set("auth_type", "guest")

	return true
}

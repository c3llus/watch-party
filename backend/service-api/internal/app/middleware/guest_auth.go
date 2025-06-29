package middleware

import (
	"net/http"
	"strings"
	"watch-party/pkg/auth"
	"watch-party/service-api/internal/service/room"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GuestAuthForRoom validates guest token specifically for a room ID from URL parameter
func GuestAuthForRoom(roomSvc *room.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// get room ID from URL parameter
		roomIDParam := c.Param("id")
		roomID, err := uuid.Parse(roomIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
			c.Abort()
			return
		}

		// get guest token from query parameter or header
		guestToken := c.Query("guestToken")
		if guestToken == "" {
			guestToken = c.GetHeader("X-Guest-Token")
		}

		if guestToken == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Guest token required"})
			c.Abort()
			return
		}

		// validate guest session and check room access
		guestSession, err := roomSvc.ValidateGuestSession(c.Request.Context(), guestToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired guest token"})
			c.Abort()
			return
		}

		// verify guest has access to this specific room
		if guestSession.RoomID != roomID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Guest token not valid for this room"})
			c.Abort()
			return
		}

		// set guest info in context for use by handlers
		c.Set("guestSession", guestSession)
		c.Set("roomID", guestSession.RoomID)
		c.Set("guestName", guestSession.GuestName)

		c.Next()
	}
}

// OptionalAuthMiddleware allows both JWT tokens and guest tokens
func OptionalAuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// check for guest token first
		guestToken := c.Query("guestToken")
		if guestToken != "" {
			// set guest token in context for handlers to use
			c.Set("guestToken", guestToken)
			c.Set("authType", "guest")
			c.Next()
			return
		}

		// check for JWT token
		var tokenString string
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			bearerToken := strings.Split(authHeader, " ")
			if len(bearerToken) == 2 && bearerToken[0] == "Bearer" {
				tokenString = bearerToken[1]
			}
		}

		if tokenString != "" {
			// validate JWT token
			claims, err := jwtManager.ValidateToken(tokenString)
			if err == nil {
				// set JWT claims in context
				c.Set("user", claims)
				c.Set("authType", "jwt")
				c.Next()
				return
			}
		}

		// no valid authentication found
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		c.Abort()
	}
}

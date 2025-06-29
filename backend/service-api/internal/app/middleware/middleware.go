package middleware

import (
	"watch-party/pkg/auth"

	"github.com/gin-gonic/gin"
)

// OptionalAuth middleware that allows both authenticated users and guests
func OptionalAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// try JWT authentication first
		token := c.GetHeader("Authorization")
		if token != "" {
			// use the existing JWT auth logic
			authMiddleware := auth.AuthMiddleware(jwtManager)
			authMiddleware(c)
			if c.IsAborted() {
				return
			}
		}
		// if no JWT token provided, continue without authentication
		// guest token validation will be handled in individual handlers
		c.Next()
	}
}

type MiddlewareProvider interface {
}

type middleware struct {
}

func NewMiddleware() MiddlewareProvider {
	return &middleware{}
}

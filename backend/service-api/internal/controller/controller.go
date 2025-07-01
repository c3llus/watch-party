package controller

import (
	authService "watch-party/service-api/internal/service/auth"

	"github.com/gin-gonic/gin"
)

// ControllerProvider defines the controller interface
type ControllerProvider interface {
	RegisterAdmin(c *gin.Context)
	RegisterUser(c *gin.Context)
	Login(c *gin.Context)
	Logout(c *gin.Context)
	GetProfile(c *gin.Context)
}

// controller implements the controller interface
type controller struct {
	authService authService.Service
}

// NewController creates a new controller instance
func NewController(authService authService.Service) ControllerProvider {
	return &controller{
		authService: authService,
	}
}

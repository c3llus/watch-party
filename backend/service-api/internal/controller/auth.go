package controller

import (
	"net/http"
	"watch-party/pkg/logger"
	"watch-party/pkg/model"

	"github.com/gin-gonic/gin"
)

// Login handles user authentication
func (ctrl *controller) Login(c *gin.Context) {
	var req model.LoginRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		logger.Error(err, "failed to bind login request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	response, err := ctrl.authService.Login(&req)
	if err != nil {
		logger.Error(err, "failed to login user")
		if err.Error() == "invalid credentials" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	logger.Infof("user logged in successfully: %s", response.User.Email)
	c.JSON(http.StatusOK, gin.H{
		"access_token":  response.AccessToken,
		"refresh_token": response.RefreshToken,
		"user":          response.User.ToProfile(),
	})
}

// Logout handles user logout
func (ctrl *controller) Logout(c *gin.Context) {
	type LogoutRequest struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	var req LogoutRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		logger.Error(err, "failed to bind logout request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	err = ctrl.authService.Logout(req.RefreshToken)
	if err != nil {
		logger.Error(err, "failed to logout user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	logger.Info("user logged out successfully")
	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}

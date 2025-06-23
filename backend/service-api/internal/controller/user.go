package controller

import (
	"net/http"
	"watch-party/pkg/logger"
	"watch-party/pkg/model"

	"github.com/gin-gonic/gin"
)

// RegisterUser handles user registration
func (ctrl *controller) RegisterUser(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(err, "failed to bind register request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	user, err := ctrl.authService.RegisterUser(&req)
	if err != nil {
		logger.Error(err, "failed to register user")
		if err.Error() == "user already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	logger.Infof("user registered successfully: %s", user.Email)
	c.JSON(http.StatusCreated, gin.H{
		"message": "user registered successfully",
		"user":    user.ToProfile(),
	})
}

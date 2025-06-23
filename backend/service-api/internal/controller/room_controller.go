package controller

import (
	"net/http"
	"watch-party/pkg/auth"
	"watch-party/pkg/model"
	roomService "watch-party/service-api/internal/service/room"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RoomController handles room-related HTTP requests
type RoomController struct {
	roomService *roomService.Service
}

// NewRoomController creates a new room controller
func NewRoomController(roomService *roomService.Service) *RoomController {
	return &RoomController{
		roomService: roomService,
	}
}

// CreateRoom handles POST /api/v1/rooms
func (rc *RoomController) CreateRoom(c *gin.Context) {
	// get user ID from JWT token
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, ok := userClaims.(*auth.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	// parse request
	var req model.CreateRoomRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// create room
	response, err := rc.roomService.CreateRoom(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetRoom handles GET /api/v1/rooms/:id
func (rc *RoomController) GetRoom(c *gin.Context) {
	// get user ID from JWT token
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, ok := userClaims.(*auth.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	// parse room ID from URL
	roomIDParam := c.Param("id")
	roomID, err := uuid.Parse(roomIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// get room
	room, err := rc.roomService.GetRoom(c.Request.Context(), claims.UserID, roomID)
	if err != nil {
		if err.Error() == "access denied" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, room)
}

// InviteUser handles POST /api/v1/rooms/:id/invite
func (rc *RoomController) InviteUser(c *gin.Context) {
	// get user ID from JWT token
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, ok := userClaims.(*auth.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	// parse room ID from URL
	roomIDParam := c.Param("id")
	roomID, err := uuid.Parse(roomIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// parse request
	var req model.InviteUserRequest
	err = c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// send invitation
	response, err := rc.roomService.InviteUser(c.Request.Context(), claims.UserID, roomID, &req)
	if err != nil {
		if err.Error() == "only room host can send invitations" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// JoinRoom handles POST /api/v1/rooms/join
func (rc *RoomController) JoinRoom(c *gin.Context) {
	// get user ID from JWT token
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, ok := userClaims.(*auth.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	// parse request
	var req model.JoinRoomRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// join room
	response, err := rc.roomService.JoinRoomByInvitation(c.Request.Context(), claims.UserID, &req)
	if err != nil {
		if err.Error() == "invalid invitation token" || err.Error() == "invitation has expired" || err.Error() == "invitation has already been used" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// JoinRoomByToken handles GET /api/v1/rooms/join?token=<token> for web links
func (rc *RoomController) JoinRoomByToken(c *gin.Context) {
	// get user ID from JWT token
	userClaims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, ok := userClaims.(*auth.JWTClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication token"})
		return
	}

	// get token from query parameter
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation token is required"})
		return
	}

	// join room
	req := &model.JoinRoomRequest{InviteToken: token}
	response, err := rc.roomService.JoinRoomByInvitation(c.Request.Context(), claims.UserID, req)
	if err != nil {
		if err.Error() == "invalid invitation token" || err.Error() == "invitation has expired" || err.Error() == "invitation has already been used" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

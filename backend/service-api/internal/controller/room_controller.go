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
		if err.Error() == "invalid invitation token" || err.Error() == "invitation has expired" {
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
		if err.Error() == "invalid invitation token" || err.Error() == "invitation has expired" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// JoinRoomByID handles GET /api/v1/rooms/join/{room_id}
func (rc *RoomController) JoinRoomByID(c *gin.Context) {
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
	roomIDParam := c.Param("room_id")
	roomID, err := uuid.Parse(roomIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// join room by ID
	response, err := rc.roomService.JoinRoomByID(c.Request.Context(), claims.UserID, roomID)
	if err != nil {
		if err.Error() == "access denied - you need to be invited to this room" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied - you need to be invited to this room"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Guest access endpoints

// RequestGuestAccess handles POST /api/v1/rooms/:id/request-access (no auth required)
func (rc *RoomController) RequestGuestAccess(c *gin.Context) {
	// parse room ID
	roomIDStr := c.Param("id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// parse request
	var req model.GuestAccessRequestRequest
	err = c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// create guest access request
	response, err := rc.roomService.RequestGuestAccess(c.Request.Context(), roomID, &req)
	if err != nil {
		if err.Error() == "room not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit access request"})
		return
	}

	c.JSON(http.StatusAccepted, response)
}

// GetPendingGuestRequests handles GET /api/v1/rooms/:id/guest-requests (host only)
func (rc *RoomController) GetPendingGuestRequests(c *gin.Context) {
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

	// parse room ID
	roomIDStr := c.Param("id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// get pending requests
	requests, err := rc.roomService.GetPendingGuestRequests(c.Request.Context(), claims.UserID, roomID)
	if err != nil {
		if err.Error() == "access denied - only room host can view guest requests" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only room host can view guest requests"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get guest requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

// ApproveGuestRequest handles POST /api/v1/rooms/:roomId/guest-requests/:requestId/approve (host only)
func (rc *RoomController) ApproveGuestRequest(c *gin.Context) {
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

	// parse room ID
	roomIDStr := c.Param("id")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// parse request ID
	requestIDStr := c.Param("requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// parse request body
	var req model.ApproveGuestRequest
	err = c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// approve/deny request
	response, err := rc.roomService.ApproveGuestRequest(c.Request.Context(), claims.UserID, roomID, requestID, req.Approved)
	if err != nil {
		if err.Error() == "access denied - only room host can approve guest requests" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only room host can approve guest requests"})
			return
		}
		if err.Error() == "guest request has already been reviewed" {
			c.JSON(http.StatusConflict, gin.H{"error": "Guest request has already been reviewed"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process guest request"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ValidateGuestSession handles GET /api/v1/guest/validate/:token (no auth required)
func (rc *RoomController) ValidateGuestSession(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session token required"})
		return
	}

	session, err := rc.roomService.ValidateGuestSession(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"room_id":    session.RoomID,
		"guest_name": session.GuestName,
		"expires_at": session.ExpiresAt,
	})
}

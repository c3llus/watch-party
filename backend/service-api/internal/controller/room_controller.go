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

// GetRoomForGuest handles GET /api/v1/guest/rooms/:id (guest token auth required)
func (rc *RoomController) GetRoomForGuest(c *gin.Context) {
	// parse room ID from URL
	roomIDParam := c.Param("id")
	roomID, err := uuid.Parse(roomIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// guest session is already validated by middleware
	// get room info for guest
	roomInfo, err := rc.roomService.GetRoomForGuest(c.Request.Context(), roomID)
	if err != nil {
		if err.Error() == "room not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, roomInfo)
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
		if err.Error() == "access denied - you need access to this room" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied - you need access to this room"})
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

// GetPendingGuestRequests handles GET /api/v1/rooms/:id/guest-requests (admin only)
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

	// check if user is admin
	if claims.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get guest requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

// ApproveGuestRequest handles POST /api/v1/rooms/:roomId/guest-requests/:requestId/approve (admin only)
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

	// check if user is admin
	if claims.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
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

// GetRooms handles GET /api/v1/rooms (admin only)
func (rc *RoomController) GetRooms(c *gin.Context) {
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

	// get user's rooms
	rooms, err := rc.roomService.GetUserRooms(c.Request.Context(), claims.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rooms)
}

// CheckGuestRequestStatus handles GET /api/v1/guest-requests/:requestId/status (public endpoint)
func (rc *RoomController) CheckGuestRequestStatus(c *gin.Context) {
	requestID := c.Param("requestId")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Request ID is required"})
		return
	}

	// parse request ID as UUID
	requestUUID, err := uuid.Parse(requestID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID format"})
		return
	}

	status, sessionToken, expiresAt, err := rc.roomService.CheckGuestRequestStatus(c.Request.Context(), requestUUID)
	if err != nil {
		if err.Error() == "request not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"status": status,
	}

	// include session token if approved
	if status == "approved" && sessionToken != "" {
		response["session_token"] = sessionToken
		response["expires_at"] = expiresAt
	}

	c.JSON(http.StatusOK, response)
}

// RequestRoomAccess handles POST /api/v1/rooms/:id/room-access (authenticated users)
func (rc *RoomController) RequestRoomAccess(c *gin.Context) {
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

	// parse request body
	var req model.UserRoomAccessRequestRequest
	err = c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// request room access
	response, err := rc.roomService.RequestRoomAccess(c.Request.Context(), claims.UserID, roomID, req)
	if err != nil {
		if err.Error() == "room not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
			return
		}
		if err.Error() == "user already has access to this room" {
			c.JSON(http.StatusConflict, gin.H{"error": "User already has access to this room"})
			return
		}
		if err.Error() == "user already has a pending request for this room" {
			c.JSON(http.StatusConflict, gin.H{"error": "User already has a pending request for this room"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit room access request"})
		return
	}

	c.JSON(http.StatusAccepted, response)
}

// GetPendingRoomAccessRequests handles GET /api/v1/rooms/:id/room-access (host only)
func (rc *RoomController) GetPendingRoomAccessRequests(c *gin.Context) {
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
	requests, err := rc.roomService.GetPendingRoomAccessRequests(c.Request.Context(), claims.UserID, roomID)
	if err != nil {
		if err.Error() == "access denied - only room host can view room access requests" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only room host can view room access requests"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get room access requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requests": requests})
}

// ApproveRoomAccessRequest handles POST /api/v1/rooms/:roomId/room-access/:userId/approve (host only)
func (rc *RoomController) ApproveRoomAccessRequest(c *gin.Context) {
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

	// parse user ID
	userIDStr := c.Param("userId")
	requestedUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// parse request body
	var req model.ApproveUserAccessRequest
	err = c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// approve/deny request
	response, err := rc.roomService.ApproveRoomAccessRequest(c.Request.Context(), claims.UserID, roomID, requestedUserID, req.Approved)
	if err != nil {
		if err.Error() == "access denied - only room host can approve room access requests" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only room host can approve room access requests"})
			return
		}
		if err.Error() == "room access request has already been reviewed" {
			c.JSON(http.StatusConflict, gin.H{"error": "Room access request has already been reviewed"})
			return
		}
		if err.Error() == "room access request not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Room access request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process room access request"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// CheckRoomAccessRequestStatus handles GET /api/v1/rooms/:roomId/room-access/status (authenticated users)
func (rc *RoomController) CheckRoomAccessRequestStatus(c *gin.Context) {
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

	status, err := rc.roomService.CheckRoomAccessRequestStatus(c.Request.Context(), claims.UserID, roomID)
	if err != nil {
		if err.Error() == "room access request not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "No room access request found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status,
	})
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"watch-party/pkg/logger"
	"watch-party/pkg/model"
	"watch-party/service-sync/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// SyncHandler handles HTTP requests for sync service
type SyncHandler struct {
	service  service.SyncService
	upgrader websocket.Upgrader
}

// NewSyncHandler creates a new sync handler instance
func NewSyncHandler(service service.SyncService) *SyncHandler {
	return &SyncHandler{
		service: service,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// allow all origins for development
				// in production, implement proper origin validation
				return true
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections for room synchronization
func (h *SyncHandler) HandleWebSocket(c *gin.Context) {
	// parse room ID from URL
	roomIDStr := c.Param("roomID")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	// check for guest session token first
	guestToken := c.Query("guestToken")
	var userID uuid.UUID
	var username string

	if guestToken != "" {
		// handle guest connection
		// validate guest session token with API service
		resp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/v1/guest/validate/%s", guestToken))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to validate guest session"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired guest session"})
			return
		}

		// parse validation response
		var validationResp struct {
			Valid     bool   `json:"valid"`
			RoomID    string `json:"room_id"`
			GuestName string `json:"guest_name"`
		}

		err = json.NewDecoder(resp.Body).Decode(&validationResp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse guest session"})
			return
		}

		if !validationResp.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Guest session is not valid"})
			return
		}

		// verify room ID matches
		if validationResp.RoomID != roomID.String() {
			c.JSON(http.StatusForbidden, gin.H{"error": "Guest session is for a different room"})
			return
		}

		// generate temporary UUID for guest session
		userID = uuid.New()
		username = validationResp.GuestName + " (Guest)"
	} else {
		// Handle authenticated user connection
		userIDStr := c.Query("user_id")
		if userIDStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User ID or guest token required"})
			return
		}

		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		username = c.Query("username")
		if username == "" {
			username = "Anonymous"
		}
	}

	// upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Error(err, "failed to upgrade connection to WebSocket")
		return
	}
	defer conn.Close()

	// handle the WebSocket connection
	ctx := context.Background()
	err = h.service.HandleConnection(ctx, roomID, userID, username, conn)
	if err != nil {
		logger.Error(err, "failed to handle WebSocket connection")
		// send error message to client before closing
		conn.WriteJSON(&model.WebSocketMessage{
			Type: model.MessageTypeError,
			Payload: model.ErrorMessage{
				Code:    "CONNECTION_ERROR",
				Message: err.Error(),
			},
		})
	}
}

// GetRoomState retrieves the current room state
func (h *SyncHandler) GetRoomState(c *gin.Context) {
	// parse room ID from URL
	roomIDStr := c.Param("roomID")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	ctx := context.Background()
	state, err := h.service.GetRoomState(ctx, roomID)
	if err != nil {
		logger.Error(err, "failed to get room state")
		c.JSON(http.StatusNotFound, gin.H{"error": "Room sync session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"state": state,
	})
}

// GetRoomParticipants retrieves room participants
func (h *SyncHandler) GetRoomParticipants(c *gin.Context) {
	// parse room ID from URL
	roomIDStr := c.Param("roomID")
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	ctx := context.Background()
	participants, err := h.service.GetRoomParticipants(ctx, roomID)
	if err != nil {
		logger.Error(err, "failed to get room participants")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get participants"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"participants": participants,
		"count":        len(participants),
	})
}

// helper functions for authentication/authorization
// in production, these would be middleware

func (h *SyncHandler) getUserFromToken(c *gin.Context) (uuid.UUID, string, error) {
	// extract JWT token from Authorization header
	// validate and parse token
	// return user ID and username
	// placeholder implementation
	userIDStr := c.GetHeader("X-User-ID")
	username := c.GetHeader("X-Username")

	if userIDStr == "" {
		return uuid.Nil, "", fmt.Errorf("user ID not found")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid user ID")
	}

	if username == "" {
		username = "Anonymous"
	}

	return userID, username, nil
}

func (h *SyncHandler) extractPaginationParams(c *gin.Context) (int, int) {
	page := 1
	limit := 50

	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	return page, limit
}

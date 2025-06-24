package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"watch-party/pkg/logger"
	"watch-party/pkg/model"
	"watch-party/pkg/redis"
	"watch-party/service-sync/internal/repository"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// SyncService defines the interface for sync service operations
type SyncService interface {
	// websocket operations
	HandleConnection(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn) error
	BroadcastSync(ctx context.Context, message *model.SyncMessage) error

	// participant operations
	JoinRoom(ctx context.Context, roomID, userID uuid.UUID, username string) error
	LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error

	// state synchronization
	SyncAction(ctx context.Context, message *model.SyncMessage) error
	GetRoomState(ctx context.Context, roomID uuid.UUID) (*model.RoomState, error)
	GetRoomParticipants(ctx context.Context, roomID uuid.UUID) ([]model.ParticipantInfo, error)
}

type syncService struct {
	syncRepo    repository.SyncRepository // Redis repo for real-time sync state
	redis       *redis.Client
	connections map[uuid.UUID]map[uuid.UUID]*websocket.Conn // roomID -> userID -> connection
	connMutex   sync.RWMutex
}

// NewSyncService creates a new sync service instance
func NewSyncService(syncRepo repository.SyncRepository, redisClient *redis.Client) SyncService {
	service := &syncService{
		syncRepo:    syncRepo,
		redis:       redisClient,
		connections: make(map[uuid.UUID]map[uuid.UUID]*websocket.Conn),
	}

	// start Redis subscription handler
	go service.handleRedisMessages()

	return service
}

// GetRoomState retrieves the current room state
func (s *syncService) GetRoomState(ctx context.Context, roomID uuid.UUID) (*model.RoomState, error) {
	state, err := s.syncRepo.GetRoomState(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room state: %w", err)
	}
	return state, nil
}

// GetRoomParticipants retrieves room participants
func (s *syncService) GetRoomParticipants(ctx context.Context, roomID uuid.UUID) ([]model.ParticipantInfo, error) {
	participants, err := s.syncRepo.GetParticipants(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}
	return participants, nil
}

// HandleConnection handles a new WebSocket connection
func (s *syncService) HandleConnection(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn) error {
	// add connection to map
	s.addConnection(roomID, userID, conn)
	defer s.removeConnection(roomID, userID)

	// add participant to room
	err := s.JoinRoom(ctx, roomID, userID, username)
	if err != nil {
		logger.Error(err, "failed to join room")
	}

	// send current room state to the new connection
	state, err := s.GetRoomState(ctx, roomID)
	if err == nil {
		s.sendToConnection(conn, &model.WebSocketMessage{
			Type:    model.MessageTypeState,
			Payload: state,
		})
	}

	// send current participants list
	participants, err := s.GetRoomParticipants(ctx, roomID)
	if err == nil {
		s.sendToConnection(conn, &model.WebSocketMessage{
			Type:    model.MessageTypeParticipants,
			Payload: participants,
		})
	}

	// handle incoming messages
	s.handleConnectionMessages(ctx, roomID, userID, username, conn)

	return nil
}

// JoinRoom adds a user to a room
func (s *syncService) JoinRoom(ctx context.Context, roomID, userID uuid.UUID, username string) error {
	// create participant info
	participant := &model.ParticipantInfo{
		UserID:      userID,
		Username:    username,
		IsHost:      false, // will be determined by room state or host logic
		JoinedAt:    time.Now(),
		LastSeen:    time.Now(),
		IsBuffering: false,
	}

	// add participant to Redis
	err := s.syncRepo.AddParticipant(ctx, roomID, userID, participant)
	if err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}

	// set user presence
	err = s.syncRepo.SetUserPresence(ctx, userID, roomID, "active")
	if err != nil {
		logger.Error(err, "failed to set user presence")
	}

	// broadcast join event
	joinMessage := &model.SyncMessage{
		ID:        uuid.New(),
		RoomID:    roomID,
		UserID:    userID,
		Username:  username,
		Action:    model.ActionJoin,
		Timestamp: time.Now(),
	}

	s.BroadcastSync(ctx, joinMessage)

	logger.Infof("user %s joined room %s", username, roomID)
	return nil
}

// LeaveRoom removes a user from a room
func (s *syncService) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	// remove participant from Redis
	err := s.syncRepo.RemoveParticipant(ctx, roomID, userID)
	if err != nil {
		logger.Error(err, "failed to remove participant")
	}

	// remove user presence
	err = s.syncRepo.RemoveUserPresence(ctx, userID)
	if err != nil {
		logger.Error(err, "failed to remove user presence")
	}

	// broadcast leave event
	leaveMessage := &model.SyncMessage{
		ID:        uuid.New(),
		RoomID:    roomID,
		UserID:    userID,
		Action:    model.ActionLeave,
		Timestamp: time.Now(),
	}

	s.BroadcastSync(ctx, leaveMessage)

	logger.Infof("user %s left room %s", userID, roomID)
	return nil
}

// SyncAction processes a sync action (play, pause, seek, etc.)
func (s *syncService) SyncAction(ctx context.Context, message *model.SyncMessage) error {
	// acquire lock to prevent conflicts
	acquired, err := s.syncRepo.AcquireRoomLock(ctx, message.RoomID, message.UserID)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("room is locked by another user")
	}
	defer s.syncRepo.ReleaseRoomLock(ctx, message.RoomID)

	// get current room state
	state, err := s.syncRepo.GetRoomState(ctx, message.RoomID)
	if err != nil {
		// create initial state if not exists
		state = &model.RoomState{
			RoomID:       message.RoomID,
			IsPlaying:    false,
			CurrentTime:  0.0,
			Duration:     message.Data.Duration,
			PlaybackRate: 1.0,
			LastUpdated:  time.Now(),
			UpdatedBy:    message.UserID,
		}
	}

	// update state based on action
	switch message.Action {
	case model.ActionPlay:
		state.IsPlaying = true
		if message.Data.CurrentTime > 0 {
			state.CurrentTime = message.Data.CurrentTime
		}
	case model.ActionPause:
		state.IsPlaying = false
		if message.Data.CurrentTime > 0 {
			state.CurrentTime = message.Data.CurrentTime
		}
	case model.ActionSeek:
		state.CurrentTime = message.Data.CurrentTime
	}

	// update other fields
	if message.Data.PlaybackRate > 0 {
		state.PlaybackRate = message.Data.PlaybackRate
	}
	state.LastUpdated = time.Now()
	state.UpdatedBy = message.UserID

	// save updated state
	err = s.syncRepo.SetRoomState(ctx, state)
	if err != nil {
		return fmt.Errorf("failed to update room state: %w", err)
	}

	// update participant presence
	s.syncRepo.UpdateParticipantPresence(ctx, message.RoomID, message.UserID)

	// broadcast the sync message
	s.BroadcastSync(ctx, message)

	return nil
}

// BroadcastSync broadcasts a sync message to all room participants
func (s *syncService) BroadcastSync(ctx context.Context, message *model.SyncMessage) error {
	// publish to Redis for cross-instance communication
	err := s.syncRepo.PublishEvent(ctx, message.RoomID, message)
	if err != nil {
		logger.Error(err, "failed to publish event to Redis")
	}

	// broadcast to local connections
	s.broadcastToRoom(message.RoomID, &model.WebSocketMessage{
		Type:    model.MessageTypeSync,
		Payload: message,
	})

	return nil
}

// Connection management helpers
func (s *syncService) addConnection(roomID, userID uuid.UUID, conn *websocket.Conn) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	if s.connections[roomID] == nil {
		s.connections[roomID] = make(map[uuid.UUID]*websocket.Conn)
	}
	s.connections[roomID][userID] = conn
}

func (s *syncService) removeConnection(roomID, userID uuid.UUID) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	if roomConns, exists := s.connections[roomID]; exists {
		delete(roomConns, userID)
		if len(roomConns) == 0 {
			delete(s.connections, roomID)
		}
	}
}

func (s *syncService) broadcastToRoom(roomID uuid.UUID, message *model.WebSocketMessage) {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	if roomConns, exists := s.connections[roomID]; exists {
		for userID, conn := range roomConns {
			if err := s.sendToConnection(conn, message); err != nil {
				logger.Errorf(err, "failed to send message to user %s", userID)
			}
		}
	}
}

func (s *syncService) sendToConnection(conn *websocket.Conn, message *model.WebSocketMessage) error {
	return conn.WriteJSON(message)
}

// handleConnectionMessages handles incoming WebSocket messages from a connection
func (s *syncService) handleConnectionMessages(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn) {
	defer func() {
		s.LeaveRoom(ctx, roomID, userID)
		conn.Close()
	}()

	for {
		var message model.SyncMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf(err, "websocket error for user %s in room %s", userID, roomID)
			}
			break
		}

		// set message metadata
		message.ID = uuid.New()
		message.RoomID = roomID
		message.UserID = userID
		message.Username = username
		message.Timestamp = time.Now()

		// process the sync action
		if err := s.SyncAction(ctx, &message); err != nil {
			logger.Error(err, "failed to process sync action")
			// send error back to client
			errorMsg := &model.WebSocketMessage{
				Type: model.MessageTypeError,
				Payload: &model.ErrorMessage{
					Code:    "SYNC_ERROR",
					Message: err.Error(),
				},
			}
			s.sendToConnection(conn, errorMsg)
		}

		// update participant presence
		s.syncRepo.UpdateParticipantPresence(ctx, roomID, userID)
	}
}

// handleRedisMessages handles Redis pub/sub messages for cross-instance sync
func (s *syncService) handleRedisMessages() {
	// this would subscribe to all room events, but for now we'll implement
	// room-specific subscriptions when connections are established
	logger.Info("Redis message handler started")

	// TODO: Implement proper Redis subscription handling
	// This would typically involve:
	// 1. Subscribing to a global sync channel
	// 2. Processing incoming sync events
	// 3. Broadcasting to local connections
}

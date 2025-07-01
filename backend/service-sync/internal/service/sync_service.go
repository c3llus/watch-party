package service

import (
	"context"
	"encoding/json"
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
	syncRepo    repository.SyncRepository
	redis       *redis.Client
	connections map[uuid.UUID]map[uuid.UUID]*websocket.Conn
	connMutex   sync.RWMutex
	// per-connection mutexes to prevent concurrent writes to WebSocket connections
	connWriteMutexes map[uuid.UUID]map[uuid.UUID]*sync.Mutex
	writeMutexLock   sync.RWMutex
}

// NewSyncService creates a new sync service instance
func NewSyncService(syncRepo repository.SyncRepository, redisClient *redis.Client) SyncService {
	service := &syncService{
		syncRepo:         syncRepo,
		redis:            redisClient,
		connections:      make(map[uuid.UUID]map[uuid.UUID]*websocket.Conn),
		connWriteMutexes: make(map[uuid.UUID]map[uuid.UUID]*sync.Mutex),
	}

	// start Redis subscription handler
	go service.handleRedisMessages()

	return service
}

// GetRoomState retrieves the current room state
func (s *syncService) GetRoomState(ctx context.Context, roomID uuid.UUID) (*model.RoomState, error) {
	state, err := s.syncRepo.GetRoomState(ctx, roomID)
	if err != nil {
		defaultState := &model.RoomState{
			RoomID:       roomID,
			IsPlaying:    false,
			CurrentTime:  0.0,
			Duration:     0.0,
			PlaybackRate: 1.0,
			LastUpdated:  time.Now(),
			UpdatedBy:    uuid.Nil,
		}

		if saveErr := s.syncRepo.SetRoomState(ctx, defaultState); saveErr != nil {
			logger.Error(saveErr, "failed to save default room state")
		}

		return defaultState, nil
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
	logger.Infof("new connection: user %s (%s) joining room %s", username, userID, roomID)

	// check existing connections BEFORE adding this user
	s.connMutex.RLock()
	existingConns := 0
	if roomConns, exists := s.connections[roomID]; exists {
		existingConns = len(roomConns)
	}
	s.connMutex.RUnlock()

	logger.Infof("room %s has %d existing connections before adding new user", roomID, existingConns)

	// now add the new connection
	s.addConnection(roomID, userID, conn)
	defer s.removeConnection(roomID, userID)

	err := s.JoinRoom(ctx, roomID, userID, username)
	if err != nil {
		logger.Error(err, "failed to join room")
	}

	if existingConns > 0 {
		// other users exist, request live state from first connected user
		logger.Infof("requesting live state for new user %s from existing users in room %s", username, roomID)
		s.requestLiveStateFromExistingUser(ctx, roomID, userID, conn)
	} else {
		// first user in room, send stored state
		logger.Infof("sending stored state to first user %s in room %s", username, roomID)
		state, err := s.GetRoomState(ctx, roomID)
		if err == nil {
			logger.Infof("sending stored room state: playing=%v, time=%.2f", state.IsPlaying, state.CurrentTime)
			if err := s.sendToConnectionSafe(roomID, userID, conn, &model.WebSocketMessage{
				Type:    model.MessageTypeState,
				Payload: state,
			}); err != nil {
				logger.Error(err, "failed to send room state")
			}
		} else {
			logger.Error(err, "failed to get stored room state")
		}
	}

	// get participants and send the list
	participants, err := s.GetRoomParticipants(ctx, roomID)
	if err == nil {
		logger.Infof("room %s now has %d total participants", roomID, len(participants))
		for i, p := range participants {
			logger.Infof("participant %d: %s (%s)", i+1, p.Username, p.UserID)
		}
		if err := s.sendToConnectionSafe(roomID, userID, conn, &model.WebSocketMessage{
			Type:    model.MessageTypeParticipants,
			Payload: participants,
		}); err != nil {
			logger.Error(err, "failed to send participants list")
		}
	} else {
		logger.Error(err, "failed to get room participants")
	}

	s.handleConnectionMessages(ctx, roomID, userID, username, conn)

	return nil
}

// JoinRoom adds a user to a room
func (s *syncService) JoinRoom(ctx context.Context, roomID, userID uuid.UUID, username string) error {
	participant := &model.ParticipantInfo{
		UserID:      userID,
		Username:    username,
		IsHost:      false,
		JoinedAt:    time.Now(),
		LastSeen:    time.Now(),
		IsBuffering: false,
	}

	err := s.syncRepo.AddParticipant(ctx, roomID, userID, participant)
	if err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}

	err = s.syncRepo.SetUserPresence(ctx, userID, roomID, "active")
	if err != nil {
		logger.Error(err, "failed to set user presence")
	}

	joinMessage := &model.SyncMessage{
		ID:        uuid.New(),
		RoomID:    roomID,
		UserID:    userID,
		Username:  username,
		Action:    model.ActionJoin,
		Timestamp: time.Now(),
	}

	// add to user logs - no longer needed, handled in frontend
	// s.addUserLog(joinMessage)

	s.BroadcastSync(ctx, joinMessage)

	logger.Infof("user %s joined room %s", username, roomID)
	return nil
}

// LeaveRoom removes a user from a room
func (s *syncService) LeaveRoom(ctx context.Context, roomID, userID uuid.UUID) error {
	err := s.syncRepo.RemoveParticipant(ctx, roomID, userID)
	if err != nil {
		logger.Error(err, "failed to remove participant")
	}

	err = s.syncRepo.RemoveUserPresence(ctx, userID)
	if err != nil {
		logger.Error(err, "failed to remove user presence")
	}

	leaveMessage := &model.SyncMessage{
		ID:        uuid.New(),
		RoomID:    roomID,
		UserID:    userID,
		Action:    model.ActionLeave,
		Timestamp: time.Now(),
	}

	// add to user logs - no longer needed, handled in frontend
	// s.addUserLog(leaveMessage)

	s.BroadcastSync(ctx, leaveMessage)

	logger.Infof("user %s left room %s", userID, roomID)
	return nil
}

// SyncAction processes a sync action (play, pause, seek, etc.)
func (s *syncService) SyncAction(ctx context.Context, message *model.SyncMessage) error {
	logger.Infof("📥 PROCESSING SYNC ACTION: %s from user %s in room %s (time: %.2f)",
		message.Action, message.Username, message.RoomID, message.Data.CurrentTime)

	acquired, err := s.syncRepo.AcquireRoomLock(ctx, message.RoomID, message.UserID)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("room is locked by another user")
	}
	defer s.syncRepo.ReleaseRoomLock(ctx, message.RoomID)

	state, err := s.syncRepo.GetRoomState(ctx, message.RoomID)
	if err != nil {
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

	if message.Data.PlaybackRate > 0 {
		state.PlaybackRate = message.Data.PlaybackRate
	}
	state.LastUpdated = time.Now()
	state.UpdatedBy = message.UserID

	err = s.syncRepo.SetRoomState(ctx, state)
	if err != nil {
		return fmt.Errorf("failed to update room state: %w", err)
	}

	s.syncRepo.UpdateParticipantPresence(ctx, message.RoomID, message.UserID)

	// add to user logs - no longer needed, handled in frontend
	// s.addUserLog(message)

	s.BroadcastSync(ctx, message)

	return nil
}

// BroadcastSync broadcasts a sync message to all room participants
func (s *syncService) BroadcastSync(ctx context.Context, message *model.SyncMessage) error {
	logger.Infof("📤 BROADCASTING SYNC: %s from user %s to room %s (time: %.2f)",
		message.Action, message.Username, message.RoomID, message.Data.CurrentTime)

	err := s.syncRepo.PublishEvent(ctx, message.RoomID, message)
	if err != nil {
		logger.Error(err, "failed to publish event to Redis")
		s.broadcastSyncToRoom(message.RoomID, message, message.UserID)
	}

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

	// also initialize write mutex for this connection
	s.writeMutexLock.Lock()
	if s.connWriteMutexes[roomID] == nil {
		s.connWriteMutexes[roomID] = make(map[uuid.UUID]*sync.Mutex)
	}
	s.connWriteMutexes[roomID][userID] = &sync.Mutex{}
	s.writeMutexLock.Unlock()
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

	// also clean up write mutex
	s.writeMutexLock.Lock()
	if roomMutexes, exists := s.connWriteMutexes[roomID]; exists {
		delete(roomMutexes, userID)
		if len(roomMutexes) == 0 {
			delete(s.connWriteMutexes, roomID)
		}
	}
	s.writeMutexLock.Unlock()
}

func (s *syncService) broadcastToRoom(roomID uuid.UUID, message *model.WebSocketMessage) {
	s.connMutex.RLock()
	roomConnections, exists := s.connections[roomID]
	s.connMutex.RUnlock()

	if !exists {
		return
	}

	for userID, conn := range roomConnections {
		go func(userID uuid.UUID, conn *websocket.Conn) {
			select {
			case <-time.After(100 * time.Millisecond):
				logger.Warnf("timeout sending message to user %s", userID)
			default:
				if err := s.sendToConnectionSafe(roomID, userID, conn, message); err != nil {
					logger.Errorf(err, "failed to send message to user %s", userID)
				}
			}
		}(userID, conn)
	}
}

// broadcastSyncToRoom broadcasts a sync message to all room participants in the frontend-expected format
func (s *syncService) broadcastSyncToRoom(roomID uuid.UUID, syncMessage *model.SyncMessage, excludeUserID uuid.UUID) {
	logger.Infof("📤 SENDING SYNC to room %s: %s from user %s (excluding %s)",
		roomID, syncMessage.Action, syncMessage.Username, excludeUserID)

	frontendSyncData := map[string]interface{}{
		"action":       string(syncMessage.Action),
		"current_time": syncMessage.Data.CurrentTime,
		"timestamp":    syncMessage.Timestamp.Format(time.RFC3339),
		"user_id":      syncMessage.UserID.String(),
		"username":     syncMessage.Username,
	}

	// include data object if there's additional data (like chat messages)
	if syncMessage.Data.ChatMessage != "" || syncMessage.Data.Duration > 0 || syncMessage.Data.PlaybackRate > 0 || syncMessage.Data.IsBuffering {
		frontendSyncData["data"] = map[string]interface{}{
			"current_time":  syncMessage.Data.CurrentTime,
			"duration":      syncMessage.Data.Duration,
			"playback_rate": syncMessage.Data.PlaybackRate,
			"is_buffering":  syncMessage.Data.IsBuffering,
			"chat_message":  syncMessage.Data.ChatMessage,
		}
	}

	webSocketMessage := &model.WebSocketMessage{
		Type:    model.MessageTypeSync,
		Payload: frontendSyncData,
	}

	s.broadcastToRoomExcluding(roomID, webSocketMessage, excludeUserID)
}

func (s *syncService) broadcastToRoomExcluding(roomID uuid.UUID, message *model.WebSocketMessage, excludeUserID uuid.UUID) {
	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	if roomConns, exists := s.connections[roomID]; exists {
		for userID, conn := range roomConns {
			if userID == excludeUserID {
				continue
			}
			go func(userID uuid.UUID, conn *websocket.Conn) {
				if err := s.sendToConnectionSafe(roomID, userID, conn, message); err != nil {
					logger.Errorf(err, "failed to send message to user %s", userID)
				}
			}(userID, conn)
		}
	}
}

func (s *syncService) sendToConnection(conn *websocket.Conn, message *model.WebSocketMessage) error {
	return conn.WriteJSON(message)
}

// sendToConnectionSafe sends a message to a specific connection with proper synchronization
func (s *syncService) sendToConnectionSafe(roomID, userID uuid.UUID, conn *websocket.Conn, message *model.WebSocketMessage) error {
	// get the write mutex for this specific connection
	s.writeMutexLock.RLock()
	var writeMutex *sync.Mutex
	if s.connWriteMutexes[roomID] != nil {
		writeMutex = s.connWriteMutexes[roomID][userID]
	}
	s.writeMutexLock.RUnlock()

	// if we have a mutex, use it to serialize writes
	if writeMutex != nil {
		writeMutex.Lock()
		defer writeMutex.Unlock()
	}

	return conn.WriteJSON(message)
}

func (s *syncService) sendErrorToConnection(conn *websocket.Conn, code, message string) {
	errorMsg := &model.WebSocketMessage{
		Type: model.MessageTypeError,
		Payload: &model.ErrorMessage{
			Code:    code,
			Message: message,
		},
	}
	// use the basic sendToConnection for error messages
	// errors are typically sent in response to immediate requests, so concurrency is less likely
	s.sendToConnection(conn, errorMsg)
}

// sendErrorToConnectionSafe sends error message with proper synchronization when IDs are available
func (s *syncService) sendErrorToConnectionSafe(roomID, userID uuid.UUID, conn *websocket.Conn, code, message string) {
	errorMsg := &model.WebSocketMessage{
		Type: model.MessageTypeError,
		Payload: &model.ErrorMessage{
			Code:    code,
			Message: message,
		},
	}
	if err := s.sendToConnectionSafe(roomID, userID, conn, errorMsg); err != nil {
		logger.Errorf(err, "failed to send error message to user %s", userID)
	}
}

// handleConnectionMessages handles incoming WebSocket messages from a connection
func (s *syncService) handleConnectionMessages(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn) {
	defer func() {
		s.LeaveRoom(ctx, roomID, userID)
		conn.Close()
	}()

	for {
		rawMessage, err := s.readWebSocketMessage(conn, userID, roomID)
		if err != nil {
			break
		}

		logger.Infof("📥 RECEIVED MESSAGE from user %s in room %s: %+v", username, roomID, rawMessage)

		s.processWebSocketMessage(ctx, roomID, userID, username, conn, rawMessage)
		s.syncRepo.UpdateParticipantPresence(ctx, roomID, userID)
	}
}

// readWebSocketMessage reads and validates incoming websocket message
func (s *syncService) readWebSocketMessage(conn *websocket.Conn, userID, roomID uuid.UUID) (map[string]interface{}, error) {
	var rawMessage map[string]interface{}
	err := conn.ReadJSON(&rawMessage)
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			logger.Errorf(err, "websocket error for user %s in room %s", userID, roomID)
		}
		return nil, err
	}
	return rawMessage, nil
}

// processWebSocketMessage routes and processes different message types
func (s *syncService) processWebSocketMessage(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn, rawMessage map[string]interface{}) {
	// check for special message types first
	if msgType, hasType := rawMessage["type"].(string); hasType {
		switch msgType {
		case "provide_state":
			s.handleProvideState(ctx, roomID, userID, username, conn, rawMessage)
			return
		case "request_state":
			s.handleRequestState(ctx, roomID, userID, username, conn, rawMessage)
			return
		}
	}

	// if no special type, treat as direct sync message (unified format)
	s.handleDirectSyncMessage(ctx, roomID, userID, username, conn, rawMessage)
}

// handleLegacySyncAction processes legacy frontend sync_action format
func (s *syncService) handleLegacySyncAction(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn, rawMessage map[string]interface{}) {
	data, hasData := rawMessage["data"].(map[string]interface{})
	if !hasData {
		logger.Warnf("sync_action message missing data field from user %s", username)
		return
	}

	action, hasAction := data["action"].(string)
	if !hasAction {
		logger.Warnf("sync_action message missing action field from user %s", username)
		return
	}

	message := s.createSyncMessage(roomID, userID, username, action)

	// extract current time from legacy format
	if currentTime, ok := data["currentTime"].(float64); ok {
		message.Data.CurrentTime = currentTime
	}

	s.executeSyncAction(ctx, conn, &message)
}

// handleDirectSyncMessage processes direct sync message format
func (s *syncService) handleDirectSyncMessage(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn, rawMessage map[string]interface{}) {
	action, hasAction := rawMessage["action"].(string)
	if !hasAction {
		logger.Warnf("direct sync message missing action field from user %s", username)
		return
	}

	message := s.createSyncMessage(roomID, userID, username, action)

	// extract data from direct format
	if data, ok := rawMessage["data"].(map[string]interface{}); ok {
		if currentTime, ok := data["current_time"].(float64); ok {
			message.Data.CurrentTime = currentTime
		}
		if chatMessage, ok := data["chat_message"].(string); ok {
			message.Data.ChatMessage = chatMessage
		}
	}

	// all actions (including chat) are handled as sync actions
	s.executeSyncAction(ctx, conn, &message)
}

// createSyncMessage creates a new sync message with common fields
func (s *syncService) createSyncMessage(roomID, userID uuid.UUID, username, action string) model.SyncMessage {
	return model.SyncMessage{
		ID:        uuid.New(),
		RoomID:    roomID,
		UserID:    userID,
		Username:  username,
		Timestamp: time.Now(),
		Action:    model.SyncAction(action),
		Data:      model.SyncData{}, // initialize empty data struct
	}
}

// executeSyncAction processes the sync action and handles errors
func (s *syncService) executeSyncAction(ctx context.Context, conn *websocket.Conn, message *model.SyncMessage) {
	err := s.SyncAction(ctx, message)
	if err != nil {
		logger.Error(err, "failed to process sync action")
		s.sendErrorToConnection(conn, "SYNC_ERROR", err.Error())
	}
}

// requestLiveStateFromExistingUser requests current video state from the first connected user (any existing participant)
func (s *syncService) requestLiveStateFromExistingUser(ctx context.Context, roomID, requesterID uuid.UUID, requesterConn *websocket.Conn) {
	logger.Infof("looking for existing participants to request live state from in room %s", roomID)

	s.connMutex.RLock()
	defer s.connMutex.RUnlock()

	roomConns, exists := s.connections[roomID]
	if !exists || len(roomConns) == 0 {
		logger.Warnf("no connections found for room %s", roomID)
		s.sendErrorToConnection(requesterConn, "NO_PARTICIPANTS", "No other participants in room")
		return
	}

	logger.Infof("room %s has %d active connections", roomID, len(roomConns))

	// find the first connected user (any existing participant) excluding the requester
	var sourceUserID uuid.UUID
	var sourceConn *websocket.Conn
	for userID, conn := range roomConns {
		if userID != requesterID {
			sourceUserID = userID
			sourceConn = conn
			logger.Infof("found existing participant %s to request state from", userID)
			break
		}
	}

	if sourceConn == nil {
		// if no other users, fall back to stored state
		logger.Warnf("no other users found in room %s, falling back to stored state", roomID)
		s.sendStoredRoomStateSafe(ctx, roomID, requesterID, requesterConn)
		return
	}

	// store the pending request so we can route the response back
	s.storePendingStateRequest(roomID, requesterID, requesterConn)

	// request current state from existing participant
	stateRequestMsg := &model.WebSocketMessage{
		Type: model.MessageTypeRequestState,
		Payload: map[string]interface{}{
			"requester_id": requesterID.String(),
		},
	}

	logger.Infof("requesting live state from participant %s for new participant %s in room %s", sourceUserID, requesterID, roomID)
	if err := s.sendToConnectionSafe(roomID, sourceUserID, sourceConn, stateRequestMsg); err != nil {
		logger.Error(err, "failed to send state request to existing participant")
		s.sendStoredRoomStateSafe(ctx, roomID, requesterID, requesterConn)
	} else {
		logger.Infof("state request sent successfully to participant %s", sourceUserID)
	}
}

// handleExistingUserStateResponse processes state response from existing participant and forwards to requesting user
func (s *syncService) handleExistingUserStateResponse(ctx context.Context, roomID, sourceUserID uuid.UUID, rawMessage map[string]interface{}) {
	logger.Infof("received state response from participant %s in room %s", sourceUserID, roomID)

	// extract the requester ID and state data
	requesterIDStr, ok := rawMessage["requester_id"].(string)
	if !ok {
		logger.Error(nil, "invalid provide_state message: missing requester_id")
		return
	}

	requesterID, err := uuid.Parse(requesterIDStr)
	if err != nil {
		logger.Error(err, "invalid requester_id in provide_state message")
		return
	}

	stateData, ok := rawMessage["state"].(map[string]interface{})
	if !ok {
		logger.Error(nil, "invalid provide_state message: missing state data")
		return
	}

	// log the received state data
	if currentTime, exists := stateData["current_time"]; exists {
		logger.Infof("received live state from %s: current_time=%.2f", sourceUserID, currentTime)
	}
	if isPlaying, exists := stateData["is_playing"]; exists {
		logger.Infof("received live state from %s: is_playing=%v", sourceUserID, isPlaying)
	}

	// get the pending request
	requesterConn := s.getPendingStateRequest(roomID, requesterID)
	if requesterConn == nil {
		logger.Warnf("no pending state request found for user %s in room %s", requesterID, roomID)
		return
	}

	// forward the state to the requesting user
	stateMsg := &model.WebSocketMessage{
		Type:    model.MessageTypeState,
		Payload: stateData,
	}

	logger.Infof("forwarding live state from %s to %s in room %s", sourceUserID, requesterID, roomID)
	if err := s.sendToConnectionSafe(roomID, requesterID, requesterConn, stateMsg); err != nil {
		logger.Error(err, "failed to send state to requesting user")
	} else {
		logger.Infof("live state successfully forwarded to %s", requesterID)
	}

	// clean up the pending request
	s.removePendingStateRequest(roomID, requesterID)
}

// sendStoredRoomState sends the stored room state as fallback
func (s *syncService) sendStoredRoomState(ctx context.Context, roomID uuid.UUID, conn *websocket.Conn) {
	state, err := s.GetRoomState(ctx, roomID)
	if err == nil {
		s.sendToConnection(conn, &model.WebSocketMessage{
			Type:    model.MessageTypeState,
			Payload: state,
		})
	} else {
		s.sendErrorToConnection(conn, "STATE_ERROR", "Failed to get room state")
	}
}

// sendStoredRoomStateSafe sends the stored room state as fallback with proper synchronization
func (s *syncService) sendStoredRoomStateSafe(ctx context.Context, roomID, userID uuid.UUID, conn *websocket.Conn) {
	state, err := s.GetRoomState(ctx, roomID)
	if err == nil {
		if err := s.sendToConnectionSafe(roomID, userID, conn, &model.WebSocketMessage{
			Type:    model.MessageTypeState,
			Payload: state,
		}); err != nil {
			logger.Error(err, "failed to send stored room state")
		}
	} else {
		s.sendErrorToConnectionSafe(roomID, userID, conn, "STATE_ERROR", "Failed to get room state")
	}
}

// pending state request management
var pendingStateRequests = make(map[string]map[uuid.UUID]*websocket.Conn)
var pendingRequestsMutex sync.RWMutex

func (s *syncService) storePendingStateRequest(roomID, requesterID uuid.UUID, conn *websocket.Conn) {
	pendingRequestsMutex.Lock()
	defer pendingRequestsMutex.Unlock()

	roomKey := roomID.String()
	if pendingStateRequests[roomKey] == nil {
		pendingStateRequests[roomKey] = make(map[uuid.UUID]*websocket.Conn)
	}
	pendingStateRequests[roomKey][requesterID] = conn

	// set a timeout to clean up stale requests
	go func() {
		time.Sleep(10 * time.Second)
		s.removePendingStateRequest(roomID, requesterID)
	}()
}

func (s *syncService) getPendingStateRequest(roomID, requesterID uuid.UUID) *websocket.Conn {
	pendingRequestsMutex.RLock()
	defer pendingRequestsMutex.RUnlock()

	roomKey := roomID.String()
	if pendingStateRequests[roomKey] != nil {
		return pendingStateRequests[roomKey][requesterID]
	}
	return nil
}

func (s *syncService) removePendingStateRequest(roomID, requesterID uuid.UUID) {
	pendingRequestsMutex.Lock()
	defer pendingRequestsMutex.Unlock()

	roomKey := roomID.String()
	if pendingStateRequests[roomKey] != nil {
		delete(pendingStateRequests[roomKey], requesterID)
		if len(pendingStateRequests[roomKey]) == 0 {
			delete(pendingStateRequests, roomKey)
		}
	}
}

// handleRedisMessages handles Redis pub/sub messages for cross-instance sync
func (s *syncService) handleRedisMessages() {
	ctx := context.Background()

	pubsub := s.redis.PSubscribe(ctx, "room:*:events")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		var syncMessage model.SyncMessage
		if err := json.Unmarshal([]byte(msg.Payload), &syncMessage); err != nil {
			logger.Errorf(err, "failed to unmarshal sync message from Redis")
			continue
		}

		s.connMutex.RLock()
		roomConnections, hasRoom := s.connections[syncMessage.RoomID]
		connectionCount := 0
		if hasRoom {
			connectionCount = len(roomConnections)
		}
		s.connMutex.RUnlock()

		if hasRoom && connectionCount > 0 {
			// broadcast all actions (including chat) as sync messages
			s.broadcastSyncToRoom(syncMessage.RoomID, &syncMessage, syncMessage.UserID)
		}
	}
}

// handleProvideState processes provide_state messages from existing users
func (s *syncService) handleProvideState(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn, rawMessage map[string]interface{}) {
	logger.Infof("processing provide_state message from user %s", username)

	requesterIDStr, hasRequesterID := rawMessage["requester_id"].(string)
	if !hasRequesterID {
		logger.Warnf("provide_state message missing requester_id from user %s", username)
		return
	}

	requesterID, err := uuid.Parse(requesterIDStr)
	if err != nil {
		logger.Errorf(err, "invalid requester_id in provide_state message from user %s", username)
		return
	}

	state, hasState := rawMessage["state"].(map[string]interface{})
	if !hasState {
		logger.Warnf("provide_state message missing state from user %s", username)
		return
	}

	// find the requester's connection and send them the state
	s.connMutex.RLock()
	requesterConn, exists := s.findConnection(roomID, requesterID)
	s.connMutex.RUnlock()

	if !exists || requesterConn == nil {
		logger.Warnf("requester %s not found or connection is nil for provide_state", requesterID)
		return
	}

	// send state directly to the requester
	stateMessage := &model.WebSocketMessage{
		Type:    model.MessageTypeState,
		Payload: state,
	}

	err = s.sendToConnectionSafe(roomID, requesterID, requesterConn, stateMessage)
	if err != nil {
		logger.Errorf(err, "failed to send state to requester %s", requesterID)
		return
	}

	logger.Infof("successfully sent state from %s to requester %s", username, requesterID)
}

// handleRequestState processes request_state messages from new users
func (s *syncService) handleRequestState(ctx context.Context, roomID, userID uuid.UUID, username string, conn *websocket.Conn, rawMessage map[string]interface{}) {
	logger.Infof("processing request_state message from user %s", username)

	// this method can trigger a request to existing users if needed
	// for now, we'll let the normal join flow handle state requests
	s.requestLiveStateFromExistingUser(ctx, roomID, userID, conn)
}

// findConnection finds a connection for a specific user in a room
func (s *syncService) findConnection(roomID, userID uuid.UUID) (*websocket.Conn, bool) {
	if roomConns, exists := s.connections[roomID]; exists {
		if conn, userExists := roomConns[userID]; userExists {
			return conn, true
		}
	}
	return nil, false
}

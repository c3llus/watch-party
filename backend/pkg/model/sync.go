package model

import (
	"time"

	"github.com/google/uuid"
)

// SyncAction represents different synchronization actions
type SyncAction string

const (
	ActionPlay      SyncAction = "play"
	ActionPause     SyncAction = "pause"
	ActionSeek      SyncAction = "seek"
	ActionJoin      SyncAction = "join"
	ActionLeave     SyncAction = "leave"
	ActionBuffering SyncAction = "buffering"
	ActionReady     SyncAction = "ready"
)

// SyncMessage represents a synchronization message between clients
type SyncMessage struct {
	ID        uuid.UUID  `json:"id"`
	RoomID    uuid.UUID  `json:"room_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Username  string     `json:"username"`
	Action    SyncAction `json:"action"`
	Timestamp time.Time  `json:"timestamp"`
	Data      SyncData   `json:"data"`
}

// SyncData contains the payload data for sync actions
type SyncData struct {
	CurrentTime  float64                `json:"current_time,omitempty"`  // video current time in seconds
	Duration     float64                `json:"duration,omitempty"`      // video total duration
	PlaybackRate float64                `json:"playback_rate,omitempty"` // playback speed
	IsBuffering  bool                   `json:"is_buffering,omitempty"`
	Extra        map[string]interface{} `json:"extra,omitempty"` // additional data
}

// RoomState represents the current state of a room
type RoomState struct {
	RoomID       uuid.UUID `json:"room_id"`
	IsPlaying    bool      `json:"is_playing"`
	CurrentTime  float64   `json:"current_time"`
	Duration     float64   `json:"duration"`
	PlaybackRate float64   `json:"playback_rate"`
	LastUpdated  time.Time `json:"last_updated"`
	UpdatedBy    uuid.UUID `json:"updated_by"`
}

// ParticipantInfo represents information about a room participant
type ParticipantInfo struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	IsHost      bool      `json:"is_host"`
	JoinedAt    time.Time `json:"joined_at"`
	LastSeen    time.Time `json:"last_seen"`
	IsBuffering bool      `json:"is_buffering"`
}

// RoomSession represents an active room session with participants
type RoomSession struct {
	RoomID       uuid.UUID                     `json:"room_id"`
	State        RoomState                     `json:"state"`
	Participants map[uuid.UUID]ParticipantInfo `json:"participants"`
	CreatedAt    time.Time                     `json:"created_at"`
	UpdatedAt    time.Time                     `json:"updated_at"`
}

// WebSocketMessage represents the structure of WebSocket messages
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WebSocket message types
const (
	MessageTypeSync         = "sync"
	MessageTypeState        = "state"
	MessageTypeParticipants = "participants"
	MessageTypeError        = "error"
	MessageTypeHeartbeat    = "heartbeat"
)

// ErrorMessage represents an error message
type ErrorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HeartbeatMessage represents a heartbeat message
type HeartbeatMessage struct {
	Timestamp time.Time `json:"timestamp"`
	UserID    uuid.UUID `json:"user_id"`
}

// JoinRoomSyncRequest represents a request to join a room for sync
type JoinRoomSyncRequest struct {
	RoomID uuid.UUID `json:"room_id" binding:"required"`
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

// CreateRoomSyncRequest represents a request to create a room sync session
type CreateRoomSyncRequest struct {
	RoomID   uuid.UUID `json:"room_id" binding:"required"`
	HostID   uuid.UUID `json:"host_id" binding:"required"`
	MovieID  uuid.UUID `json:"movie_id" binding:"required"`
	Duration float64   `json:"duration"`
}

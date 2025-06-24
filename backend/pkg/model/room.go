package model

import (
	"time"

	"github.com/google/uuid"
)

// Room represents a watch party room
type Room struct {
	ID        uuid.UUID `json:"id" db:"id"`
	MovieID   uuid.UUID `json:"movie_id" db:"movie_id"`
	HostID    uuid.UUID `json:"host_id" db:"host_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RoomAccess represents user access to a room
type RoomAccess struct {
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	RoomID     uuid.UUID `json:"room_id" db:"room_id"`
	AccessType string    `json:"access_type" db:"access_type"` // "granted" or "guest"
	Status     string    `json:"status" db:"status"`           // "granted", "invited", "requested", "denied"
	GrantedAt  time.Time `json:"granted_at" db:"granted_at"`
}

// RoomAccessType constants
const (
	AccessTypeGranted = "granted"
	AccessTypeGuest   = "guest"
)

// RoomAccessStatus constants
const (
	StatusGranted   = "granted"   // User has full access
	StatusInvited   = "invited"   // User is invited and can join
	StatusRequested = "requested" // Guest requested access
	StatusDenied    = "denied"    // Access was denied
)

// CreateRoomRequest represents the request to create a new room
type CreateRoomRequest struct {
	MovieID uuid.UUID `json:"movie_id" binding:"required"`
}

// CreateRoomResponse represents the response after creating a room
type CreateRoomResponse struct {
	Room        Room   `json:"room"`
	InviteToken string `json:"invite_token,omitempty"`
	Message     string `json:"message"`
}

// RoomWithDetails represents a room with additional details
type RoomWithDetails struct {
	Room
	Movie       Movie `json:"movie"`
	Host        User  `json:"host"`
	MemberCount int   `json:"member_count"`
}

// InviteUserRequest represents the request to invite a user to a room
type InviteUserRequest struct {
	Email   string `json:"email" binding:"required,email"`
	Message string `json:"message,omitempty"`
}

// InviteUserResponse represents the response after inviting a user
type InviteUserResponse struct {
	InviteToken string    `json:"invite_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	Message     string    `json:"message"`
}

// JoinRoomRequest represents the request to join a room
type JoinRoomRequest struct {
	InviteToken string `json:"invite_token,omitempty"`
}

// JoinRoomByCodeRequest represents the request to join a room by code
type JoinRoomByCodeRequest struct {
	RoomCode string `json:"room_code" binding:"required"`
}

// JoinRoomByIDRequest represents the request to join a room by ID
type JoinRoomByIDRequest struct {
	RoomID uuid.UUID `json:"room_id" binding:"required"`
}

// JoinRoomResponse represents the response after joining a room
type JoinRoomResponse struct {
	Room    RoomWithDetails `json:"room"`
	Message string          `json:"message"`
}

// RoomInvitation represents an invitation to join a room
type RoomInvitation struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	RoomID    uuid.UUID  `json:"room_id" db:"room_id"`
	InviterID uuid.UUID  `json:"inviter_id" db:"inviter_id"`
	Email     string     `json:"email" db:"email"`
	Token     string     `json:"token" db:"token"`
	Message   string     `json:"message" db:"message"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// RoomSessionDB represents a persistent room session in the database
type RoomSessionDB struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	RoomID      uuid.UUID  `json:"room_id" db:"room_id"`
	HostID      uuid.UUID  `json:"host_id" db:"host_id"`
	MovieID     uuid.UUID  `json:"movie_id" db:"movie_id"`
	SessionName string     `json:"session_name" db:"session_name"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty" db:"ended_at"`
}

// RoomEvent represents an event that occurred during a session
type RoomEvent struct {
	ID        uuid.UUID              `json:"id" db:"id"`
	SessionID uuid.UUID              `json:"session_id" db:"session_id"`
	UserID    uuid.UUID              `json:"user_id" db:"user_id"`
	EventType string                 `json:"event_type" db:"event_type"`
	EventData map[string]interface{} `json:"event_data" db:"event_data"`
	VideoTime *float64               `json:"video_time,omitempty" db:"video_time"`
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
}

// GuestAccessRequest represents a guest request to join a room
type GuestAccessRequest struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	RoomID         uuid.UUID  `json:"room_id" db:"room_id"`
	GuestName      string     `json:"guest_name" db:"guest_name"`
	RequestMessage string     `json:"request_message" db:"request_message"`
	Status         string     `json:"status" db:"status"` // pending, approved, denied
	RequestedAt    time.Time  `json:"requested_at" db:"requested_at"`
	ReviewedBy     *uuid.UUID `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty" db:"reviewed_at"`
}

// RequestGuestAccessRequest represents the request payload for guest access
type RequestGuestAccessRequest struct {
	GuestName      string `json:"guest_name" binding:"required"`
	RequestMessage string `json:"request_message,omitempty"`
}

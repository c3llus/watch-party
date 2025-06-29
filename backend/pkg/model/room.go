package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Room represents a watch party room
type Room struct {
	ID          uuid.UUID `json:"id" db:"id"`
	MovieID     uuid.UUID `json:"movie_id" db:"movie_id"`
	HostID      uuid.UUID `json:"host_id" db:"host_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RoomAccess represents user access to a room
type RoomAccess struct {
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	RoomID     uuid.UUID `json:"room_id" db:"room_id"`
	AccessType string    `json:"access_type" db:"access_type"` // "granted" or "guest"
	Status     string    `json:"status" db:"status"`           // "granted", "requested", "denied"
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
	StatusRequested = "requested" // User requested access
	StatusDenied    = "denied"    // Access was denied
)

// GuestAccessStatus constants (for guest requests specifically)
const (
	GuestStatusPending  = "pending"  // Guest request is pending review
	GuestStatusApproved = "approved" // Guest request was approved
	GuestStatusDenied   = "denied"   // Guest request was denied
)

// CreateRoomRequest represents the request to create a new room
type CreateRoomRequest struct {
	MovieID     uuid.UUID `json:"movie_id" binding:"required"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
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

// GuestAccessRequest represents a guest's request to join a room
type GuestAccessRequest struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	RoomID         uuid.UUID  `json:"room_id" db:"room_id"`
	GuestName      string     `json:"guest_name" db:"guest_name"`
	RequestMessage string     `json:"request_message" db:"request_message"`
	Status         string     `json:"status" db:"status"`
	RequestedAt    time.Time  `json:"requested_at" db:"requested_at"`
	ReviewedBy     *uuid.UUID `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt     *time.Time `json:"reviewed_at" db:"reviewed_at"`
}

// GuestSession represents a temporary session for an approved guest
type GuestSession struct {
	ID           uuid.UUID `json:"id" db:"id"`
	RoomID       uuid.UUID `json:"room_id" db:"room_id"`
	GuestName    string    `json:"guest_name" db:"guest_name"`
	SessionToken string    `json:"session_token" db:"session_token"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	ApprovedBy   uuid.UUID `json:"approved_by" db:"approved_by"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// RoomGuestInfo represents basic room information for guests (public, no auth required)
type RoomGuestInfo struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Movie       MovieGuestInfo `json:"movie"`
}

// MovieGuestInfo represents basic movie information for guests
type MovieGuestInfo struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

// Guest request/response models
type GuestAccessRequestRequest struct {
	GuestName      string `json:"guest_name" binding:"required"`
	RequestMessage string `json:"request_message"`
}

type GuestAccessRequestResponse struct {
	RequestID uuid.UUID `json:"request_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

type ApproveGuestRequest struct {
	Approved bool   `json:"approved"`
	Message  string `json:"message"`
}

type ApproveGuestResponse struct {
	RequestID    uuid.UUID `json:"request_id"`
	Status       string    `json:"status"`
	SessionToken string    `json:"session_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Message      string    `json:"message"`
}

// UserRoomAccessRequest represents a logged-in user's request to join a room
type UserRoomAccessRequest struct {
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	RoomID         uuid.UUID  `json:"room_id" db:"room_id"`
	RequestMessage string     `json:"request_message" db:"request_message"`
	Status         string     `json:"status" db:"status"` // "requested", "approved", "denied"
	RequestedAt    time.Time  `json:"requested_at" db:"requested_at"`
	ReviewedBy     *uuid.UUID `json:"reviewed_by" db:"reviewed_by"`
	ReviewedAt     *time.Time `json:"reviewed_at" db:"reviewed_at"`
}

// UserRoomAccessRequestRequest represents the request to join a room as a logged-in user
type UserRoomAccessRequestRequest struct {
	RequestMessage string `json:"request_message"`
}

// UserRoomAccessRequestResponse represents the response after requesting room access
type UserRoomAccessRequestResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ApproveUserAccessRequest represents the request to approve/deny user room access
type ApproveUserAccessRequest struct {
	Approved bool   `json:"approved"`
	Message  string `json:"message"`
}

// ApproveUserAccessResponse represents the response after approving/denying user access
type ApproveUserAccessResponse struct {
	UserID  uuid.UUID `json:"user_id"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
}

type GuestRequest struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	RoomID       uuid.UUID      `json:"room_id" db:"room_id"`
	GuestName    string         `json:"guest_name" db:"guest_name"`
	Message      sql.NullString `json:"message" db:"message"`
	Status       string         `json:"status" db:"status"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	SessionToken sql.NullString `json:"session_token" db:"session_token"`
	ExpiresAt    sql.NullTime   `json:"expires_at" db:"expires_at"`
}

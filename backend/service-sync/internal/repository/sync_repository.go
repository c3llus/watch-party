package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"watch-party/pkg/model"
	"watch-party/pkg/redis"

	"github.com/google/uuid"
	redislib "github.com/redis/go-redis/v9"
)

// SyncRepository handles real-time sync state operations (Redis-based)
type SyncRepository interface {
	// room state operations
	SetRoomState(ctx context.Context, state *model.RoomState) error
	GetRoomState(ctx context.Context, roomID uuid.UUID) (*model.RoomState, error)
	DeleteRoomState(ctx context.Context, roomID uuid.UUID) error

	// participant operations
	AddParticipant(ctx context.Context, roomID, userID uuid.UUID, participant *model.ParticipantInfo) error
	RemoveParticipant(ctx context.Context, roomID, userID uuid.UUID) error
	GetParticipants(ctx context.Context, roomID uuid.UUID) ([]model.ParticipantInfo, error)
	UpdateParticipantPresence(ctx context.Context, roomID, userID uuid.UUID) error

	// presence operations
	SetUserPresence(ctx context.Context, userID uuid.UUID, roomID uuid.UUID, status string) error
	GetUserPresence(ctx context.Context, userID uuid.UUID) (string, error)
	RemoveUserPresence(ctx context.Context, userID uuid.UUID) error

	// room management
	GetActiveRooms(ctx context.Context, limit int64) ([]uuid.UUID, error)
	CleanupInactiveRooms(ctx context.Context, inactiveDuration time.Duration) error

	// event operations
	PublishEvent(ctx context.Context, roomID uuid.UUID, event *model.SyncMessage) error
	SubscribeToRoomEvents(ctx context.Context, roomID uuid.UUID) (*redislib.PubSub, error)

	// locking for conflict resolution
	AcquireRoomLock(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (bool, error)
	ReleaseRoomLock(ctx context.Context, roomID uuid.UUID) error
}

type syncRepository struct {
	redis *redis.Client
}

// NewSyncRepository creates a new sync repository instance
func NewSyncRepository(redisClient *redis.Client) SyncRepository {
	return &syncRepository{
		redis: redisClient,
	}
}

// Redis key helpers
func (r *syncRepository) roomSyncKey(roomID uuid.UUID) string {
	return fmt.Sprintf("watch-party:room:sync:%s", roomID.String())
}

func (r *syncRepository) roomParticipantsKey(roomID uuid.UUID) string {
	return fmt.Sprintf("watch-party:room:participants:%s", roomID.String())
}

func (r *syncRepository) userPresenceKey(userID uuid.UUID) string {
	return fmt.Sprintf("watch-party:user:presence:%s", userID.String())
}

func (r *syncRepository) roomEventsKey(roomID uuid.UUID) string {
	return fmt.Sprintf("watch-party:room:events:%s", roomID.String())
}

func (r *syncRepository) activeRoomsKey() string {
	return "watch-party:rooms:active"
}

func (r *syncRepository) roomLockKey(roomID uuid.UUID) string {
	return fmt.Sprintf("watch-party:room:lock:%s", roomID.String())
}

// SetRoomState sets the room state in Redis
func (r *syncRepository) SetRoomState(ctx context.Context, state *model.RoomState) error {
	roomKey := r.roomSyncKey(state.RoomID)
	now := time.Now().Unix()

	roomData := []interface{}{
		"room_id", state.RoomID.String(),
		"is_playing", strconv.FormatBool(state.IsPlaying),
		"current_time", fmt.Sprintf("%.2f", state.CurrentTime),
		"duration", fmt.Sprintf("%.2f", state.Duration),
		"playback_rate", fmt.Sprintf("%.2f", state.PlaybackRate),
		"last_updated", strconv.FormatInt(now, 10),
		"updated_by", state.UpdatedBy.String(),
	}

	// Set room state
	err := r.redis.HSet(ctx, roomKey, roomData...)
	if err != nil {
		return fmt.Errorf("failed to set room state: %w", err)
	}

	// Set expiration
	err = r.redis.Expire(ctx, roomKey, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	// Update active rooms index
	err = r.redis.ZAdd(ctx, r.activeRoomsKey(), redislib.Z{
		Score:  float64(now),
		Member: state.RoomID.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to update active rooms: %w", err)
	}

	return nil
}

// GetRoomState retrieves the room state from Redis
func (r *syncRepository) GetRoomState(ctx context.Context, roomID uuid.UUID) (*model.RoomState, error) {
	roomKey := r.roomSyncKey(roomID)

	data, err := r.redis.HGetAll(ctx, roomKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get room state: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("room state not found")
	}

	state := &model.RoomState{}

	// Parse room_id
	if roomIDStr, ok := data["room_id"]; ok {
		if state.RoomID, err = uuid.Parse(roomIDStr); err != nil {
			return nil, fmt.Errorf("invalid room_id: %w", err)
		}
	} else {
		state.RoomID = roomID
	}

	// Parse is_playing
	if isPlayingStr, ok := data["is_playing"]; ok {
		if state.IsPlaying, err = strconv.ParseBool(isPlayingStr); err != nil {
			return nil, fmt.Errorf("invalid is_playing: %w", err)
		}
	}

	// Parse current_time
	if currentTimeStr, ok := data["current_time"]; ok {
		if state.CurrentTime, err = strconv.ParseFloat(currentTimeStr, 64); err != nil {
			return nil, fmt.Errorf("invalid current_time: %w", err)
		}
	}

	// Parse duration
	if durationStr, ok := data["duration"]; ok {
		if state.Duration, err = strconv.ParseFloat(durationStr, 64); err != nil {
			return nil, fmt.Errorf("invalid duration: %w", err)
		}
	}

	// Parse playback_rate
	if playbackRateStr, ok := data["playback_rate"]; ok {
		if state.PlaybackRate, err = strconv.ParseFloat(playbackRateStr, 64); err != nil {
			return nil, fmt.Errorf("invalid playback_rate: %w", err)
		}
	}

	// Parse last_updated
	if lastUpdatedStr, ok := data["last_updated"]; ok {
		if timestamp, err := strconv.ParseInt(lastUpdatedStr, 10, 64); err == nil {
			state.LastUpdated = time.Unix(timestamp, 0)
		}
	}

	// Parse updated_by
	if updatedByStr, ok := data["updated_by"]; ok {
		if state.UpdatedBy, err = uuid.Parse(updatedByStr); err != nil {
			return nil, fmt.Errorf("invalid updated_by: %w", err)
		}
	}

	return state, nil
}

// DeleteRoomState removes the room state from Redis
func (r *syncRepository) DeleteRoomState(ctx context.Context, roomID uuid.UUID) error {
	roomKey := r.roomSyncKey(roomID)
	participantsKey := r.roomParticipantsKey(roomID)
	eventsKey := r.roomEventsKey(roomID)

	err := r.redis.Delete(ctx, roomKey, participantsKey, eventsKey)
	if err != nil {
		return fmt.Errorf("failed to delete room state: %w", err)
	}

	// Remove from active rooms
	err = r.redis.ZRem(ctx, r.activeRoomsKey(), roomID.String())
	if err != nil {
		return fmt.Errorf("failed to remove from active rooms: %w", err)
	}

	return nil
}

// AddParticipant adds a participant to a room
func (r *syncRepository) AddParticipant(ctx context.Context, roomID, userID uuid.UUID, participant *model.ParticipantInfo) error {
	participantsKey := r.roomParticipantsKey(roomID)

	participantData, err := json.Marshal(participant)
	if err != nil {
		return fmt.Errorf("failed to marshal participant data: %w", err)
	}

	err = r.redis.HSet(ctx, participantsKey, userID.String(), string(participantData))
	if err != nil {
		return fmt.Errorf("failed to add participant: %w", err)
	}

	// Set expiration
	err = r.redis.Expire(ctx, participantsKey, 24*time.Hour)
	if err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	return nil
}

// RemoveParticipant removes a participant from a room
func (r *syncRepository) RemoveParticipant(ctx context.Context, roomID, userID uuid.UUID) error {
	participantsKey := r.roomParticipantsKey(roomID)

	err := r.redis.HDel(ctx, participantsKey, userID.String())
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	return nil
}

// GetParticipants retrieves all participants in a room
func (r *syncRepository) GetParticipants(ctx context.Context, roomID uuid.UUID) ([]model.ParticipantInfo, error) {
	participantsKey := r.roomParticipantsKey(roomID)

	data, err := r.redis.HGetAll(ctx, participantsKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	participants := make([]model.ParticipantInfo, 0, len(data))

	for _, participantData := range data {
		var participant model.ParticipantInfo
		if err := json.Unmarshal([]byte(participantData), &participant); err != nil {
			continue // skip invalid entries
		}
		participants = append(participants, participant)
	}

	return participants, nil
}

// UpdateParticipantPresence updates the last seen time for a participant
func (r *syncRepository) UpdateParticipantPresence(ctx context.Context, roomID, userID uuid.UUID) error {
	participantsKey := r.roomParticipantsKey(roomID)

	// Get current participant data
	participantData, err := r.redis.HGet(ctx, participantsKey, userID.String())
	if err != nil {
		return fmt.Errorf("participant not found: %w", err)
	}

	var participant model.ParticipantInfo
	if err := json.Unmarshal([]byte(participantData), &participant); err != nil {
		return fmt.Errorf("failed to unmarshal participant data: %w", err)
	}

	// Update last seen
	participant.LastSeen = time.Now()

	// Marshal and store back
	updatedData, err := json.Marshal(participant)
	if err != nil {
		return fmt.Errorf("failed to marshal updated participant data: %w", err)
	}

	err = r.redis.HSet(ctx, participantsKey, userID.String(), string(updatedData))
	if err != nil {
		return fmt.Errorf("failed to update participant presence: %w", err)
	}

	return nil
}

// SetUserPresence sets user presence information
func (r *syncRepository) SetUserPresence(ctx context.Context, userID uuid.UUID, roomID uuid.UUID, status string) error {
	presenceKey := r.userPresenceKey(userID)

	presenceData := map[string]interface{}{
		"room_id":   roomID.String(),
		"status":    status,
		"timestamp": time.Now().Unix(),
	}

	err := r.redis.Set(ctx, presenceKey, presenceData, 60*time.Second)
	if err != nil {
		return fmt.Errorf("failed to set user presence: %w", err)
	}

	return nil
}

// GetUserPresence retrieves user presence information
func (r *syncRepository) GetUserPresence(ctx context.Context, userID uuid.UUID) (string, error) {
	presenceKey := r.userPresenceKey(userID)

	var presenceData map[string]interface{}
	err := r.redis.Get(ctx, presenceKey, &presenceData)
	if err != nil {
		return "", fmt.Errorf("failed to get user presence: %w", err)
	}

	data, err := json.Marshal(presenceData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal presence data: %w", err)
	}

	return string(data), nil
}

// RemoveUserPresence removes user presence information
func (r *syncRepository) RemoveUserPresence(ctx context.Context, userID uuid.UUID) error {
	presenceKey := r.userPresenceKey(userID)

	err := r.redis.Delete(ctx, presenceKey)
	if err != nil {
		return fmt.Errorf("failed to remove user presence: %w", err)
	}

	return nil
}

// GetActiveRooms retrieves currently active rooms
func (r *syncRepository) GetActiveRooms(ctx context.Context, limit int64) ([]uuid.UUID, error) {
	roomIDStrs, err := r.redis.ZRevRange(ctx, r.activeRoomsKey(), 0, limit-1)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rooms: %w", err)
	}

	roomIDs := make([]uuid.UUID, 0, len(roomIDStrs))
	for _, roomIDStr := range roomIDStrs {
		if roomID, err := uuid.Parse(roomIDStr); err == nil {
			roomIDs = append(roomIDs, roomID)
		}
	}

	return roomIDs, nil
}

// CleanupInactiveRooms removes rooms that have been inactive for the specified duration
func (r *syncRepository) CleanupInactiveRooms(ctx context.Context, inactiveDuration time.Duration) error {
	cutoffTime := time.Now().Add(-inactiveDuration).Unix()

	// Get inactive rooms
	roomIDStrs, err := r.redis.ZRangeByScore(ctx, r.activeRoomsKey(), &redislib.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%d", cutoffTime),
	})
	if err != nil {
		return fmt.Errorf("failed to get inactive rooms: %w", err)
	}

	// Delete inactive rooms
	for _, roomIDStr := range roomIDStrs {
		if roomID, err := uuid.Parse(roomIDStr); err == nil {
			r.DeleteRoomState(ctx, roomID)
		}
	}

	return nil
}

// PublishEvent publishes a sync event to the room's event stream
func (r *syncRepository) PublishEvent(ctx context.Context, roomID uuid.UUID, event *model.SyncMessage) error {
	err := r.redis.Publish(ctx, fmt.Sprintf("room:%s:events", roomID.String()), event)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// SubscribeToRoomEvents subscribes to room events
func (r *syncRepository) SubscribeToRoomEvents(ctx context.Context, roomID uuid.UUID) (*redislib.PubSub, error) {
	channel := fmt.Sprintf("room:%s:events", roomID.String())
	pubsub := r.redis.Subscribe(ctx, channel)
	return pubsub, nil
}

// AcquireRoomLock acquires a lock for a room to prevent conflicts
func (r *syncRepository) AcquireRoomLock(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (bool, error) {
	lockKey := r.roomLockKey(roomID)

	acquired, err := r.redis.SetNX(ctx, lockKey, userID.String(), 5*time.Second)
	if err != nil {
		return false, fmt.Errorf("failed to acquire room lock: %w", err)
	}

	return acquired, nil
}

// ReleaseRoomLock releases a room lock
func (r *syncRepository) ReleaseRoomLock(ctx context.Context, roomID uuid.UUID) error {
	lockKey := r.roomLockKey(roomID)

	err := r.redis.Delete(ctx, lockKey)
	if err != nil {
		return fmt.Errorf("failed to release room lock: %w", err)
	}

	return nil
}

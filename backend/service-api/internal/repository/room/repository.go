package room

import (
	"context"
	"database/sql"
	"time"
	"watch-party/pkg/model"

	"github.com/google/uuid"
)

// Repository handles room data operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new room repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateRoom creates a new room
func (r *Repository) CreateRoom(ctx context.Context, room *model.Room) error {
	query := `
		INSERT INTO rooms (id, movie_id, host_id, created_at)
		VALUES ($1, $2, $3, $4)`

	_, err := r.db.ExecContext(ctx, query, room.ID, room.MovieID, room.HostID, room.CreatedAt)
	return err
}

// GetRoomByID retrieves a room by ID
func (r *Repository) GetRoomByID(ctx context.Context, roomID uuid.UUID) (*model.Room, error) {
	var room model.Room
	query := `SELECT id, movie_id, host_id, created_at FROM rooms WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, roomID)
	err := row.Scan(&room.ID, &room.MovieID, &room.HostID, &room.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &room, nil
}

// GetRoomWithDetails retrieves a room with movie and host details
func (r *Repository) GetRoomWithDetails(ctx context.Context, roomID uuid.UUID) (*model.RoomWithDetails, error) {
	var roomDetails model.RoomWithDetails
	query := `
		SELECT 
			r.id, r.movie_id, r.host_id, r.created_at,
			m.id, m.title, m.description, m.storage_provider, m.storage_path,
			m.duration_seconds, m.file_size, m.mime_type, m.uploaded_by, m.created_at,
			u.id, u.email, u.role, u.created_at
		FROM rooms r
		JOIN movies m ON r.movie_id = m.id
		JOIN users u ON r.host_id = u.id
		WHERE r.id = $1`

	row := r.db.QueryRowContext(ctx, query, roomID)
	err := row.Scan(
		&roomDetails.ID, &roomDetails.MovieID, &roomDetails.HostID, &roomDetails.CreatedAt,
		&roomDetails.Movie.ID, &roomDetails.Movie.Title, &roomDetails.Movie.Description,
		&roomDetails.Movie.StorageProvider, &roomDetails.Movie.StoragePath,
		&roomDetails.Movie.DurationSeconds, &roomDetails.Movie.FileSize,
		&roomDetails.Movie.MimeType, &roomDetails.Movie.UploadedBy, &roomDetails.Movie.CreatedAt,
		&roomDetails.Host.ID, &roomDetails.Host.Email, &roomDetails.Host.Role, &roomDetails.Host.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Get member count
	memberCount, err := r.GetRoomMemberCount(ctx, roomID)
	if err != nil {
		return nil, err
	}
	roomDetails.MemberCount = memberCount

	return &roomDetails, nil
}

// GetRoomMemberCount returns the number of members in a room
func (r *Repository) GetRoomMemberCount(ctx context.Context, roomID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM room_access WHERE room_id = $1`

	row := r.db.QueryRowContext(ctx, query, roomID)
	err := row.Scan(&count)
	return count, err
}

// GrantRoomAccess grants access to a room for a user
func (r *Repository) GrantRoomAccess(ctx context.Context, access *model.RoomAccess) error {
	query := `
		INSERT INTO room_access (user_id, room_id, access_type, granted_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, room_id) DO UPDATE SET
			access_type = $3,
			granted_at = $4`

	_, err := r.db.ExecContext(ctx, query, access.UserID, access.RoomID, access.AccessType, access.GrantedAt)
	return err
}

// CheckRoomAccess checks if a user has access to a room
func (r *Repository) CheckRoomAccess(ctx context.Context, userID, roomID uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM room_access WHERE user_id = $1 AND room_id = $2`

	row := r.db.QueryRowContext(ctx, query, userID, roomID)
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateInvitation creates a new room invitation
func (r *Repository) CreateInvitation(ctx context.Context, invitation *model.RoomInvitation) error {
	query := `
		INSERT INTO room_invitations (id, room_id, inviter_id, email, token, message, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		invitation.ID, invitation.RoomID, invitation.InviterID, invitation.Email,
		invitation.Token, invitation.Message, invitation.ExpiresAt, invitation.CreatedAt)
	return err
}

// GetInvitationByToken retrieves an invitation by token
func (r *Repository) GetInvitationByToken(ctx context.Context, token string) (*model.RoomInvitation, error) {
	var invitation model.RoomInvitation
	query := `
		SELECT id, room_id, inviter_id, email, token, message, expires_at, used_at, created_at
		FROM room_invitations 
		WHERE token = $1`

	row := r.db.QueryRowContext(ctx, query, token)
	err := row.Scan(&invitation.ID, &invitation.RoomID, &invitation.InviterID,
		&invitation.Email, &invitation.Token, &invitation.Message,
		&invitation.ExpiresAt, &invitation.UsedAt, &invitation.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &invitation, nil
}

// MarkInvitationUsed marks an invitation as used
func (r *Repository) MarkInvitationUsed(ctx context.Context, token string) error {
	query := `UPDATE room_invitations SET used_at = $1 WHERE token = $2`
	_, err := r.db.ExecContext(ctx, query, time.Now(), token)
	return err
}

// IsRoomHost checks if a user is the host of a room
func (r *Repository) IsRoomHost(ctx context.Context, userID, roomID uuid.UUID) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM rooms WHERE id = $1 AND host_id = $2`

	row := r.db.QueryRowContext(ctx, query, roomID, userID)
	err := row.Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

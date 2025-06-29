package room

import (
	"context"
	"database/sql"
	"time"
	"watch-party/pkg/logger"
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
		INSERT INTO rooms (id, movie_id, host_id, name, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query, room.ID, room.MovieID, room.HostID, room.Name, room.Description, room.CreatedAt)
	return err
}

// GetRoomByID retrieves a room by ID
func (r *Repository) GetRoomByID(ctx context.Context, roomID uuid.UUID) (*model.Room, error) {
	var room model.Room
	query := `SELECT id, movie_id, host_id, name, description, created_at FROM rooms WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, roomID)
	err := row.Scan(&room.ID, &room.MovieID, &room.HostID, &room.Name, &room.Description, &room.CreatedAt)
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
			r.id, r.movie_id, r.host_id, r.name, r.description, r.created_at,
			m.id, m.title, m.description, m.original_file_path, m.transcoded_file_path,
			m.hls_playlist_url, m.duration_seconds, m.file_size, m.mime_type, m.status,
			m.uploaded_by, m.created_at, m.processing_started_at, m.processing_ended_at,
			u.id, u.email, u.role, u.created_at
		FROM rooms r
		JOIN movies m ON r.movie_id = m.id
		JOIN users u ON r.host_id = u.id
		WHERE r.id = $1`

	row := r.db.QueryRowContext(ctx, query, roomID)
	err := row.Scan(
		&roomDetails.ID, &roomDetails.MovieID, &roomDetails.HostID, &roomDetails.Name, &roomDetails.Description, &roomDetails.CreatedAt,
		&roomDetails.Movie.ID, &roomDetails.Movie.Title, &roomDetails.Movie.Description,
		&roomDetails.Movie.OriginalFilePath, &roomDetails.Movie.TranscodedFilePath,
		&roomDetails.Movie.HLSPlaylistURL, &roomDetails.Movie.DurationSeconds, &roomDetails.Movie.FileSize,
		&roomDetails.Movie.MimeType, &roomDetails.Movie.Status, &roomDetails.Movie.UploadedBy, &roomDetails.Movie.CreatedAt,
		&roomDetails.Movie.ProcessingStartedAt, &roomDetails.Movie.ProcessingEndedAt,
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
		INSERT INTO room_access (user_id, room_id, access_type, status, granted_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, room_id) DO UPDATE SET
			access_type = $3,
			status = $4,
			granted_at = $5`

	_, err := r.db.ExecContext(ctx, query, access.UserID, access.RoomID, access.AccessType, access.Status, access.GrantedAt)
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

// GetUserRoomAccess retrieves the access record for a user in a room
func (r *Repository) GetUserRoomAccess(ctx context.Context, userID, roomID uuid.UUID) (*model.RoomAccess, error) {
	var access model.RoomAccess
	query := `SELECT user_id, room_id, access_type, status, granted_at FROM room_access WHERE user_id = $1 AND room_id = $2`

	row := r.db.QueryRowContext(ctx, query, userID, roomID)
	err := row.Scan(&access.UserID, &access.RoomID, &access.AccessType, &access.Status, &access.GrantedAt)
	if err != nil {
		return nil, err
	}

	return &access, nil
}

// UpdateRoomAccess updates the access record for a user in a room
func (r *Repository) UpdateRoomAccess(ctx context.Context, access *model.RoomAccess) error {
	query := `
		UPDATE room_access 
		SET access_type = $3, status = $4, granted_at = $5
		WHERE user_id = $1 AND room_id = $2`

	_, err := r.db.ExecContext(ctx, query, access.UserID, access.RoomID, access.AccessType, access.Status, access.GrantedAt)
	return err
}

// Guest access methods

// CreateGuestAccessRequest creates a new guest access request
func (r *Repository) CreateGuestAccessRequest(ctx context.Context, req *model.GuestAccessRequest) error {
	query := `
		INSERT INTO guest_access_requests (id, room_id, guest_name, request_message, status, requested_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query, req.ID, req.RoomID, req.GuestName, req.RequestMessage, req.Status, req.RequestedAt)
	return err
}

// GetGuestAccessRequest retrieves a guest access request by ID
func (r *Repository) GetGuestAccessRequest(ctx context.Context, requestID uuid.UUID) (*model.GuestAccessRequest, error) {
	var req model.GuestAccessRequest
	query := `
		SELECT id, room_id, guest_name, request_message, status, requested_at, reviewed_by, reviewed_at
		FROM guest_access_requests WHERE id = $1`

	row := r.db.QueryRowContext(ctx, query, requestID)
	err := row.Scan(&req.ID, &req.RoomID, &req.GuestName, &req.RequestMessage, &req.Status, &req.RequestedAt, &req.ReviewedBy, &req.ReviewedAt)
	if err != nil {
		return nil, err
	}

	return &req, nil
}

// GetPendingGuestRequests retrieves all pending guest requests for a room
func (r *Repository) GetPendingGuestRequests(ctx context.Context, roomID uuid.UUID) ([]model.GuestAccessRequest, error) {
	var requests []model.GuestAccessRequest
	query := `
		SELECT id, room_id, guest_name, request_message, status, requested_at, reviewed_by, reviewed_at
		FROM guest_access_requests 
		WHERE room_id = $1 AND status = 'pending'
		ORDER BY requested_at ASC`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var req model.GuestAccessRequest
		err := rows.Scan(&req.ID, &req.RoomID, &req.GuestName, &req.RequestMessage, &req.Status, &req.RequestedAt, &req.ReviewedBy, &req.ReviewedAt)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	return requests, rows.Err()
}

// UpdateGuestAccessRequest updates the status of a guest access request
func (r *Repository) UpdateGuestAccessRequest(ctx context.Context, requestID uuid.UUID, status string, reviewedBy uuid.UUID) error {
	query := `
		UPDATE guest_access_requests 
		SET status = $1, reviewed_by = $2, reviewed_at = NOW()
		WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, status, reviewedBy, requestID)
	return err
}

// CreateGuestSession creates a temporary session for an approved guest
func (r *Repository) CreateGuestSession(ctx context.Context, session *model.GuestSession) error {
	query := `
		INSERT INTO guest_sessions (id, room_id, guest_name, session_token, expires_at, approved_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query, session.ID, session.RoomID, session.GuestName, session.SessionToken, session.ExpiresAt, session.ApprovedBy, session.CreatedAt)
	return err
}

// GetGuestSessionByToken retrieves a guest session by token
func (r *Repository) GetGuestSessionByToken(ctx context.Context, token string) (*model.GuestSession, error) {
	var session model.GuestSession
	query := `
		SELECT id, room_id, guest_name, session_token, expires_at, approved_by, created_at
		FROM guest_sessions 
		WHERE session_token = $1 AND expires_at > NOW()`

	row := r.db.QueryRowContext(ctx, query, token)
	err := row.Scan(&session.ID, &session.RoomID, &session.GuestName, &session.SessionToken, &session.ExpiresAt, &session.ApprovedBy, &session.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// CleanupExpiredGuestSessions removes expired guest sessions
func (r *Repository) CleanupExpiredGuestSessions(ctx context.Context) error {
	query := `DELETE FROM guest_sessions WHERE expires_at <= NOW()`
	_, err := r.db.ExecContext(ctx, query)
	return err
}

// CheckUserMovieAccess checks if a user has access to stream a specific movie
// by verifying they are a member of a room containing that movie
func (r *Repository) CheckUserMovieAccess(ctx context.Context, userID uuid.UUID, movieID uuid.UUID) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM room_access ra
		JOIN rooms r ON ra.room_id = r.id
		WHERE ra.user_id = $1 
		  AND r.movie_id = $2 
		  AND ra.status = 'granted'`

	logger.Infof("Checking movie access for user %s to movie %s", userID, movieID)
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, movieID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// CheckRoomContainsMovie verifies if a specific room contains the given movie
func (r *Repository) CheckRoomContainsMovie(ctx context.Context, roomID uuid.UUID, movieID uuid.UUID) (bool, error) {
	query := `SELECT COUNT(*) FROM rooms WHERE id = $1 AND movie_id = $2`

	var count int
	err := r.db.QueryRowContext(ctx, query, roomID, movieID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserRooms retrieves all rooms where the user is host or has access
func (r *Repository) GetUserRooms(ctx context.Context, userID uuid.UUID) ([]*model.RoomWithDetails, error) {
	var rooms []*model.RoomWithDetails
	query := `
		SELECT DISTINCT
			r.id, r.movie_id, r.host_id, r.name, r.description, r.created_at,
			m.id, m.title, m.description, m.original_file_path, m.transcoded_file_path,
			m.hls_playlist_url, m.duration_seconds, m.file_size, m.mime_type, m.status,
			m.uploaded_by, m.created_at, m.processing_started_at, m.processing_ended_at,
			u.id, u.email, u.role, u.created_at
		FROM rooms r
		JOIN movies m ON r.movie_id = m.id
		JOIN users u ON r.host_id = u.id
		LEFT JOIN room_access ra ON r.id = ra.room_id
		WHERE r.host_id = $1 OR (ra.user_id = $1 AND ra.status = 'granted')
		ORDER BY r.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var roomDetails model.RoomWithDetails
		err := rows.Scan(
			&roomDetails.ID, &roomDetails.MovieID, &roomDetails.HostID, &roomDetails.Name, &roomDetails.Description, &roomDetails.CreatedAt,
			&roomDetails.Movie.ID, &roomDetails.Movie.Title, &roomDetails.Movie.Description,
			&roomDetails.Movie.OriginalFilePath, &roomDetails.Movie.TranscodedFilePath,
			&roomDetails.Movie.HLSPlaylistURL, &roomDetails.Movie.DurationSeconds, &roomDetails.Movie.FileSize,
			&roomDetails.Movie.MimeType, &roomDetails.Movie.Status, &roomDetails.Movie.UploadedBy, &roomDetails.Movie.CreatedAt,
			&roomDetails.Movie.ProcessingStartedAt, &roomDetails.Movie.ProcessingEndedAt,
			&roomDetails.Host.ID, &roomDetails.Host.Email, &roomDetails.Host.Role, &roomDetails.Host.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Get member count
		memberCount, err := r.GetRoomMemberCount(ctx, roomDetails.ID)
		if err != nil {
			return nil, err
		}
		roomDetails.MemberCount = memberCount

		rooms = append(rooms, &roomDetails)
	}

	return rooms, nil
}

// GetGuestRequestByID retrieves a guest request by ID
func (r *Repository) GetGuestRequestByID(ctx context.Context, requestID uuid.UUID) (*model.GuestRequest, error) {
	var request model.GuestRequest
	query := `
		SELECT 
			gar.id, 
			gar.room_id, 
			gar.guest_name, 
			gar.request_message, 
			gar.status, 
			gar.requested_at,
			gs.session_token,
			gs.expires_at
		FROM guest_access_requests gar
		LEFT JOIN guest_sessions gs ON gar.room_id = gs.room_id AND gar.guest_name = gs.guest_name
		WHERE gar.id = $1`

	row := r.db.QueryRowContext(ctx, query, requestID)
	err := row.Scan(
		&request.ID,
		&request.RoomID,
		&request.GuestName,
		&request.Message,
		&request.Status,
		&request.CreatedAt,
		&request.SessionToken,
		&request.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	return &request, nil
}

// GetPendingRoomAccessRequests retrieves all pending room access requests for a room (for users, not guests)
func (r *Repository) GetPendingRoomAccessRequests(ctx context.Context, roomID uuid.UUID) ([]model.UserRoomAccessRequest, error) {
	var requests []model.UserRoomAccessRequest
	query := `
		SELECT 
			ra.user_id, 
			ra.room_id, 
			'' as request_message,
			ra.status, 
			ra.granted_at as requested_at,
			NULL as reviewed_by,
			NULL as reviewed_at
		FROM room_access ra
		WHERE ra.room_id = $1 AND ra.status = 'requested'
		ORDER BY ra.granted_at ASC`

	rows, err := r.db.QueryContext(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var req model.UserRoomAccessRequest
		err := rows.Scan(&req.UserID, &req.RoomID, &req.RequestMessage, &req.Status, &req.RequestedAt, &req.ReviewedBy, &req.ReviewedAt)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}

	return requests, rows.Err()
}

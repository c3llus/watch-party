package room

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
	"watch-party/pkg/config"
	"watch-party/pkg/email"
	"watch-party/pkg/model"
	roomRepo "watch-party/service-api/internal/repository/room"
	userRepo "watch-party/service-api/internal/repository/user"

	"github.com/google/uuid"
)

// Service provides room-related services.
type Service struct {
	roomRepo     *roomRepo.Repository
	userRepo     userRepo.Repository
	emailService email.Provider
	config       *config.Config
}

// NewService creates a new room service instance.
func NewService(roomRepo *roomRepo.Repository, userRepo userRepo.Repository, emailService email.Provider, config *config.Config) *Service {
	return &Service{
		roomRepo:     roomRepo,
		userRepo:     userRepo,
		emailService: emailService,
		config:       config,
	}
}

// CreateRoom creates a new room
func (s *Service) CreateRoom(ctx context.Context, userID uuid.UUID, req *model.CreateRoomRequest) (*model.CreateRoomResponse, error) {
	// create room
	room := &model.Room{
		ID:        uuid.New(),
		MovieID:   req.MovieID,
		HostID:    userID,
		CreatedAt: time.Now(),
	}

	err := s.roomRepo.CreateRoom(ctx, room)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	// grant access to the host
	access := &model.RoomAccess{
		UserID:     userID,
		RoomID:     room.ID,
		AccessType: model.AccessTypeGranted,
		GrantedAt:  time.Now(),
	}

	err = s.roomRepo.GrantRoomAccess(ctx, access)
	if err != nil {
		return nil, fmt.Errorf("failed to grant host access: %w", err)
	}

	return &model.CreateRoomResponse{
		Room:    *room,
		Message: "Room created successfully",
	}, nil
}

// GetRoom retrieves a room by ID
func (s *Service) GetRoom(ctx context.Context, userID, roomID uuid.UUID) (*model.RoomWithDetails, error) {
	// check if user has access to the room
	hasAccess, err := s.roomRepo.CheckRoomAccess(ctx, userID, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to check room access: %w", err)
	}

	if !hasAccess {
		return nil, fmt.Errorf("access denied")
	}

	// get room details
	room, err := s.roomRepo.GetRoomWithDetails(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	return room, nil
}

// InviteUser sends an email invitation to join a room
func (s *Service) InviteUser(ctx context.Context, inviterID, roomID uuid.UUID, req *model.InviteUserRequest) (*model.InviteUserResponse, error) {
	// check if the inviter is the host of the room
	isHost, err := s.roomRepo.IsRoomHost(ctx, inviterID, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to check room host: %w", err)
	}

	if !isHost {
		return nil, fmt.Errorf("only room host can send invitations")
	}

	// get room details for the email
	room, err := s.roomRepo.GetRoomWithDetails(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room details: %w", err)
	}

	// get inviter details
	inviter, err := s.userRepo.GetByID(inviterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inviter details: %w", err)
	}

	// generate invitation token
	token, err := s.generateInvitationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation token: %w", err)
	}

	// create invitation record
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	invitation := &model.RoomInvitation{
		ID:        uuid.New(),
		RoomID:    roomID,
		InviterID: inviterID,
		Email:     req.Email,
		Token:     token,
		Message:   req.Message,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	err = s.roomRepo.CreateInvitation(ctx, invitation)
	if err != nil {
		return nil, fmt.Errorf("failed to create invitation: %w", err)
	}

	// send email invitation
	err = s.sendInvitationEmail(ctx, invitation, inviter, room)
	if err != nil {
		// log the error but don't fail the request
		// the invitation is created, just the email sending failed
		fmt.Printf("Warning: Failed to send invitation email: %v\n", err)
	}

	return &model.InviteUserResponse{
		InviteToken: token,
		ExpiresAt:   expiresAt,
		Message:     "Invitation sent successfully",
	}, nil
}

// JoinRoomByInvitation allows a user to join a room using an invitation token
func (s *Service) JoinRoomByInvitation(ctx context.Context, userID uuid.UUID, req *model.JoinRoomRequest) (*model.JoinRoomResponse, error) {
	// get invitation by token
	invitation, err := s.roomRepo.GetInvitationByToken(ctx, req.InviteToken)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid invitation token")
		}
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}

	// check if invitation is still valid
	if time.Now().After(invitation.ExpiresAt) {
		return nil, fmt.Errorf("invitation has expired")
	}

	if invitation.UsedAt != nil {
		return nil, fmt.Errorf("invitation has already been used")
	}

	// grant room access to the user
	access := &model.RoomAccess{
		UserID:     userID,
		RoomID:     invitation.RoomID,
		AccessType: model.AccessTypeGranted,
		GrantedAt:  time.Now(),
	}

	err = s.roomRepo.GrantRoomAccess(ctx, access)
	if err != nil {
		return nil, fmt.Errorf("failed to grant room access: %w", err)
	}

	// mark invitation as used
	err = s.roomRepo.MarkInvitationUsed(ctx, req.InviteToken)
	if err != nil {
		return nil, fmt.Errorf("failed to mark invitation as used: %w", err)
	}

	// get room details
	room, err := s.roomRepo.GetRoomWithDetails(ctx, invitation.RoomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room details: %w", err)
	}

	return &model.JoinRoomResponse{
		Room:    *room,
		Message: "Successfully joined the room",
	}, nil
}

// generateInvitationToken generates a secure random token for invitations
func (s *Service) generateInvitationToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// sendInvitationEmail sends an invitation email
func (s *Service) sendInvitationEmail(ctx context.Context, invitation *model.RoomInvitation, inviter *model.User, room *model.RoomWithDetails) error {
	// construct invitation URL
	inviteURL := fmt.Sprintf("%s/rooms/join?token=%s", s.config.Email.Templates.BaseURL, invitation.Token)

	// prepare template data
	templateData := email.InvitationTemplateData{
		TemplateData: email.TemplateData{
			RecipientName: invitation.Email,
			SenderName:    inviter.Email,
			AppName:       s.config.Email.Templates.AppName,
			AppURL:        s.config.Email.Templates.BaseURL,
		},
		RoomID:      invitation.RoomID.String(),
		MovieTitle:  room.Movie.Title,
		InviterName: inviter.Email,
		InviteURL:   inviteURL,
		ExpiresAt:   invitation.ExpiresAt.Format("January 2, 2006 at 3:04 PM"),
	}

	// send email
	return s.emailService.SendTemplateEmail(ctx, []string{invitation.Email}, email.TemplateRoomInvitation, templateData)
}

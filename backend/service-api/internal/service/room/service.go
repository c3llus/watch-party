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
		Status:     model.StatusGranted,
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

// InviteUser sends an email invitation and adds user to room access list
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

	// check if user exists by email
	invitedUser, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		// user doesn't exist yet - we'll send them the room link anyway
		// they can register and join later
		fmt.Printf("Note: Invited user %s not found in system, sending room link anyway\n", req.Email)
	}

	// if user exists, add them to room access list immediately
	if invitedUser != nil {
		access := &model.RoomAccess{
			UserID:     invitedUser.ID,
			RoomID:     roomID,
			AccessType: model.AccessTypeGranted,
			Status:     model.StatusInvited,
			GrantedAt:  time.Now(),
		}

		err = s.roomRepo.GrantRoomAccess(ctx, access)
		if err != nil {
			return nil, fmt.Errorf("failed to grant room access: %w", err)
		}
	}

	// send email invitation with persistent room link
	err = s.sendInvitationEmailWithRoomLink(ctx, req, inviter, room)
	if err != nil {
		// log the error but don't fail the request
		fmt.Printf("Warning: Failed to send invitation email: %v\n", err)
	}

	return &model.InviteUserResponse{
		InviteToken: "",          // No longer using tokens
		ExpiresAt:   time.Time{}, // No expiration
		Message:     "Invitation sent successfully. User can join anytime using the room link.",
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

	// Note: Removed single-use restriction to allow multiple joins like Google Meet
	// if invitation.UsedAt != nil {
	//     return nil, fmt.Errorf("invitation has already been used")
	// }

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

	// Note: Removed invitation marking as used to allow multiple joins
	// err = s.roomRepo.MarkInvitationUsed(ctx, req.InviteToken)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to mark invitation as used: %w", err)
	// }

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

// JoinRoomByID allows a user to join a room using room ID (new Google Meet-style method)
func (s *Service) JoinRoomByID(ctx context.Context, userID uuid.UUID, roomID uuid.UUID) (*model.JoinRoomResponse, error) {
	// check if user already has access to the room
	hasAccess, err := s.roomRepo.CheckRoomAccess(ctx, userID, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to check room access: %w", err)
	}

	if !hasAccess {
		// check if user is invited
		accessRecord, err := s.roomRepo.GetUserRoomAccess(ctx, userID, roomID)
		if err != nil || accessRecord == nil || accessRecord.Status != model.StatusInvited {
			return nil, fmt.Errorf("access denied - you need to be invited to this room")
		}

		// user is invited, upgrade their status to granted
		accessRecord.Status = model.StatusGranted
		accessRecord.GrantedAt = time.Now()

		err = s.roomRepo.UpdateRoomAccess(ctx, accessRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to update room access: %w", err)
		}
	}

	// get room details
	room, err := s.roomRepo.GetRoomWithDetails(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
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

// sendInvitationEmailWithRoomLink sends an invitation email with persistent room link
func (s *Service) sendInvitationEmailWithRoomLink(ctx context.Context, req *model.InviteUserRequest, inviter *model.User, room *model.RoomWithDetails) error {
	// construct room URL (persistent link)
	roomURL := fmt.Sprintf("%s/rooms/join/%s", s.config.Email.Templates.BaseURL, room.ID.String())

	// prepare template data for new persistent link format
	templateData := email.InvitationTemplateData{
		TemplateData: email.TemplateData{
			RecipientName: req.Email,
			SenderName:    inviter.Email,
			AppName:       s.config.Email.Templates.AppName,
			AppURL:        s.config.Email.Templates.BaseURL,
		},
		RoomID:      room.ID.String(),
		MovieTitle:  room.Movie.Title,
		InviterName: inviter.Email,
		InviteURL:   roomURL,
		ExpiresAt:   "Never (you can join anytime!)",
	}

	// send email
	return s.emailService.SendTemplateEmail(ctx, []string{req.Email}, email.TemplateRoomInvitation, templateData)
}

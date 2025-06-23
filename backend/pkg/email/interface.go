package email

import (
	"context"
)

// Provider defines the interface for email providers
type Provider interface {
	// SendEmail sends an email with the specified content
	SendEmail(ctx context.Context, to []string, subject string, body EmailBody) error

	// SendTemplateEmail sends an email using a template
	SendTemplateEmail(ctx context.Context, to []string, templateName string, data interface{}) error

	// ValidateProvider validates the provider configuration
	ValidateProvider(ctx context.Context) error
}

// EmailBody represents the email content
type EmailBody struct {
	HTML string // HTML content
	Text string // Plain text content
}

// TemplateData represents common template data for emails
type TemplateData struct {
	RecipientName string
	SenderName    string
	AppName       string
	AppURL        string
}

// InvitationTemplateData represents data for room invitation emails
type InvitationTemplateData struct {
	TemplateData
	RoomID      string
	MovieTitle  string
	InviterName string
	InviteURL   string
	ExpiresAt   string
}

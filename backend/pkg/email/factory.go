package email

import (
	"context"
	"fmt"
	"watch-party/pkg/config"
)

// email provider constants
const (
	ProviderSMTP     = "smtp"
	ProviderSendGrid = "sendgrid"
)

// NewEmailProvider creates an email provider based on configuration
func NewEmailProvider(ctx context.Context, cfg *config.EmailConfig) (Provider, error) {
	switch cfg.Provider {
	case ProviderSMTP:
		if cfg.SMTP.Host == "" || cfg.SMTP.Port == 0 || cfg.SMTP.Username == "" {
			return nil, fmt.Errorf("SMTP host, port, and username are required")
		}
		return NewSMTPProvider(cfg.SMTP)

	case ProviderSendGrid:
		if cfg.SendGrid.APIKey == "" {
			return nil, fmt.Errorf("SendGrid API key is required")
		}
		return NewSendGridProvider(cfg.SendGrid)

	default:
		return nil, fmt.Errorf("unsupported email provider: %s", cfg.Provider)
	}
}

package email

import (
	"context"
	"watch-party/pkg/logger"
)

// NoOpProvider implements the Provider interface but does absolutely nothing
// Perfect for freemium mode or when emails should be completely disabled
type NoOpProvider struct {
	mode string
}

// NewNoOpProvider creates a new no-op email provider
func NewNoOpProvider(mode string) *NoOpProvider {
	return &NoOpProvider{
		mode: mode,
	}
}

// SendEmail does nothing gracefully
func (n *NoOpProvider) SendEmail(ctx context.Context, to []string, subject string, body EmailBody) error {
	// Silently succeed - no logging to avoid spam
	return nil
}

// SendTemplateEmail does nothing gracefully
func (n *NoOpProvider) SendTemplateEmail(ctx context.Context, to []string, templateName string, data interface{}) error {
	// Silently succeed - no logging to avoid spam
	return nil
}

// ValidateProvider always succeeds
func (n *NoOpProvider) ValidateProvider(ctx context.Context) error {
	if n.mode != "silent" {
		logger.Infof("📭 Email provider disabled (mode: %s) - emails will be silently ignored", n.mode)
	}
	return nil
}

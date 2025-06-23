package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"watch-party/pkg/config"
)

// SendGridProvider implements Provider for SendGrid
type SendGridProvider struct {
	config config.SendGridConfig
	client *http.Client
}

// NewSendGridProvider creates a new SendGrid email provider
func NewSendGridProvider(cfg config.SendGridConfig) (*SendGridProvider, error) {
	return &SendGridProvider{
		config: cfg,
		client: &http.Client{},
	}, nil
}

// SendEmail sends an email using SendGrid API
func (sg *SendGridProvider) SendEmail(ctx context.Context, to []string, subject string, body EmailBody) error {
	// prepare recipients
	personalizations := make([]map[string]interface{}, 0)
	for _, recipient := range to {
		personalizations = append(personalizations, map[string]interface{}{
			"to": []map[string]string{
				{"email": recipient},
			},
		})
	}

	// prepare content
	content := make([]map[string]string, 0)
	if body.Text != "" {
		content = append(content, map[string]string{
			"type":  "text/plain",
			"value": body.Text,
		})
	}
	if body.HTML != "" {
		content = append(content, map[string]string{
			"type":  "text/html",
			"value": body.HTML,
		})
	}

	// if no content provided, use a default text
	if len(content) == 0 {
		content = append(content, map[string]string{
			"type":  "text/plain",
			"value": "This email was sent from WatchParty",
		})
	}

	// create the email payload
	payload := map[string]interface{}{
		"personalizations": personalizations,
		"from": map[string]string{
			"email": sg.config.FromEmail,
			"name":  sg.config.FromName,
		},
		"subject": subject,
		"content": content,
	}

	// convert to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal email payload: %w", err)
	}

	// create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", strings.NewReader(string(jsonPayload)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// set headers
	req.Header.Set("Authorization", "Bearer "+sg.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	// send request
	resp, err := sg.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	defer resp.Body.Close()

	// check response
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SendGrid API returned status %d", resp.StatusCode)
	}

	return nil
}

// SendTemplateEmail sends an email using a template
func (sg *SendGridProvider) SendTemplateEmail(ctx context.Context, to []string, templateName string, data interface{}) error {
	body, err := renderTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	subject := getTemplateSubject(templateName, data)
	return sg.SendEmail(ctx, to, subject, body)
}

// ValidateProvider validates the SendGrid configuration
func (sg *SendGridProvider) ValidateProvider(ctx context.Context) error {
	if sg.config.APIKey == "" {
		return fmt.Errorf("SendGrid API key is required")
	}
	if sg.config.FromEmail == "" {
		return fmt.Errorf("SendGrid from email is required")
	}
	if sg.config.FromName == "" {
		log.Println("Warning: SendGrid from name is not set, using default")
		sg.config.FromName = "WatchParty"
	}
	return nil
}

package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"watch-party/pkg/config"
)

// SMTPProvider implements Provider for generic SMTP servers
type SMTPProvider struct {
	config config.SMTPConfig
}

// NewSMTPProvider creates a new SMTP email provider
func NewSMTPProvider(cfg config.SMTPConfig) (*SMTPProvider, error) {
	// set default port if not specified
	if cfg.Port == 0 {
		cfg.Port = 587
	}

	return &SMTPProvider{
		config: cfg,
	}, nil
}

// SendEmail sends an email using SMTP
func (s *SMTPProvider) SendEmail(ctx context.Context, to []string, subject string, body EmailBody) error {
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)

	// create message
	msg := s.createMessage(s.config.Username, to, subject, body)

	// send email
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	if s.config.UseTLS {
		// use STARTTLS for security (most common for SMTP)
		client, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server: %w", err)
		}
		defer client.Quit()

		// start TLS if supported
		ok, _ := client.Extension("STARTTLS")
		if ok {
			tlsConfig := &tls.Config{
				InsecureSkipVerify: false,
				ServerName:         s.config.Host,
			}
			err = client.StartTLS(tlsConfig)
			if err != nil {
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}

		err = client.Auth(auth)
		if err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}

		err = client.Mail(s.config.Username)
		if err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		for _, recipient := range to {
			err = client.Rcpt(recipient)
			if err != nil {
				return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
			}
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}

		_, err = w.Write([]byte(msg))
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		return w.Close()
	} else {
		// use plain SMTP without TLS
		return smtp.SendMail(addr, auth, s.config.Username, to, []byte(msg))
	}
}

// SendTemplateEmail sends an email using a template
func (s *SMTPProvider) SendTemplateEmail(ctx context.Context, to []string, templateName string, data interface{}) error {
	body, err := renderTemplate(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	subject := getTemplateSubject(templateName, data)
	return s.SendEmail(ctx, to, subject, body)
}

// ValidateProvider validates the SMTP configuration
func (s *SMTPProvider) ValidateProvider(ctx context.Context) error {
	if s.config.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if s.config.Port == 0 {
		return fmt.Errorf("SMTP port is required")
	}
	if s.config.Username == "" {
		return fmt.Errorf("SMTP username is required")
	}
	if s.config.Password == "" {
		return fmt.Errorf("SMTP password is required")
	}
	return nil
}

// createMessage creates an email message
func (s *SMTPProvider) createMessage(from string, to []string, subject string, body EmailBody) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("From: %s\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ",")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if body.HTML != "" && body.Text != "" {
		// multipart message
		boundary := "boundary-watchparty-email"
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary))

		// text part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		msg.WriteString(body.Text)
		msg.WriteString("\r\n\r\n")

		// hTML part
		msg.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		msg.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		msg.WriteString(body.HTML)
		msg.WriteString("\r\n\r\n")

		msg.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if body.HTML != "" {
		// hTML only
		msg.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
		msg.WriteString(body.HTML)
	} else {
		// text only
		msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
		msg.WriteString(body.Text)
	}

	return msg.String()
}

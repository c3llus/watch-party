package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
)

// template names
const (
	TemplateRoomInvitation = "room_invitation"
)

// renderTemplate renders an email template with the given data
func renderTemplate(templateName string, data interface{}) (EmailBody, error) {
	switch templateName {
	case TemplateRoomInvitation:
		return renderRoomInvitationTemplate(data)
	default:
		return EmailBody{}, fmt.Errorf("unknown template: %s", templateName)
	}
}

// getTemplateSubject returns the subject for a given template
func getTemplateSubject(templateName string, data interface{}) string {
	switch templateName {
	case TemplateRoomInvitation:
		inviteData, ok := data.(InvitationTemplateData)
		if ok {
			return fmt.Sprintf("🎬 Join %s to watch %s on WatchParty!", inviteData.InviterName, inviteData.MovieTitle)
		}
		return "You're invited to a WatchParty!"
	default:
		return "WatchParty Notification"
	}
}

// renderRoomInvitationTemplate renders the room invitation email template
func renderRoomInvitationTemplate(data interface{}) (EmailBody, error) {
	inviteData, ok := data.(InvitationTemplateData)
	if !ok {
		return EmailBody{}, fmt.Errorf("invalid template data type for room invitation")
	}

	// render HTML
	htmlTmpl, err := template.New("html").Parse(invitationTemplateHTML)
	if err != nil {
		return EmailBody{}, fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var htmlBuf bytes.Buffer
	err = htmlTmpl.Execute(&htmlBuf, inviteData)
	if err != nil {
		return EmailBody{}, fmt.Errorf("failed to execute HTML template: %w", err)
	}

	// render Text
	textTmpl, err := template.New("text").Parse(invitationTextTemplate)
	if err != nil {
		return EmailBody{}, fmt.Errorf("failed to parse text template: %w", err)
	}

	var textBuf bytes.Buffer
	err = textTmpl.Execute(&textBuf, inviteData)
	if err != nil {
		return EmailBody{}, fmt.Errorf("failed to execute text template: %w", err)
	}

	return EmailBody{
		HTML: strings.TrimSpace(htmlBuf.String()),
		Text: strings.TrimSpace(textBuf.String()),
	}, nil
}

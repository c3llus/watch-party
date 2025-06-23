package email

const (
	invitationTemplateHTML string = `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
		<title>WatchParty Invitation</title>
	</head>
	<body>
		<div>
			<div>
				<img src="https://c3llus.dev/favicon.svg" alt="Logo" width="32" height="32">
				<h1>You're Invited to a Watch Party!</h1>
			</div>
			<p>Hi there!</p>
			<p>{{.InviterName}} has invited you to watch a movie together on {{.AppName}}.</p>
			<p>Movie: {{.MovieTitle}}</p>
			<p>
				<a href="{{.InviteURL}}">Join Watch Party</a>
			</p>
			<p>Or copy and paste this link in your browser:</p>
			<p>{{.InviteURL}}</p>
			<p>Ready to sync up and enjoy the movie together? Click the button above to join the party!</p>
			<p>This invitation was sent by {{.AppName}}</p>
			<p>If you didn't expect this invitation, you can safely ignore this email.</p>
		</div>
	</body>
	</html>`

	invitationTextTemplate = `
		{{.AppName}} - Watch Party Invitation

		Hi there!

		{{.InviterName}} has invited you to watch a movie together on {{.AppName}}.

		Movie: {{.MovieTitle}}
		Invited by: {{.InviterName}}
		{{if .ExpiresAt}}Invitation expires: {{.ExpiresAt}}{{end}}

		Join the watch party by clicking this link:
		{{.InviteURL}}

		Ready to sync up and enjoy the movie together? Copy the link above and paste it in your browser to join the party!

		---
		This invitation was sent by {{.AppName}}
		If you didn't expect this invitation, you can safely ignore this email.
	`
)

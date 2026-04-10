package services

import "github.com/orkestra/backend/internal/core/notification/models"

// defaultTemplate bundles a TemplateDoc payload for seeding.
type defaultTemplate struct {
	TemplateID  string
	Locale      string
	Subject     string
	BodyText    string
	BodyHTML    string
	Description string
	Variables   []string
}

// defaultTemplates are seeded into the DB on module Start() if missing.
// They are plain text/template strings — Go's text/template + html/template
// pipelines render them at dispatch time.
var defaultTemplates = []defaultTemplate{
	{
		TemplateID:  models.CategoryAuthVerifyEmail,
		Locale:      "en",
		Subject:     "Verify your {{.AppName}} email",
		Description: "Sent on signup to confirm the user's email address.",
		Variables:   []string{"AppName", "UserName", "VerifyURL", "ExpiresIn", "SupportEmail", "UnsubscribeURL", "PreferencesURL"},
		BodyText: `Hi {{.UserName}},

Welcome to {{.AppName}}. Please verify your email address by visiting the link below:

{{.VerifyURL}}

This link expires in {{.ExpiresIn}}.

If you did not create an account, you can safely ignore this email.

Need help? Contact {{.SupportEmail}}.

— The {{.AppName}} team

---
You received this email because someone created an account with this address.
Manage preferences: {{.PreferencesURL}}
Unsubscribe from marketing: {{.UnsubscribeURL}}
You will still receive security-related emails.`,
		BodyHTML: `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Verify your email</title></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;max-width:560px;margin:0 auto;padding:32px 24px;color:#333;">
  <h2 style="color:#2c3e50;">Welcome to {{.AppName}}</h2>
  <p>Hi {{.UserName}},</p>
  <p>Please confirm your email address to finish setting up your account.</p>
  <p style="margin:32px 0;">
    <a href="{{.VerifyURL}}" style="background:#2c7be5;color:#fff;padding:12px 24px;text-decoration:none;border-radius:4px;display:inline-block;font-weight:600;">Verify email</a>
  </p>
  <p style="color:#6c757d;font-size:14px;">This link expires in {{.ExpiresIn}}.</p>
  <p style="color:#6c757d;font-size:14px;">If the button doesn't work, paste this URL in your browser:<br><span style="word-break:break-all;">{{.VerifyURL}}</span></p>
  <p style="color:#6c757d;font-size:14px;">If you did not create an account, you can safely ignore this email.</p>
  <hr style="border:none;border-top:1px solid #e0e0e0;margin:32px 0;">
  <p style="color:#9ca3af;font-size:12px;">You received this email because someone created an account with this address.<br>
  <a href="{{.PreferencesURL}}" style="color:#9ca3af;">Manage preferences</a> &middot;
  <a href="{{.UnsubscribeURL}}" style="color:#9ca3af;">Unsubscribe from marketing</a><br>
  You will still receive security-related emails.</p>
</body>
</html>`,
	},
	{
		TemplateID:  models.CategoryAuthResetPassword,
		Locale:      "en",
		Subject:     "Reset your {{.AppName}} password",
		Description: "Sent when the user requests a password reset.",
		Variables:   []string{"AppName", "UserName", "ResetURL", "ExpiresIn", "SupportEmail", "RequestIP", "UnsubscribeURL", "PreferencesURL"},
		BodyText: `Hi {{.UserName}},

We received a request to reset your {{.AppName}} password. Use the link below within {{.ExpiresIn}} to pick a new one:

{{.ResetURL}}

If you did not request a password reset, ignore this email and your password will remain unchanged. You may want to review your account activity.

Requested from IP: {{.RequestIP}}

Need help? Contact {{.SupportEmail}}.

— The {{.AppName}} team

---
Manage preferences: {{.PreferencesURL}}
You will still receive security-related emails.`,
		BodyHTML: `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>Reset your password</title></head>
<body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;max-width:560px;margin:0 auto;padding:32px 24px;color:#333;">
  <h2 style="color:#2c3e50;">Reset your password</h2>
  <p>Hi {{.UserName}},</p>
  <p>We received a request to reset your {{.AppName}} password. Click the button below to pick a new one.</p>
  <p style="margin:32px 0;">
    <a href="{{.ResetURL}}" style="background:#2c7be5;color:#fff;padding:12px 24px;text-decoration:none;border-radius:4px;display:inline-block;font-weight:600;">Reset password</a>
  </p>
  <p style="color:#6c757d;font-size:14px;">This link expires in {{.ExpiresIn}}.</p>
  <p style="color:#6c757d;font-size:14px;">If the button doesn't work, paste this URL in your browser:<br><span style="word-break:break-all;">{{.ResetURL}}</span></p>
  <p style="color:#6c757d;font-size:14px;">If you did not request a password reset, ignore this email and your password will remain unchanged. You may want to review your account activity.</p>
  <p style="color:#6c757d;font-size:14px;">Requested from IP: <code>{{.RequestIP}}</code></p>
  <hr style="border:none;border-top:1px solid #e0e0e0;margin:32px 0;">
  <p style="color:#9ca3af;font-size:12px;"><a href="{{.PreferencesURL}}" style="color:#9ca3af;">Manage preferences</a><br>You will still receive security-related emails.</p>
</body>
</html>`,
	},
}

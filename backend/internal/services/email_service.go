package services

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"

	"github.com/nimbus/backend/internal/repository"
)

// SMTPConfig holds SMTP connection settings
type SMTPConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
	Enabled   bool
}

type EmailService struct {
	settingsRepo *repository.SettingsRepository
}

func NewEmailService(settingsRepo *repository.SettingsRepository) *EmailService {
	return &EmailService{settingsRepo: settingsRepo}
}

// GetSMTPConfig loads SMTP config from system_settings with env var fallback
func (s *EmailService) GetSMTPConfig(ctx context.Context) (*SMTPConfig, error) {
	config := &SMTPConfig{}

	// Try loading from system_settings first
	settings, err := s.settingsRepo.GetAll(ctx)
	if err == nil {
		settingsMap := make(map[string]string)
		for _, setting := range settings {
			settingsMap[setting.Key] = setting.Value
		}

		if v, ok := settingsMap["smtp_host"]; ok && v != "" {
			config.Host = v
		}
		if v, ok := settingsMap["smtp_port"]; ok && v != "" {
			if port, err := strconv.Atoi(v); err == nil {
				config.Port = port
			}
		}
		if v, ok := settingsMap["smtp_username"]; ok {
			config.Username = v
		}
		if v, ok := settingsMap["smtp_password"]; ok {
			config.Password = v
		}
		if v, ok := settingsMap["smtp_from_email"]; ok && v != "" {
			config.FromEmail = v
		}
		if v, ok := settingsMap["smtp_from_name"]; ok && v != "" {
			config.FromName = v
		}
		if v, ok := settingsMap["smtp_enabled"]; ok {
			config.Enabled = v == "true"
		}
	}

	// Fall back to env vars for any unset values
	if config.Host == "" {
		config.Host = os.Getenv("SMTP_HOST")
	}
	if config.Port == 0 {
		if portStr := os.Getenv("SMTP_PORT"); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				config.Port = port
			}
		}
	}
	if config.Username == "" {
		config.Username = os.Getenv("SMTP_USERNAME")
	}
	if config.Password == "" {
		config.Password = os.Getenv("SMTP_PASSWORD")
	}
	if config.FromEmail == "" {
		config.FromEmail = os.Getenv("SMTP_FROM_EMAIL")
	}
	if config.FromName == "" {
		config.FromName = os.Getenv("SMTP_FROM_NAME")
		if config.FromName == "" {
			config.FromName = "Nimbus"
		}
	}

	// If env vars provide config but enabled wasn't set, auto-enable
	if !config.Enabled && config.Host != "" {
		config.Enabled = true
	}

	return config, nil
}

// SMTPStatus describes where SMTP configuration comes from
type SMTPStatus struct {
	Configured bool   `json:"configured"`
	Source     string `json:"source"` // "env", "database", or "none"
}

// GetSMTPStatus checks whether SMTP is configured and from which source
func (s *EmailService) GetSMTPStatus(ctx context.Context) *SMTPStatus {
	// Check database settings first
	settings, err := s.settingsRepo.GetAll(ctx)
	if err == nil {
		for _, setting := range settings {
			if setting.Key == "smtp_host" && setting.Value != "" {
				return &SMTPStatus{Configured: true, Source: "database"}
			}
		}
	}

	// Check env vars
	if os.Getenv("SMTP_HOST") != "" {
		return &SMTPStatus{Configured: true, Source: "env"}
	}

	return &SMTPStatus{Configured: false, Source: "none"}
}

// SendEmail sends an email via SMTP
func (s *EmailService) SendEmail(ctx context.Context, to, subject, htmlBody string) error {
	config, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	if !config.Enabled || config.Host == "" {
		return errors.New("SMTP is not configured")
	}

	if config.Port == 0 {
		config.Port = 587
	}

	from := config.FromEmail
	if from == "" {
		return errors.New("SMTP from email is not configured")
	}

	// Build email headers and body
	headers := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		config.FromName, from, to, subject)
	msg := []byte(headers + htmlBody)

	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

	// Use TLS for port 465, STARTTLS for others
	if config.Port == 465 {
		return s.sendWithImplicitTLS(config, addr, from, to, msg)
	}

	return s.sendWithSTARTTLS(config, addr, from, to, msg)
}

// sendWithSTARTTLS sends email using STARTTLS (port 587)
func (s *EmailService) sendWithSTARTTLS(config *SMTPConfig, addr, from, to string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer c.Close()

	// STARTTLS
	tlsConfig := &tls.Config{ServerName: config.Host}
	if err := c.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	// Auth if credentials provided
	if config.Username != "" && config.Password != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close email writer: %w", err)
	}

	return c.Quit()
}

// sendWithImplicitTLS sends email using implicit TLS (port 465)
func (s *EmailService) sendWithImplicitTLS(config *SMTPConfig, addr, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{ServerName: config.Host}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect via TLS: %w", err)
	}

	c, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer c.Close()

	// Auth if credentials provided
	if config.Username != "" && config.Password != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	if err := c.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM failed: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("SMTP RCPT TO failed: %w", err)
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA failed: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close email writer: %w", err)
	}

	return c.Quit()
}

// SendPasswordResetEmail sends a password reset email
func (s *EmailService) SendPasswordResetEmail(ctx context.Context, to, rawToken, userName, frontendURL string) error {
	resetLink := frontendURL + "/reset-password?token=" + rawToken

	data := struct {
		UserName  string
		ResetLink string
	}{
		UserName:  userName,
		ResetLink: resetLink,
	}

	var body strings.Builder
	if err := passwordResetTemplate.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.SendEmail(ctx, to, "Reset your Nimbus password", body.String())
}

// TestConnection tests the SMTP connection
func (s *EmailService) TestConnection(ctx context.Context) error {
	config, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}

	if !config.Enabled || config.Host == "" {
		return errors.New("SMTP is not configured")
	}

	if config.Port == 0 {
		config.Port = 587
	}

	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

	// Test TLS connection for port 465
	if config.Port == 465 {
		tlsConfig := &tls.Config{ServerName: config.Host}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect via TLS: %w", err)
		}
		c, err := smtp.NewClient(conn, config.Host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer c.Close()

		if config.Username != "" && config.Password != "" {
			auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("SMTP authentication failed: %w", err)
			}
		}
		return c.Quit()
	}

	// Test STARTTLS connection
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer c.Close()

	tlsConfig := &tls.Config{ServerName: config.Host}
	if err := c.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("STARTTLS failed: %w", err)
	}

	if config.Username != "" && config.Password != "" {
		auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}

	return c.Quit()
}

var passwordResetTemplate = template.Must(template.New("password_reset").Parse(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; color: #333;">
  <h2 style="color: #1a1a1a;">Reset your password</h2>
  <p>Hi {{.UserName}},</p>
  <p>We received a request to reset your Nimbus password. Click the button below to set a new password:</p>
  <p style="text-align: center; margin: 30px 0;">
    <a href="{{.ResetLink}}" style="background-color: #3b82f6; color: #fff; padding: 12px 24px; text-decoration: none; border-radius: 6px; display: inline-block; font-weight: 500;">Reset Password</a>
  </p>
  <p>This link will expire in <strong>1 hour</strong>. If the link has expired, you can request a new one using "Forgot password?" on the login page.</p>
  <p>If you didn't request a password reset, you can safely ignore this email.</p>
  <hr style="border: none; border-top: 1px solid #e5e5e5; margin: 30px 0;">
  <p style="font-size: 12px; color: #999;">This email was sent by Nimbus. If the button doesn't work, copy and paste this link into your browser: {{.ResetLink}}</p>
</body>
</html>`))

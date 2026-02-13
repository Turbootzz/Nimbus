package services

import (
	"context"
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"mime"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nimbus/backend/internal/repository"
)

const smtpDialTimeout = 10 * time.Second

// SMTPConfig holds SMTP connection settings
type SMTPConfig struct {
	Host       string
	Port       int
	Username   string
	Password   string
	FromEmail  string
	FromName   string
	Enabled    bool
	enabledSet bool // tracks whether Enabled was explicitly configured
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
			config.enabledSet = true
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

	// Auto-enable only when Enabled was not explicitly configured
	if !config.enabledSet && config.Host != "" {
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

	// Sanitize header values to prevent CRLF injection
	from = sanitizeHeader(from)
	to = sanitizeHeader(to)
	subject = sanitizeHeader(subject)

	// Build email headers and body (RFC 2047 encode From name for special characters)
	encodedName := mime.QEncoding.Encode("utf-8", sanitizeHeader(config.FromName))
	headers := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		encodedName, from, to, subject)
	msg := []byte(headers + htmlBody)

	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

	// Use TLS for port 465, STARTTLS for others
	if config.Port == 465 {
		return s.sendWithImplicitTLS(config, addr, from, to, msg)
	}

	return s.sendWithSTARTTLS(config, addr, from, to, msg)
}

// sanitizeHeader strips CR and LF characters to prevent CRLF header injection
func sanitizeHeader(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// sendWithSTARTTLS sends email using STARTTLS (port 587)
func (s *EmailService) sendWithSTARTTLS(config *SMTPConfig, addr, from, to string, msg []byte) error {
	dialer := net.Dialer{Timeout: smtpDialTimeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	c, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer c.Close()

	// Check if server supports STARTTLS before attempting
	if ok, _ := c.Extension("STARTTLS"); !ok {
		return errors.New("SMTP server does not support STARTTLS")
	}

	tlsConfig := &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}
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
	tlsConfig := &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}
	dialer := net.Dialer{Timeout: smtpDialTimeout}

	conn, err := tls.DialWithDialer(&dialer, "tcp", addr, tlsConfig)
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
		UserName    string
		ResetLink   string
		FrontendURL string
	}{
		UserName:    userName,
		ResetLink:   resetLink,
		FrontendURL: frontendURL,
	}

	var body strings.Builder
	if err := passwordResetTemplate.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	return s.SendEmail(ctx, to, "Reset your Nimbus password", body.String())
}

// TestConnection tests the SMTP connection using saved config
func (s *EmailService) TestConnection(ctx context.Context) error {
	config, err := s.GetSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SMTP config: %w", err)
	}
	return s.TestConnectionWithConfig(config)
}

// TestConnectionWithConfig tests the SMTP connection using the provided config
func (s *EmailService) TestConnectionWithConfig(config *SMTPConfig) error {
	if !config.Enabled || config.Host == "" {
		return errors.New("SMTP is not configured")
	}

	if config.Port == 0 {
		config.Port = 587
	}

	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))

	dialer := net.Dialer{Timeout: smtpDialTimeout}

	// Test TLS connection for port 465
	if config.Port == 465 {
		tlsConfig := &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}
		conn, err := tls.DialWithDialer(&dialer, "tcp", addr, tlsConfig)
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
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	c, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer c.Close()

	if ok, _ := c.Extension("STARTTLS"); !ok {
		return errors.New("SMTP server does not support STARTTLS")
	}

	tlsConfig := &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}
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

//go:embed templates/password_reset.html
var passwordResetHTML string

var passwordResetTemplate = template.Must(template.New("password_reset").Parse(passwordResetHTML))

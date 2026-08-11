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

const (
	smtpDialTimeout    = 10 * time.Second
	smtpSessionTimeout = 30 * time.Second
)

// TLS modes for outgoing SMTP connections
const (
	TLSModeSTARTTLS = "starttls" // plaintext connection upgraded via STARTTLS
	TLSModeImplicit = "tls"      // TLS from the first byte, typically port 465
	TLSModeNone     = "none"     // no encryption, for trusted local relays only
)

// SMTPConfig holds SMTP connection settings
type SMTPConfig struct {
	Host          string
	Port          int
	Username      string
	Password      string
	FromEmail     string
	FromName      string
	Enabled       bool
	TLSMode       string // one of the TLSMode constants; empty derives from the port
	TLSSkipVerify bool   // skip certificate verification (self-signed relays)
	enabledSet    bool   // tracks whether Enabled was explicitly configured
}

// IsValidTLSMode reports whether mode is supported; empty means "derive from port"
func IsValidTLSMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", TLSModeSTARTTLS, TLSModeImplicit, TLSModeNone:
		return true
	}
	return false
}

// resolveTLSMode normalizes mode; unset falls back to implicit TLS on 465, STARTTLS elsewhere
func resolveTLSMode(mode string, port int) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case TLSModeSTARTTLS:
		return TLSModeSTARTTLS
	case TLSModeImplicit:
		return TLSModeImplicit
	case TLSModeNone:
		return TLSModeNone
	}
	if port == 465 {
		return TLSModeImplicit
	}
	return TLSModeSTARTTLS
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
	skipVerifySet := false

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
		if v, ok := settingsMap["smtp_tls_mode"]; ok && v != "" {
			config.TLSMode = v
		}
		if v, ok := settingsMap["smtp_tls_skip_verify"]; ok && v != "" {
			config.TLSSkipVerify = v == "true"
			skipVerifySet = true
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
	if config.FromEmail == "" {
		config.FromEmail = os.Getenv("SMTP_FROM_EMAIL")
	}
	if config.FromName == "" {
		config.FromName = os.Getenv("SMTP_FROM_NAME")
		if config.FromName == "" {
			config.FromName = "Nimbus"
		}
	}
	if config.TLSMode == "" {
		config.TLSMode = os.Getenv("SMTP_TLS_MODE")
	}
	if !skipVerifySet {
		config.TLSSkipVerify = os.Getenv("SMTP_TLS_SKIP_VERIFY") == "true"
	}

	// Credentials are unusable without encryption, so don't resurrect env
	// leftovers that the admin cleared for an unencrypted relay
	if resolveTLSMode(config.TLSMode, config.Port) != TLSModeNone {
		if config.Username == "" {
			config.Username = os.Getenv("SMTP_USERNAME")
		}
		if config.Password == "" {
			config.Password = os.Getenv("SMTP_PASSWORD")
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

	if err := prepareSMTPConfig(config); err != nil {
		return err
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

	c, err := dialSMTP(ctx, config)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := authenticateSMTP(c, config); err != nil {
		return err
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

// sanitizeHeader strips CR and LF characters to prevent CRLF header injection
func sanitizeHeader(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// prepareSMTPConfig validates a config and applies the port and TLS mode defaults.
func prepareSMTPConfig(config *SMTPConfig) error {
	if !config.Enabled || config.Host == "" {
		return errors.New("SMTP is not configured")
	}

	if config.Port == 0 {
		config.Port = 587
	}
	config.TLSMode = resolveTLSMode(config.TLSMode, config.Port)

	// Credentials would travel in cleartext without encryption
	if config.TLSMode == TLSModeNone && (config.Username != "" || config.Password != "") {
		return errors.New("SMTP authentication requires TLS: use the starttls or tls mode, or clear the username and password")
	}

	return nil
}

func tlsConfigFor(config *SMTPConfig) *tls.Config {
	return &tls.Config{
		ServerName:         config.Host,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: config.TLSSkipVerify, //nolint:gosec // opt-in for self-signed relays
	}
}

// dialSMTP opens an SMTP client using the config's TLS mode. Caller must close it.
func dialSMTP(ctx context.Context, config *SMTPConfig) (*smtp.Client, error) {
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	dialer := &net.Dialer{Timeout: smtpDialTimeout}

	if config.TLSMode == TLSModeImplicit {
		tlsDialer := tls.Dialer{NetDialer: dialer, Config: tlsConfigFor(config)}
		conn, err := tlsDialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("failed to connect via TLS: %w", err)
		}
		return newSMTPClient(ctx, conn, config.Host)
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	c, err := newSMTPClient(ctx, conn, config.Host)
	if err != nil {
		return nil, err
	}

	if config.TLSMode == TLSModeNone {
		return c, nil
	}

	if ok, _ := c.Extension("STARTTLS"); !ok {
		c.Close()
		return nil, errors.New("SMTP server does not support STARTTLS")
	}
	if err := c.StartTLS(tlsConfigFor(config)); err != nil {
		c.Close()
		return nil, fmt.Errorf("STARTTLS failed: %w", err)
	}

	return c, nil
}

func newSMTPClient(ctx context.Context, conn net.Conn, host string) (*smtp.Client, error) {
	// A TLS-only port accepts the connection but never sends a plaintext
	// greeting, so bound the session instead of blocking forever
	deadline := time.Now().Add(smtpSessionTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set SMTP connection deadline: %w", err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create SMTP client: %w", err)
	}
	return c, nil
}

func authenticateSMTP(c *smtp.Client, config *SMTPConfig) error {
	if config.Username == "" || config.Password == "" {
		return nil
	}

	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}
	return nil
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
	return s.TestConnectionWithConfig(ctx, config)
}

// TestConnectionWithConfig tests the SMTP connection using the provided config
func (s *EmailService) TestConnectionWithConfig(ctx context.Context, config *SMTPConfig) error {
	if err := prepareSMTPConfig(config); err != nil {
		return err
	}

	c, err := dialSMTP(ctx, config)
	if err != nil {
		return err
	}
	defer c.Close()

	if err := authenticateSMTP(c, config); err != nil {
		return err
	}

	return c.Quit()
}

//go:embed templates/password_reset.html
var passwordResetHTML string

var passwordResetTemplate = template.Must(template.New("password_reset").Parse(passwordResetHTML))

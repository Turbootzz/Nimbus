package services

import (
	"bufio"
	"context"
	"database/sql"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nimbus/backend/internal/repository"

	_ "github.com/mattn/go-sqlite3"
)

func setupEmailTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS system_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by TEXT
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func TestGetSMTPConfig_FromDatabase(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	// Clear env vars
	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_PORT")
	os.Unsetenv("SMTP_USERNAME")
	os.Unsetenv("SMTP_PASSWORD")
	os.Unsetenv("SMTP_FROM_EMAIL")
	os.Unsetenv("SMTP_FROM_NAME")

	// Insert DB settings
	settings := map[string]string{
		"smtp_host":       "db.smtp.com",
		"smtp_port":       "465",
		"smtp_username":   "dbuser",
		"smtp_password":   "dbpass",
		"smtp_from_email": "db@example.com",
		"smtp_from_name":  "DBName",
		"smtp_enabled":    "true",
	}
	for k, v := range settings {
		_, err := db.Exec("INSERT INTO system_settings (key, value) VALUES (?, ?)", k, v)
		if err != nil {
			t.Fatalf("Failed to insert setting %s: %v", k, err)
		}
	}

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.Host != "db.smtp.com" {
		t.Errorf("Expected host 'db.smtp.com', got '%s'", config.Host)
	}
	if config.Port != 465 {
		t.Errorf("Expected port 465, got %d", config.Port)
	}
	if config.Username != "dbuser" {
		t.Errorf("Expected username 'dbuser', got '%s'", config.Username)
	}
	if !config.Enabled {
		t.Error("Expected enabled to be true")
	}
}

func TestGetSMTPConfig_FromEnvVars(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Setenv("SMTP_HOST", "env.smtp.com")
	os.Setenv("SMTP_PORT", "587")
	os.Setenv("SMTP_USERNAME", "envuser")
	os.Setenv("SMTP_PASSWORD", "envpass")
	os.Setenv("SMTP_FROM_EMAIL", "env@example.com")
	os.Setenv("SMTP_FROM_NAME", "EnvName")
	defer func() {
		os.Unsetenv("SMTP_HOST")
		os.Unsetenv("SMTP_PORT")
		os.Unsetenv("SMTP_USERNAME")
		os.Unsetenv("SMTP_PASSWORD")
		os.Unsetenv("SMTP_FROM_EMAIL")
		os.Unsetenv("SMTP_FROM_NAME")
	}()

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.Host != "env.smtp.com" {
		t.Errorf("Expected host 'env.smtp.com', got '%s'", config.Host)
	}
	if config.Username != "envuser" {
		t.Errorf("Expected username 'envuser', got '%s'", config.Username)
	}
	if !config.Enabled {
		t.Error("Expected auto-enabled when host is set via env")
	}
}

func TestGetSMTPConfig_ExplicitDisabledNotOverridden(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Unsetenv("SMTP_HOST")

	// Insert DB settings with enabled=false but host present
	settings := map[string]string{
		"smtp_host":    "db.smtp.com",
		"smtp_enabled": "false",
	}
	for k, v := range settings {
		_, err := db.Exec("INSERT INTO system_settings (key, value) VALUES (?, ?)", k, v)
		if err != nil {
			t.Fatalf("Failed to insert setting %s: %v", k, err)
		}
	}

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.Host != "db.smtp.com" {
		t.Errorf("Expected host 'db.smtp.com', got '%s'", config.Host)
	}
	if config.Enabled {
		t.Error("Expected enabled to remain false when explicitly set in DB")
	}
}

func TestGetSMTPConfig_DefaultFromName(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Unsetenv("SMTP_HOST")
	os.Unsetenv("SMTP_FROM_NAME")

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.FromName != "Nimbus" {
		t.Errorf("Expected default FromName 'Nimbus', got '%s'", config.FromName)
	}
}

func TestGetSMTPStatus_None(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Unsetenv("SMTP_HOST")

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	status := svc.GetSMTPStatus(context.Background())
	if status.Source != "none" {
		t.Errorf("Expected source 'none', got '%s'", status.Source)
	}
	if status.Configured {
		t.Error("Expected configured false")
	}
}

func TestGetSMTPStatus_Env(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Setenv("SMTP_HOST", "smtp.test.com")
	defer os.Unsetenv("SMTP_HOST")

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	status := svc.GetSMTPStatus(context.Background())
	if status.Source != "env" {
		t.Errorf("Expected source 'env', got '%s'", status.Source)
	}
	if !status.Configured {
		t.Error("Expected configured true")
	}
}

func TestGetSMTPStatus_Database(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	os.Unsetenv("SMTP_HOST")

	_, err := db.Exec("INSERT INTO system_settings (key, value) VALUES (?, ?)", "smtp_host", "smtp.db.com")
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	status := svc.GetSMTPStatus(context.Background())
	if status.Source != "database" {
		t.Errorf("Expected source 'database', got '%s'", status.Source)
	}
}

func TestSanitizeHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"clean string", "Hello World", "Hello World"},
		{"with CR", "Hello\rWorld", "HelloWorld"},
		{"with LF", "Hello\nWorld", "HelloWorld"},
		{"with CRLF", "Hello\r\nWorld", "HelloWorld"},
		{"injection attempt", "test\r\nBcc: attacker@evil.com", "testBcc: attacker@evil.com"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeHeader(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeHeader(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTestConnectionWithConfig_NotConfigured(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	err := svc.TestConnectionWithConfig(&SMTPConfig{})
	if err == nil {
		t.Error("Expected error for unconfigured SMTP")
	}
	if err.Error() != "SMTP is not configured" {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestTestConnectionWithConfig_DisabledSMTP(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	repo := repository.NewSettingsRepository(db)
	svc := NewEmailService(repo)

	err := svc.TestConnectionWithConfig(&SMTPConfig{
		Host:    "smtp.example.com",
		Enabled: false,
	})
	if err == nil {
		t.Error("Expected error for disabled SMTP")
	}
}

func TestIsValidTLSMode(t *testing.T) {
	valid := []string{"", TLSModeSTARTTLS, TLSModeImplicit, TLSModeNone}
	for _, mode := range valid {
		if !IsValidTLSMode(mode) {
			t.Errorf("Expected '%s' to be valid", mode)
		}
	}

	// Values are normalized, so case and surrounding spaces are accepted
	if !IsValidTLSMode("  STARTTLS ") {
		t.Error("Expected normalized input to be valid")
	}

	invalid := []string{"ssl", "off", "true"}
	for _, mode := range invalid {
		if IsValidTLSMode(mode) {
			t.Errorf("Expected '%s' to be invalid", mode)
		}
	}
}

func TestResolveTLSMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		port int
		want string
	}{
		{"unset on 465 defaults to implicit TLS", "", 465, TLSModeImplicit},
		{"unset on 587 defaults to STARTTLS", "", 587, TLSModeSTARTTLS},
		{"unset on 1025 defaults to STARTTLS", "", 1025, TLSModeSTARTTLS},
		{"explicit none wins over port", TLSModeNone, 465, TLSModeNone},
		{"explicit STARTTLS wins over port", TLSModeSTARTTLS, 465, TLSModeSTARTTLS},
		{"explicit implicit TLS on custom port", TLSModeImplicit, 2465, TLSModeImplicit},
		{"case and spaces are normalized", "  NONE ", 587, TLSModeNone},
		{"unknown value falls back to port default", "garbage", 465, TLSModeImplicit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveTLSMode(tt.mode, tt.port); got != tt.want {
				t.Errorf("resolveTLSMode(%q, %d) = %q, want %q", tt.mode, tt.port, got, tt.want)
			}
		})
	}
}

func TestPrepare_AppliesDefaults(t *testing.T) {
	config := &SMTPConfig{Host: "smtp.example.com", Enabled: true}

	if err := prepareSMTPConfig(config); err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	if config.Port != 587 {
		t.Errorf("Expected default port 587, got %d", config.Port)
	}
	if config.TLSMode != TLSModeSTARTTLS {
		t.Errorf("Expected default mode starttls, got %s", config.TLSMode)
	}
}

func TestPrepare_RejectsAuthWithoutTLS(t *testing.T) {
	config := &SMTPConfig{
		Host:     "relay.internal",
		Port:     1025,
		Username: "user",
		Password: "secret",
		Enabled:  true,
		TLSMode:  TLSModeNone,
	}

	err := prepareSMTPConfig(config)
	if err == nil {
		t.Fatal("Expected error when credentials are used without TLS")
	}
	if !strings.Contains(err.Error(), "requires TLS") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestPrepare_AllowsNoTLSWithoutCredentials(t *testing.T) {
	config := &SMTPConfig{Host: "mailpit", Port: 1025, Enabled: true, TLSMode: TLSModeNone}

	if err := prepareSMTPConfig(config); err != nil {
		t.Fatalf("Expected no error for credential-less plaintext relay, got: %v", err)
	}
}

func TestGetSMTPConfig_TLSSettingsFromDatabase(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_TLS_MODE", "")
	t.Setenv("SMTP_TLS_SKIP_VERIFY", "")

	settings := map[string]string{
		"smtp_host":            "mailpit",
		"smtp_port":            "1025",
		"smtp_enabled":         "true",
		"smtp_tls_mode":        TLSModeNone,
		"smtp_tls_skip_verify": "true",
	}
	for k, v := range settings {
		if _, err := db.Exec("INSERT INTO system_settings (key, value) VALUES (?, ?)", k, v); err != nil {
			t.Fatalf("Failed to insert setting %s: %v", k, err)
		}
	}

	svc := NewEmailService(repository.NewSettingsRepository(db))
	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.TLSMode != TLSModeNone {
		t.Errorf("Expected TLS mode 'none', got '%s'", config.TLSMode)
	}
	if !config.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be true")
	}
}

func TestGetSMTPConfig_TLSSettingsFromEnv(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	t.Setenv("SMTP_HOST", "mailpit")
	t.Setenv("SMTP_PORT", "1025")
	t.Setenv("SMTP_TLS_MODE", TLSModeNone)
	t.Setenv("SMTP_TLS_SKIP_VERIFY", "true")

	svc := NewEmailService(repository.NewSettingsRepository(db))
	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.TLSMode != TLSModeNone {
		t.Errorf("Expected TLS mode 'none', got '%s'", config.TLSMode)
	}
	if !config.TLSSkipVerify {
		t.Error("Expected TLSSkipVerify to be true")
	}
}

func TestGetSMTPConfig_DatabaseSkipVerifyOverridesEnv(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	t.Setenv("SMTP_HOST", "mailpit")
	t.Setenv("SMTP_TLS_SKIP_VERIFY", "true")

	if _, err := db.Exec("INSERT INTO system_settings (key, value) VALUES (?, ?)", "smtp_tls_skip_verify", "false"); err != nil {
		t.Fatalf("Failed to insert setting: %v", err)
	}

	svc := NewEmailService(repository.NewSettingsRepository(db))
	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.TLSSkipVerify {
		t.Error("Expected database value 'false' to override env var")
	}
}

// startFakeSMTP starts a plaintext SMTP server that never advertises STARTTLS.
// The returned channel receives the body of the first delivered message.
func startFakeSMTP(t *testing.T) (host string, port int, delivered <-chan string) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start fake SMTP server: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	out := make(chan string, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		write := func(s string) {
			conn.Write([]byte(s + "\r\n"))
		}

		write("220 fake ESMTP")

		var body strings.Builder
		inData := false

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")

			if inData {
				if line == "." {
					inData = false
					write("250 OK")
					out <- body.String()
					continue
				}
				body.WriteString(line + "\n")
				continue
			}

			switch {
			case strings.HasPrefix(line, "DATA"):
				inData = true
				write("354 Send data")
			case strings.HasPrefix(line, "QUIT"):
				write("221 Bye")
				return
			default:
				write("250 OK")
			}
		}
	}()

	return "127.0.0.1", ln.Addr().(*net.TCPAddr).Port, out
}

func TestSendEmail_PlaintextRelay(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	host, port, delivered := startFakeSMTP(t)

	t.Setenv("SMTP_HOST", host)
	t.Setenv("SMTP_PORT", strconv.Itoa(port))
	t.Setenv("SMTP_FROM_EMAIL", "noreply@example.com")
	t.Setenv("SMTP_TLS_MODE", TLSModeNone)
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")

	svc := NewEmailService(repository.NewSettingsRepository(db))

	if err := svc.SendEmail(context.Background(), "user@example.com", "Test subject", "<p>Hello</p>"); err != nil {
		t.Fatalf("SendEmail failed: %v", err)
	}

	select {
	case msg := <-delivered:
		if !strings.Contains(msg, "Subject: Test subject") {
			t.Errorf("Delivered message missing subject:\n%s", msg)
		}
		if !strings.Contains(msg, "<p>Hello</p>") {
			t.Errorf("Delivered message missing body:\n%s", msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timed out waiting for delivered message")
	}
}

func TestSendEmail_STARTTLSRequiredOnServerWithoutSupport(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	host, port, _ := startFakeSMTP(t)

	t.Setenv("SMTP_HOST", host)
	t.Setenv("SMTP_PORT", strconv.Itoa(port))
	t.Setenv("SMTP_FROM_EMAIL", "noreply@example.com")
	t.Setenv("SMTP_TLS_MODE", TLSModeSTARTTLS)
	t.Setenv("SMTP_USERNAME", "")
	t.Setenv("SMTP_PASSWORD", "")

	svc := NewEmailService(repository.NewSettingsRepository(db))

	err := svc.SendEmail(context.Background(), "user@example.com", "Test subject", "<p>Hello</p>")
	if err == nil {
		t.Fatal("Expected STARTTLS mode to fail against a server without STARTTLS")
	}
	if !strings.Contains(err.Error(), "does not support STARTTLS") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetSMTPConfig_NoTLSIgnoresEnvCredentials(t *testing.T) {
	db := setupEmailTestDB(t)
	defer db.Close()

	t.Setenv("SMTP_HOST", "mailpit")
	t.Setenv("SMTP_PORT", "1025")
	t.Setenv("SMTP_TLS_MODE", TLSModeNone)
	t.Setenv("SMTP_USERNAME", "leftover")
	t.Setenv("SMTP_PASSWORD", "leftover")

	svc := NewEmailService(repository.NewSettingsRepository(db))
	config, err := svc.GetSMTPConfig(context.Background())
	if err != nil {
		t.Fatalf("GetSMTPConfig failed: %v", err)
	}

	if config.Username != "" || config.Password != "" {
		t.Errorf("Expected env credentials to be ignored without TLS, got '%s'/'%s'", config.Username, config.Password)
	}
	if err := prepareSMTPConfig(config); err != nil {
		t.Errorf("Expected an unencrypted relay to be usable, got: %v", err)
	}
}

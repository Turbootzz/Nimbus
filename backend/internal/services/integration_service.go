package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
)

const (
	maxIntegrationsPerUser     = 50
	integrationTestTimeout     = 10 * time.Second
	defaultRefreshSeconds      = 60
	maxIntegrationOptionsBytes = 16 * 1024
)

// IntegrationService handles integration CRUD, credential encryption and
// connection tests. Decrypted credentials never leave this service.
type IntegrationService struct {
	repo    repository.IntegrationRepositoryInterface
	cipher  *utils.Cipher
	timeout time.Duration
}

func NewIntegrationService(repo repository.IntegrationRepositoryInterface, cipher *utils.Cipher) *IntegrationService {
	return &IntegrationService{repo: repo, cipher: cipher, timeout: integrationTestTimeout}
}

// Kinds lists the registered integration kinds
func (s *IntegrationService) Kinds() []integrations.Meta {
	return integrations.Kinds()
}

func (s *IntegrationService) List(ctx context.Context, userID string) ([]models.Integration, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *IntegrationService) Get(ctx context.Context, id, userID string) (*models.Integration, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *IntegrationService) Delete(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

// Create validates the request, encrypts the credentials and stores it
func (s *IntegrationService) Create(ctx context.Context, userID string, req *models.IntegrationRequest) (*models.Integration, error) {
	in, err := s.buildInput(req, nil)
	if err != nil {
		return nil, err
	}

	// The ID is set here because the credentials are bound to it
	integration := in.integration
	integration.ID = uuid.New().String()
	integration.UserID = userID
	blob, err := s.encryptCredentials(in.creds, integration.ID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, &integration, blob, maxIntegrationsPerUser); err != nil {
		if errors.Is(err, repository.ErrIntegrationLimitReached) {
			return nil, invalid("Maximum integration limit reached (%d)", maxIntegrationsPerUser)
		}
		return nil, err
	}
	return &integration, nil
}

// Update applies the request on top of the stored integration. Omitted
// credentials are kept as they are. The last test result is cleared when the
// connection changes, since it no longer applies.
func (s *IntegrationService) Update(ctx context.Context, id, userID string, req *models.IntegrationRequest) (*models.Integration, error) {
	current, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	in, err := s.buildInput(req, current)
	if err != nil {
		return nil, err
	}

	var blob []byte
	if in.creds == nil {
		blob, err = s.repo.GetCredentials(ctx, id, userID)
	} else {
		blob, err = s.encryptCredentials(in.creds, current.ID)
	}
	if err != nil {
		return nil, err
	}

	integration := in.integration
	if in.creds != nil || connectionChanged(current, &integration) {
		integration.LastTestAt, integration.LastTestOK, integration.LastError = nil, nil, nil
	}
	if err := s.repo.Update(ctx, &integration, blob); err != nil {
		return nil, err
	}
	return &integration, nil
}

// TestUnsaved tests a connection from a request body without storing anything
func (s *IntegrationService) TestUnsaved(ctx context.Context, req *models.IntegrationRequest) (*models.IntegrationTestResult, error) {
	in, err := s.buildInput(req, nil)
	if err != nil {
		return nil, err
	}
	result := s.runTest(ctx, in.impl, &in.integration, *in.creds)
	return &result, nil
}

// TestSaved tests a stored integration and records the result
func (s *IntegrationService) TestSaved(ctx context.Context, id, userID string) (*models.IntegrationTestResult, error) {
	current, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	blob, err := s.repo.GetCredentials(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	var result models.IntegrationTestResult
	impl, ok := integrations.Get(current.Kind)
	creds, decryptErr := s.decryptCredentials(blob, current.ID)
	switch {
	case !ok:
		result.Error = fmt.Sprintf("Integration kind %q is not available", current.Kind)
	case decryptErr != nil:
		log.Printf("integration %s: %v", id, decryptErr)
		result.Error = "Stored credentials cannot be decrypted (was ENCRYPTION_KEY changed?). Enter them again to fix this."
	default:
		result = s.runTest(ctx, impl, current, creds)
	}

	var lastError *string
	if result.Error != "" {
		lastError = &result.Error
	}
	if err := s.repo.UpdateTestResult(ctx, id, userID, result.OK, lastError); err != nil {
		return nil, err
	}
	return &result, nil
}

// integrationInput is a validated request
type integrationInput struct {
	impl        integrations.Integration
	integration models.Integration
	creds       *models.IntegrationCredentials // nil = keep stored credentials
}

// buildInput validates req. current is the stored integration on update and
// nil otherwise; on update, empty request fields keep the stored value.
func (s *IntegrationService) buildInput(req *models.IntegrationRequest, current *models.Integration) (*integrationInput, error) {
	in := models.Integration{VerifyTLS: true, RefreshSeconds: defaultRefreshSeconds}
	if current != nil {
		in = *current
	}

	if kind := strings.TrimSpace(req.Kind); kind != "" {
		if current != nil && kind != current.Kind {
			return nil, invalid("Kind cannot be changed")
		}
		in.Kind = kind
	}
	impl, ok := integrations.Get(in.Kind)
	if !ok {
		return nil, invalid("Unknown integration kind")
	}
	meta := impl.Meta()

	if name := strings.TrimSpace(req.Name); name != "" {
		in.Name = name
	}
	if in.Name == "" {
		return nil, invalid("Name is required")
	}
	if utf8.RuneCountInString(in.Name) > 100 {
		return nil, invalid("Name must be 100 characters or less")
	}

	if rawURL := strings.TrimSpace(req.BaseURL); rawURL != "" {
		baseURL, err := normalizeBaseURL(rawURL)
		if err != nil {
			return nil, err
		}
		in.BaseURL = baseURL
	}
	if in.BaseURL == "" {
		return nil, invalid("Base URL is required")
	}

	if authType := strings.TrimSpace(req.AuthType); authType != "" {
		in.AuthType = authType
	}
	if in.AuthType == "" {
		in.AuthType = meta.AuthTypes[0]
	}
	if !slices.Contains(meta.AuthTypes, in.AuthType) {
		return nil, invalid("Auth type %q is not supported by %s", in.AuthType, meta.Name)
	}

	if req.VerifyTLS != nil {
		in.VerifyTLS = *req.VerifyTLS
	}
	if req.RefreshSeconds != nil {
		in.RefreshSeconds = *req.RefreshSeconds
	}
	if in.RefreshSeconds < minRefreshSeconds || in.RefreshSeconds > maxRefreshSeconds {
		return nil, invalid("Refresh interval must be between %d and %d seconds", minRefreshSeconds, maxRefreshSeconds)
	}

	if len(req.Options) > 0 && string(req.Options) != "null" {
		if len(req.Options) > maxIntegrationOptionsBytes {
			return nil, invalid("Options are too large")
		}
		var obj map[string]any
		if err := json.Unmarshal(req.Options, &obj); err != nil || obj == nil {
			return nil, invalid("Options must be a JSON object")
		}
		in.Options = req.Options
	}
	if len(in.Options) == 0 {
		in.Options = json.RawMessage(`{}`)
	}

	creds, err := resolveCredentials(in.AuthType, req.Credentials, current)
	if err != nil {
		return nil, err
	}
	return &integrationInput{impl: impl, integration: in, creds: creds}, nil
}

// connectionChanged reports whether an update touches how Nimbus connects
func connectionChanged(before, after *models.Integration) bool {
	return before.BaseURL != after.BaseURL ||
		before.AuthType != after.AuthType ||
		before.VerifyTLS != after.VerifyTLS ||
		!bytes.Equal(before.Options, after.Options)
}

// normalizeBaseURL validates a base URL and strips the trailing slash.
// Credentials, queries and fragments are rejected so no secret is ever
// stored in plain text or echoed back in responses.
func normalizeBaseURL(rawURL string) (string, error) {
	if err := utils.ValidateWebhookURL(rawURL); err != nil {
		return "", invalid("Invalid base URL: %s", err.Error())
	}
	if strings.ContainsAny(rawURL, "?#") {
		return "", invalid("Base URL must not contain a query or fragment")
	}
	if u, err := url.Parse(rawURL); err != nil || u.User != nil {
		return "", invalid("Base URL must not contain credentials; use the credential fields instead")
	}
	return strings.TrimRight(rawURL, "/"), nil
}

// resolveCredentials returns the credentials to store for authType, or nil
// to keep the stored ones (update without new credentials).
func resolveCredentials(authType string, given *models.IntegrationCredentials, current *models.Integration) (*models.IntegrationCredentials, error) {
	if authType == models.IntegrationAuthNone {
		return &models.IntegrationCredentials{}, nil
	}
	if given == nil {
		if current != nil && current.AuthType == authType && current.HasCredentials {
			return nil, nil
		}
		return nil, invalid("Credentials are required")
	}

	// Keep only the fields this auth type uses
	var creds models.IntegrationCredentials
	switch authType {
	case models.IntegrationAuthAPIKey:
		creds.APIKey = strings.TrimSpace(given.APIKey)
		if creds.APIKey == "" {
			return nil, invalid("API key is required")
		}
	case models.IntegrationAuthToken:
		creds.Token = strings.TrimSpace(given.Token)
		if creds.Token == "" {
			return nil, invalid("Token is required")
		}
	case models.IntegrationAuthBasic:
		creds.Username = strings.TrimSpace(given.Username)
		creds.Password = given.Password
		if creds.Username == "" {
			return nil, invalid("Username is required")
		}
	}
	return &creds, nil
}

// encryptCredentials returns the blob to store, bound to the integration ID;
// nil when there is nothing to store (auth type none).
func (s *IntegrationService) encryptCredentials(creds *models.IntegrationCredentials, integrationID string) ([]byte, error) {
	if creds == nil || *creds == (models.IntegrationCredentials{}) {
		return nil, nil
	}
	plaintext, err := json.Marshal(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to encode credentials: %w", err)
	}
	return s.cipher.Encrypt(plaintext, []byte(integrationID))
}

func (s *IntegrationService) decryptCredentials(blob []byte, integrationID string) (models.IntegrationCredentials, error) {
	var creds models.IntegrationCredentials
	if blob == nil {
		return creds, nil
	}
	plaintext, err := s.cipher.Decrypt(blob, []byte(integrationID))
	if err != nil {
		return creds, err
	}
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return creds, fmt.Errorf("failed to decode credentials: %w", err)
	}
	return creds, nil
}

// runTest calls the kind's Test with a fresh safe client and a timeout.
// Errors are redacted before they are shown or stored.
func (s *IntegrationService) runTest(ctx context.Context, impl integrations.Integration, integration *models.Integration, creds models.IntegrationCredentials) models.IntegrationTestResult {
	client := utils.NewSafeClient(integration.VerifyTLS, s.timeout)
	defer client.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	conn := &integrations.Conn{
		BaseURL:   integration.BaseURL,
		Creds:     creds,
		VerifyTLS: integration.VerifyTLS,
		Options:   integration.Options,
		Client:    client,
		State:     &integrations.State{},
	}

	start := time.Now()
	err := callTest(ctx, impl, conn)
	result := models.IntegrationTestResult{OK: err == nil, LatencyMs: time.Since(start).Milliseconds()}
	if err != nil {
		result.Error = redactSecrets(err.Error(), creds)
	}
	return result
}

// callTest turns a panic in a kind into an error so one buggy kind can't
// crash the server.
func callTest(ctx context.Context, impl integrations.Integration, conn *integrations.Conn) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("integration %s crashed: %v", impl.Kind(), r)
		}
	}()
	return impl.Test(ctx, conn)
}

// redactSecrets removes credential values from a message. Kinds may put
// secrets in URLs and net/http echoes the full URL in its errors.
func redactSecrets(msg string, creds models.IntegrationCredentials) string {
	secrets := []string{creds.APIKey, creds.Password, creds.Token}
	if creds.Password != "" {
		// Basic auth header value
		secrets = append(secrets, base64.StdEncoding.EncodeToString([]byte(creds.Username+":"+creds.Password)))
	}
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		for _, form := range []string{secret, url.QueryEscape(secret), url.PathEscape(secret)} {
			msg = strings.ReplaceAll(msg, form, "[redacted]")
		}
	}
	return msg
}

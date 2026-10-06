package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
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
	unixScheme                 = "unix://"
	// socketBaseURL is what requests over the Docker socket are sent to
	socketBaseURL = "http://docker"
)

// IntegrationService handles integration CRUD, credential encryption and
// connection tests. Decrypted credentials never leave this service.
type IntegrationService struct {
	repo    repository.IntegrationRepositoryInterface
	cipher  *utils.Cipher
	timeout time.Duration
	poller  Poller
	// dockerSocket is the socket DOCKER_SOCKET allows; empty means none
	dockerSocket string
	// states keeps each integration's session data (e.g. a login sid)
	// between polls, keyed by integration ID
	states sync.Map
}

// NewIntegrationService builds the service. dockerSocket is the unix socket
// kinds like Docker may use (DOCKER_SOCKET); empty allows none.
func NewIntegrationService(repo repository.IntegrationRepositoryInterface, cipher *utils.Cipher, dockerSocket string) *IntegrationService {
	return &IntegrationService{repo: repo, cipher: cipher, timeout: integrationTestTimeout, dockerSocket: dockerSocket}
}

// Kinds lists the integration kinds the user can set up
func (s *IntegrationService) Kinds(isAdmin bool) []integrations.Meta {
	kinds := integrations.Kinds()
	if !isAdmin {
		kinds = slices.DeleteFunc(kinds, func(m integrations.Meta) bool { return m.AdminOnly })
	}
	return kinds
}

func (s *IntegrationService) List(ctx context.Context, userID string) ([]models.Integration, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *IntegrationService) Get(ctx context.Context, id, userID string) (*models.Integration, error) {
	return s.repo.GetByID(ctx, id, userID)
}

// SetPoller lets the service tell the widget poller about changes
func (s *IntegrationService) SetPoller(p Poller) {
	s.poller = p
}

func (s *IntegrationService) Delete(ctx context.Context, id, userID string) error {
	// Prepared before the delete: ending a session needs the credentials
	var closeSession func()
	if _, open := s.states.Load(id); open {
		if current, err := s.repo.GetByID(ctx, id, userID); err == nil {
			closeSession = s.sessionCloser(ctx, current)
		}
	}
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return err
	}
	s.states.Delete(id)
	if closeSession != nil {
		closeSession()
	}
	kick(s.poller)
	return nil
}

// sessionCloser prepares ending the session the integration's kind keeps on
// its app (integrations.Closer), with its current URL and credentials; nil
// when there is none. The returned func logs out in the background, so an
// unreachable app never holds up the request; failures are only logged, as
// the app ends the session itself after a while.
func (s *IntegrationService) sessionCloser(ctx context.Context, integration *models.Integration) func() {
	if _, open := s.states.Load(integration.ID); !open {
		return nil
	}
	impl, ok := integrations.Get(integration.Kind)
	if !ok {
		return nil
	}
	closer, ok := impl.(integrations.Closer)
	if !ok {
		return nil
	}
	conn, err := s.Conn(ctx, integration)
	if err != nil {
		log.Printf("integration %s: can't end its session: %v", integration.ID, err)
		return nil
	}
	return func() {
		go func() {
			defer conn.Client.CloseIdleConnections()
			ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
			defer cancel()
			if err := callClose(ctx, closer, conn); err != nil {
				log.Printf("integration %s: could not end its session: %s", integration.ID, redactSecrets(err.Error(), conn.Creds))
			}
		}()
	}
}

// Create validates the request, encrypts the credentials and stores it
func (s *IntegrationService) Create(ctx context.Context, userID string, isAdmin bool, req *models.IntegrationRequest) (*models.Integration, error) {
	in, err := s.buildInput(req, nil, isAdmin)
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
func (s *IntegrationService) Update(ctx context.Context, id, userID string, isAdmin bool, req *models.IntegrationRequest) (*models.Integration, error) {
	current, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	in, err := s.buildInput(req, current, isAdmin)
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
	reconnect := in.creds != nil || connectionChanged(current, &integration)
	var closeSession func()
	if reconnect {
		integration.LastTestAt, integration.LastTestOK, integration.LastError = nil, nil, nil
		// The old session is ended with the old URL and credentials
		closeSession = s.sessionCloser(ctx, current)
	}
	if err := s.repo.Update(ctx, &integration, blob); err != nil {
		return nil, err
	}
	if reconnect {
		// A session from the old connection may not fit the new one
		s.states.Delete(integration.ID)
		if closeSession != nil {
			closeSession()
		}
	}
	kick(s.poller)
	return &integration, nil
}

// Conn connects to a stored integration for polling. It keeps the
// integration's session state between calls.
func (s *IntegrationService) Conn(ctx context.Context, integration *models.Integration) (*integrations.Conn, error) {
	blob, err := s.repo.GetCredentials(ctx, integration.ID, integration.UserID)
	if err != nil {
		return nil, err
	}
	creds, err := s.decryptCredentials(blob, integration.ID)
	if err != nil {
		log.Printf("integration %s: %v", integration.ID, err)
		return nil, errors.New("stored credentials cannot be decrypted (was ENCRYPTION_KEY changed?)")
	}
	state, _ := s.states.LoadOrStore(integration.ID, &integrations.State{})
	return s.newConn(integration, creds, state.(*integrations.State))
}

// Fetch polls a stored integration for its KPIs. Errors are redacted.
func (s *IntegrationService) Fetch(ctx context.Context, integration *models.Integration) (*integrations.Payload, error) {
	impl, ok := integrations.Get(integration.Kind)
	if !ok {
		return nil, fmt.Errorf("integration kind %q is not available", integration.Kind)
	}
	conn, err := s.Conn(ctx, integration)
	if err != nil {
		return nil, err
	}
	defer conn.Client.CloseIdleConnections()

	payload, err := callFetch(ctx, impl, conn)
	if err != nil {
		return nil, errors.New(redactSecrets(err.Error(), conn.Creds))
	}
	return payload, nil
}

// TestUnsaved tests a connection from a request body without storing anything
func (s *IntegrationService) TestUnsaved(ctx context.Context, isAdmin bool, req *models.IntegrationRequest) (*models.IntegrationTestResult, error) {
	in, err := s.buildInput(req, nil, isAdmin)
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
func (s *IntegrationService) buildInput(req *models.IntegrationRequest, current *models.Integration, isAdmin bool) (*integrationInput, error) {
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
	if meta.AdminOnly && !isAdmin {
		return nil, invalid("Only admins can set up %s integrations", meta.Name)
	}

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
		baseURL, err := normalizeBaseURL(rawURL, meta, s.dockerSocket)
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
// stored in plain text or echoed back in responses. Kinds that support it
// may use the Docker socket the server allows.
func normalizeBaseURL(rawURL string, meta integrations.Meta, allowed string) (string, error) {
	if path, isSocket := strings.CutPrefix(rawURL, unixScheme); isSocket {
		switch {
		case !meta.DockerSocket:
			return "", invalid("%s can't use a unix socket", meta.Name)
		case allowed == "":
			return "", invalid("Set DOCKER_SOCKET=%s on the Nimbus server to use this socket", path)
		case path != allowed:
			return "", invalid("Only the socket in DOCKER_SOCKET can be used: %s%s", unixScheme, allowed)
		}
		return rawURL, nil
	}

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

// runTest calls the kind's Test with a fresh client and a timeout.
// Errors are redacted before they are shown or stored.
func (s *IntegrationService) runTest(ctx context.Context, impl integrations.Integration, integration *models.Integration, creds models.IntegrationCredentials) models.IntegrationTestResult {
	conn, err := s.newConn(integration, creds, &integrations.State{})
	if err != nil {
		return models.IntegrationTestResult{Error: err.Error()}
	}
	defer conn.Client.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	start := time.Now()
	err = callTest(ctx, impl, conn)
	result := models.IntegrationTestResult{OK: err == nil, LatencyMs: time.Since(start).Milliseconds()}
	if err != nil {
		result.Error = redactSecrets(err.Error(), creds)
	}
	return result
}

// newConn connects over the SSRF-safe client, or over the Docker socket
// the server allows. A socket that DOCKER_SOCKET no longer names is refused.
func (s *IntegrationService) newConn(integration *models.Integration, creds models.IntegrationCredentials, state *integrations.State) (*integrations.Conn, error) {
	baseURL := integration.BaseURL
	var client *http.Client
	if path, isSocket := strings.CutPrefix(baseURL, unixScheme); isSocket {
		switch {
		case s.dockerSocket == "":
			return nil, errors.New("the Nimbus server allows no socket; set DOCKER_SOCKET")
		case path != s.dockerSocket:
			return nil, fmt.Errorf("DOCKER_SOCKET now names another socket; change the URL to %s%s", unixScheme, s.dockerSocket)
		}
		baseURL, client = socketBaseURL, utils.NewUnixSocketClient(path, s.timeout)
	} else {
		client = utils.NewSafeClient(integration.VerifyTLS, s.timeout)
	}
	return &integrations.Conn{
		BaseURL:   baseURL,
		Creds:     creds,
		VerifyTLS: integration.VerifyTLS,
		Options:   integration.Options,
		Client:    client,
		State:     state,
	}, nil
}

// recoverCrash turns a panic in a kind or widget into an error, so one
// buggy implementation can't crash the server
func recoverCrash(name string, err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("%s crashed: %v", name, r)
	}
}

func callTest(ctx context.Context, impl integrations.Integration, conn *integrations.Conn) (err error) {
	defer recoverCrash("integration "+impl.Kind(), &err)
	return impl.Test(ctx, conn)
}

func callClose(ctx context.Context, closer integrations.Closer, conn *integrations.Conn) (err error) {
	defer recoverCrash("closing a session", &err)
	return closer.Close(ctx, conn)
}

func callFetch(ctx context.Context, impl integrations.Integration, conn *integrations.Conn) (payload *integrations.Payload, err error) {
	defer recoverCrash("integration "+impl.Kind(), &err)
	return impl.Fetch(ctx, conn)
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

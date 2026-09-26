package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nimbus/backend/internal/integrations"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "sup3r-s3cret/key+value"

// fakeKind pings BaseURL/ping and puts the API key in the query string, like
// some real apps do, so redaction of net/http errors is exercised.
type fakeKind struct{}

func (fakeKind) Kind() string { return "svc-fake" }
func (fakeKind) Meta() integrations.Meta {
	return integrations.Meta{Name: "Fake", AuthTypes: []string{
		models.IntegrationAuthAPIKey, models.IntegrationAuthBasic, models.IntegrationAuthNone,
	}}
}
func (fakeKind) Test(ctx context.Context, conn *integrations.Conn) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, conn.BaseURL+"/ping?apikey="+url.QueryEscape(conn.Creds.APIKey), nil)
	if err != nil {
		return err
	}
	resp, err := conn.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
func (fakeKind) Fetch(context.Context, *integrations.Conn) (*integrations.Payload, error) {
	return &integrations.Payload{}, nil
}

type panicKind struct{ fakeKind }

func (panicKind) Kind() string                                   { return "svc-panic" }
func (panicKind) Test(context.Context, *integrations.Conn) error { panic("boom") }

func init() {
	integrations.Register(fakeKind{})
	integrations.Register(panicKind{})
}

// fakeIntegrationRepo is an in-memory IntegrationRepositoryInterface
type fakeIntegrationRepo struct {
	items map[string]models.Integration
	blobs map[string][]byte
}

func newFakeIntegrationRepo() *fakeIntegrationRepo {
	return &fakeIntegrationRepo{items: map[string]models.Integration{}, blobs: map[string][]byte{}}
}

func (r *fakeIntegrationRepo) get(id, userID string) (models.Integration, error) {
	i, ok := r.items[id]
	if !ok || i.UserID != userID {
		return i, repository.ErrIntegrationNotFound
	}
	return i, nil
}

func (r *fakeIntegrationRepo) Create(_ context.Context, i *models.Integration, blob []byte, _ int) error {
	i.ID = uuid.New().String()
	i.HasCredentials = blob != nil
	r.items[i.ID], r.blobs[i.ID] = *i, blob
	return nil
}

func (r *fakeIntegrationRepo) GetByID(_ context.Context, id, userID string) (*models.Integration, error) {
	i, err := r.get(id, userID)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *fakeIntegrationRepo) GetCredentials(_ context.Context, id, userID string) ([]byte, error) {
	if _, err := r.get(id, userID); err != nil {
		return nil, err
	}
	return r.blobs[id], nil
}

func (r *fakeIntegrationRepo) ListByUserID(_ context.Context, userID string) ([]models.Integration, error) {
	list := []models.Integration{}
	for _, i := range r.items {
		if i.UserID == userID {
			list = append(list, i)
		}
	}
	return list, nil
}

func (r *fakeIntegrationRepo) Update(_ context.Context, i *models.Integration, blob []byte) error {
	if _, err := r.get(i.ID, i.UserID); err != nil {
		return err
	}
	i.HasCredentials = blob != nil
	r.items[i.ID], r.blobs[i.ID] = *i, blob
	return nil
}

func (r *fakeIntegrationRepo) UpdateTestResult(_ context.Context, id, userID string, ok bool, lastError *string) error {
	i, err := r.get(id, userID)
	if err != nil {
		return err
	}
	now := time.Now()
	i.LastTestAt, i.LastTestOK, i.LastError = &now, &ok, lastError
	r.items[id] = i
	return nil
}

func (r *fakeIntegrationRepo) Delete(_ context.Context, id, userID string) error {
	if _, err := r.get(id, userID); err != nil {
		return err
	}
	delete(r.items, id)
	return nil
}

func newTestIntegrationService(t *testing.T) (*IntegrationService, *fakeIntegrationRepo) {
	t.Helper()
	cipher, err := utils.NewCipher(bytes.Repeat([]byte{9}, 32))
	require.NoError(t, err)
	repo := newFakeIntegrationRepo()
	return NewIntegrationService(repo, cipher), repo
}

// pingServer accepts /ping only with the right API key
func pingServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" && r.URL.Query().Get("apikey") == testSecret {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)
	return server
}

func apiKeyRequest(baseURL string) *models.IntegrationRequest {
	return &models.IntegrationRequest{
		Kind:        "svc-fake",
		Name:        "Fake",
		BaseURL:     baseURL,
		AuthType:    models.IntegrationAuthAPIKey,
		Credentials: &models.IntegrationCredentials{APIKey: testSecret},
	}
}

func decryptStored(t *testing.T, svc *IntegrationService, repo *fakeIntegrationRepo, id string) models.IntegrationCredentials {
	t.Helper()
	creds, err := svc.decryptCredentials(repo.blobs[id])
	require.NoError(t, err)
	return creds
}

func TestIntegrationService_CreateEncryptsCredentials(t *testing.T) {
	svc, repo := newTestIntegrationService(t)

	req := apiKeyRequest("http://192.168.1.10:8989/")
	req.Credentials.Password = "not-for-api-key-auth"
	integration, err := svc.Create(context.Background(), "user-1", req)
	require.NoError(t, err)

	assert.True(t, integration.HasCredentials)
	assert.Equal(t, "http://192.168.1.10:8989", integration.BaseURL, "trailing slash is stripped")
	assert.True(t, integration.VerifyTLS, "verify_tls defaults to true")
	assert.Equal(t, defaultRefreshSeconds, integration.RefreshSeconds)
	assert.JSONEq(t, `{}`, string(integration.Options))

	blob := repo.blobs[integration.ID]
	assert.NotContains(t, string(blob), testSecret, "credentials must be encrypted at rest")
	assert.Equal(t, models.IntegrationCredentials{APIKey: testSecret}, decryptStored(t, svc, repo, integration.ID),
		"only the fields of the auth type are stored")
}

func TestIntegrationService_AuthNoneStoresNoBlob(t *testing.T) {
	svc, repo := newTestIntegrationService(t)

	integration, err := svc.Create(context.Background(), "user-1", &models.IntegrationRequest{
		Kind: "svc-fake", Name: "Open", BaseURL: "http://10.0.0.5", AuthType: models.IntegrationAuthNone,
		Credentials: &models.IntegrationCredentials{APIKey: "ignored"},
	})
	require.NoError(t, err)
	assert.False(t, integration.HasCredentials)
	assert.Nil(t, repo.blobs[integration.ID])
}

func TestIntegrationService_DefaultAuthTypeIsFirstOfKind(t *testing.T) {
	svc, _ := newTestIntegrationService(t)
	req := apiKeyRequest("http://10.0.0.5")
	req.AuthType = ""

	integration, err := svc.Create(context.Background(), "user-1", req)
	require.NoError(t, err)
	assert.Equal(t, models.IntegrationAuthAPIKey, integration.AuthType)
}

func TestIntegrationService_Validation(t *testing.T) {
	svc, _ := newTestIntegrationService(t)
	five, big := 5, 100000

	type validationCase struct {
		mutate func(r *models.IntegrationRequest)
		want   string
	}
	cases := map[string]validationCase{
		"unknown kind":          {func(r *models.IntegrationRequest) { r.Kind = "nope" }, "Unknown integration kind"},
		"missing kind":          {func(r *models.IntegrationRequest) { r.Kind = "" }, "Unknown integration kind"},
		"missing name":          {func(r *models.IntegrationRequest) { r.Name = "  " }, "Name is required"},
		"name too long":         {func(r *models.IntegrationRequest) { r.Name = strings.Repeat("a", 101) }, "100 characters"},
		"missing base url":      {func(r *models.IntegrationRequest) { r.BaseURL = "" }, "Base URL is required"},
		"non-http scheme":       {func(r *models.IntegrationRequest) { r.BaseURL = "ftp://nas.lan" }, "Invalid base URL"},
		"cloud metadata host":   {func(r *models.IntegrationRequest) { r.BaseURL = "http://169.254.169.254" }, "cloud metadata"},
		"credentials in url":    {func(r *models.IntegrationRequest) { r.BaseURL = "http://admin:hunter2@nas.lan" }, "must not contain credentials"},
		"query in url":          {func(r *models.IntegrationRequest) { r.BaseURL = "http://nas.lan/?apikey=x" }, "query or fragment"},
		"unsupported auth type": {func(r *models.IntegrationRequest) { r.AuthType = models.IntegrationAuthToken }, "not supported"},
		"unknown auth type":     {func(r *models.IntegrationRequest) { r.AuthType = "oauth" }, "not supported"},
		"refresh too low":       {func(r *models.IntegrationRequest) { r.RefreshSeconds = &five }, "Refresh interval"},
		"refresh too high":      {func(r *models.IntegrationRequest) { r.RefreshSeconds = &big }, "Refresh interval"},
		"options not object":    {func(r *models.IntegrationRequest) { r.Options = json.RawMessage(`[1,2]`) }, "JSON object"},
		"options too large": {func(r *models.IntegrationRequest) {
			r.Options = json.RawMessage(`{"a":"` + strings.Repeat("x", 17000) + `"}`)
		}, "too large"},
		"missing credentials": {func(r *models.IntegrationRequest) { r.Credentials = nil }, "Credentials are required"},
		"empty api key": {func(r *models.IntegrationRequest) {
			r.Credentials = &models.IntegrationCredentials{APIKey: "  "}
		}, "API key is required"},
		"basic without username": {func(r *models.IntegrationRequest) {
			r.AuthType = models.IntegrationAuthBasic
			r.Credentials = &models.IntegrationCredentials{Password: "pw"}
		}, "Username is required"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := apiKeyRequest("http://nas.lan:8989")
			tc.mutate(req)
			_, err := svc.Create(context.Background(), "user-1", req)
			var vErr *ValidationError
			require.True(t, errors.As(err, &vErr), "expected ValidationError, got %v", err)
			assert.Contains(t, vErr.Message, tc.want)
		})
	}
}

func TestIntegrationService_NullOptionsAreIgnored(t *testing.T) {
	svc, _ := newTestIntegrationService(t)
	req := apiKeyRequest("http://nas.lan")
	req.Options = json.RawMessage(`null`)

	integration, err := svc.Create(context.Background(), "user-1", req)
	require.NoError(t, err)
	assert.JSONEq(t, `{}`, string(integration.Options))
}

func TestIntegrationService_UpdateKeepsCredentialsWhenOmitted(t *testing.T) {
	svc, repo := newTestIntegrationService(t)
	ctx := context.Background()
	created, err := svc.Create(ctx, "user-1", apiKeyRequest("http://nas.lan"))
	require.NoError(t, err)
	blobBefore := repo.blobs[created.ID]

	updated, err := svc.Update(ctx, created.ID, "user-1", &models.IntegrationRequest{Name: "Renamed"})
	require.NoError(t, err)

	assert.Equal(t, "Renamed", updated.Name)
	assert.Equal(t, "http://nas.lan", updated.BaseURL, "empty fields keep stored values")
	assert.True(t, updated.HasCredentials)
	assert.Equal(t, blobBefore, repo.blobs[created.ID], "credentials untouched")
}

func TestIntegrationService_UpdateReplacesAndClearsCredentials(t *testing.T) {
	svc, repo := newTestIntegrationService(t)
	ctx := context.Background()
	created, err := svc.Create(ctx, "user-1", apiKeyRequest("http://nas.lan"))
	require.NoError(t, err)

	_, err = svc.Update(ctx, created.ID, "user-1", &models.IntegrationRequest{
		Credentials: &models.IntegrationCredentials{APIKey: "new-key"},
	})
	require.NoError(t, err)
	assert.Equal(t, "new-key", decryptStored(t, svc, repo, created.ID).APIKey)

	updated, err := svc.Update(ctx, created.ID, "user-1", &models.IntegrationRequest{AuthType: models.IntegrationAuthNone})
	require.NoError(t, err)
	assert.False(t, updated.HasCredentials)
	assert.Nil(t, repo.blobs[created.ID])
}

func TestIntegrationService_UpdateRules(t *testing.T) {
	svc, _ := newTestIntegrationService(t)
	ctx := context.Background()
	created, err := svc.Create(ctx, "user-1", apiKeyRequest("http://nas.lan"))
	require.NoError(t, err)

	var vErr *ValidationError

	_, err = svc.Update(ctx, created.ID, "user-1", &models.IntegrationRequest{Kind: "svc-panic"})
	assert.True(t, errors.As(err, &vErr), "kind change must be rejected, got %v", err)

	_, err = svc.Update(ctx, created.ID, "user-1", &models.IntegrationRequest{AuthType: models.IntegrationAuthBasic})
	assert.True(t, errors.As(err, &vErr), "auth type change needs new credentials, got %v", err)

	_, err = svc.Update(ctx, created.ID, "user-2", &models.IntegrationRequest{Name: "Hijack"})
	assert.ErrorIs(t, err, repository.ErrIntegrationNotFound)
}

func TestIntegrationService_TestUnsaved(t *testing.T) {
	svc, repo := newTestIntegrationService(t)
	server := pingServer(t)

	result, err := svc.TestUnsaved(context.Background(), apiKeyRequest(server.URL))
	require.NoError(t, err)
	assert.True(t, result.OK, "expected success, got %q", result.Error)
	assert.Empty(t, repo.items, "unsaved test must not store anything")

	req := apiKeyRequest(server.URL)
	req.Credentials.APIKey = "wrong"
	result, err = svc.TestUnsaved(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.OK)
	assert.Contains(t, result.Error, "401")
}

func TestIntegrationService_TestErrorsAreRedacted(t *testing.T) {
	svc, _ := newTestIntegrationService(t)
	server := pingServer(t)
	deadURL := server.URL
	server.Close() // connection refused: net/http echoes the full URL incl. the key

	result, err := svc.TestUnsaved(context.Background(), apiKeyRequest(deadURL))
	require.NoError(t, err)
	assert.False(t, result.OK)
	assert.Contains(t, result.Error, "[redacted]")
	assert.NotContains(t, result.Error, testSecret)
	assert.NotContains(t, result.Error, url.QueryEscape(testSecret))
}

func TestIntegrationService_TestRecoversFromPanics(t *testing.T) {
	svc, _ := newTestIntegrationService(t)
	req := apiKeyRequest("http://nas.lan")
	req.Kind = "svc-panic"

	result, err := svc.TestUnsaved(context.Background(), req)
	require.NoError(t, err)
	assert.False(t, result.OK)
	assert.Contains(t, result.Error, "crashed")
}

func TestIntegrationService_TestSavedRecordsResult(t *testing.T) {
	svc, repo := newTestIntegrationService(t)
	ctx := context.Background()
	server := pingServer(t)
	created, err := svc.Create(ctx, "user-1", apiKeyRequest(server.URL))
	require.NoError(t, err)

	result, err := svc.TestSaved(ctx, created.ID, "user-1")
	require.NoError(t, err)
	assert.True(t, result.OK, "expected success, got %q", result.Error)

	stored := repo.items[created.ID]
	require.NotNil(t, stored.LastTestOK)
	assert.True(t, *stored.LastTestOK)
	assert.Nil(t, stored.LastError)
	assert.NotNil(t, stored.LastTestAt)

	_, err = svc.TestSaved(ctx, created.ID, "user-2")
	assert.ErrorIs(t, err, repository.ErrIntegrationNotFound)
}

func TestIntegrationService_TestSavedWithWrongKey(t *testing.T) {
	svc, repo := newTestIntegrationService(t)
	ctx := context.Background()
	created, err := svc.Create(ctx, "user-1", apiKeyRequest("http://nas.lan"))
	require.NoError(t, err)

	// Simulate a lost ENCRYPTION_KEY: same data, different key
	otherCipher, err := utils.NewCipher(bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	svc.cipher = otherCipher

	result, err := svc.TestSaved(ctx, created.ID, "user-1")
	require.NoError(t, err)
	assert.False(t, result.OK)
	assert.Contains(t, result.Error, "ENCRYPTION_KEY")
	require.NotNil(t, repo.items[created.ID].LastError)
}

func TestRedactSecrets(t *testing.T) {
	creds := models.IntegrationCredentials{APIKey: "a/b c", Username: "admin", Password: "p@ss", Token: "tok"}
	msg := redactSecrets("key a/b c, query a%2Fb+c, path a%2Fb%20c, pw p%40ss, token tok, user admin", creds)

	for _, leaked := range []string{"a/b c", "a%2Fb+c", "a%2Fb%20c", "p%40ss", "tok"} {
		assert.NotContains(t, msg, leaked)
	}
	assert.Contains(t, msg, "user admin", "usernames are not secret")
}

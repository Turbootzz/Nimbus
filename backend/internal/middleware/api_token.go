package middleware

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/utils"
)

// DefaultAPITokenRateLimit is the number of requests one token may make per minute
const DefaultAPITokenRateLimit = 600

const apiTokenRateWindow = time.Minute

// Errors of API token authentication
var (
	ErrAPITokenExpired     = errors.New("api token expired")
	ErrAPITokenRateLimited = errors.New("api token rate limit exceeded")
	errInsufficientScope   = errors.New("api token lacks the required scope")
)

// ResolveAPIToken looks up a personal access token and its owner, rejects
// expired tokens and stamps usage. It returns repository.ErrAPITokenNotFound,
// ErrAPITokenExpired, repository.ErrUserNotFound or an infrastructure error.
// Scopes and the rate limit are up to the caller.
func ResolveAPIToken(ctx context.Context, token string, userRepo *repository.UserRepository, apiTokenRepo *repository.APITokenRepository) (*models.User, *models.APIToken, error) {
	apiToken, err := apiTokenRepo.GetByTokenHash(ctx, utils.HashAPIToken(token))
	if err != nil {
		return nil, nil, err
	}
	if apiToken.Expired(time.Now()) {
		return nil, nil, ErrAPITokenExpired
	}

	user, err := userRepo.GetByID(apiToken.UserID)
	if err != nil {
		return nil, nil, err
	}

	// Best-effort usage stamp, throttled so polling clients don't write per request
	if apiToken.LastUsedAt == nil || time.Since(*apiToken.LastUsedAt) > time.Minute {
		if err := apiTokenRepo.UpdateLastUsed(ctx, apiToken.ID); err != nil {
			log.Printf("Failed to update api token last_used_at: %v", err)
		}
	}

	return user, apiToken, nil
}

// rateWindow counts the requests of one token in the current window
type rateWindow struct {
	start time.Time
	count int
}

// One limiter for REST and MCP, so a token has a single budget
var (
	apiTokenRateMu    sync.Mutex
	apiTokenRateLimit = DefaultAPITokenRateLimit
	apiTokenRates     = make(map[string]*rateWindow)
)

// SetAPITokenRateLimit sets the requests per minute a token may make
func SetAPITokenRateLimit(perMinute int) {
	apiTokenRateMu.Lock()
	defer apiTokenRateMu.Unlock()
	apiTokenRateLimit = perMinute
}

// AllowAPIToken counts a request against the token's rate limit and reports
// whether it may proceed.
// ponytail: fixed window in process memory, so a burst can straddle two
// windows and the limit is per instance; use a shared store if Nimbus ever
// runs more than one backend.
func AllowAPIToken(tokenID string) bool {
	apiTokenRateMu.Lock()
	defer apiTokenRateMu.Unlock()

	now := time.Now()
	window := apiTokenRates[tokenID]
	if window == nil || now.Sub(window.start) >= apiTokenRateWindow {
		window = &rateWindow{start: now}
		apiTokenRates[tokenID] = window
	}
	window.count++
	return window.count <= apiTokenRateLimit
}

// CleanupAPITokenRateLimit drops finished windows and returns how many
func CleanupAPITokenRateLimit() int {
	apiTokenRateMu.Lock()
	defer apiTokenRateMu.Unlock()

	now := time.Now()
	removed := 0
	for id, window := range apiTokenRates {
		if now.Sub(window.start) >= apiTokenRateWindow {
			delete(apiTokenRates, id)
			removed++
		}
	}
	return removed
}

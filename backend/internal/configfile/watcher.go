package configfile

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nimbus/backend/internal/dashdoc"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
)

// DefaultInterval is how often the file is checked for changes
const DefaultInterval = 60 * time.Second

// DefaultSettle is how long a changed file must stay the same before it is
// applied: a file that is still being written can be valid YAML already, and
// applying half of it would remove the rest of the dashboard
const DefaultSettle = 2 * time.Second

// Syncer applies a document to a user's dashboard in place
// (services.DashboardTransferService)
type Syncer interface {
	Sync(ctx context.Context, userID string, doc *dashdoc.Document, prune bool) (*dashdoc.Summary, error)
}

// Users finds the owner (repository.UserRepository). An unknown email gives
// an error.
type Users interface {
	GetByEmail(email string) (*models.User, error)
}

// Options configure a Watcher
type Options struct {
	// Path is the config file (NIMBUS_CONFIG_FILE)
	Path string
	// Prune removes what the lists of the file don't have and applies its
	// preferences (NIMBUS_CONFIG_MODE=replace); otherwise the file only adds
	// and updates
	Prune bool
	// ReadOnly locks the owner's dashboard against edits (NIMBUS_CONFIG_READONLY)
	ReadOnly bool
	// Interval defaults to DefaultInterval
	Interval time.Duration
	// Settle is how long new content must stay the same before it is applied
	// (DefaultSettle); 0 applies it at once
	Settle time.Duration
	// Lookup resolves ${VAR}; defaults to os.LookupEnv
	Lookup func(string) (string, bool)
}

// Watcher applies the config file at start and whenever it changes. A file
// with an error is logged and skipped: the dashboard stays as the last good
// file left it, and the server keeps running.
type Watcher struct {
	opts   Options
	syncer Syncer
	users  Users
	// ctx ends with Stop, so a check that waits on the database doesn't
	// hold up the shutdown
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	mu sync.RWMutex
	// hash is the content that was last handled, for good or bad; handled is
	// false while it still has to be tried again
	hash    [sha256.Size]byte
	handled bool
	// lastLog avoids repeating the same message every interval
	lastLog string
	// ownerID is the admin the file belongs to, once known, and ownerEmail
	// the address the file names them by
	ownerID, ownerEmail string
	// preferences tells whether the file sets the owner's preferences
	preferences bool
}

func NewWatcher(opts Options, syncer Syncer, users Users) *Watcher {
	if opts.Interval <= 0 {
		opts.Interval = DefaultInterval
	}
	if opts.Lookup == nil {
		opts.Lookup = os.LookupEnv
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{opts: opts, syncer: syncer, users: users, ctx: ctx, cancel: cancel, done: make(chan struct{})}
}

// Start applies the file and then keeps checking it
func (w *Watcher) Start() {
	mode := "merge"
	if w.opts.Prune {
		mode = "replace"
	}
	log.Printf("Config file: %s (mode %s, read-only %t)", w.opts.Path, mode, w.opts.ReadOnly)
	go func() {
		defer close(w.done)
		ticker := time.NewTicker(w.opts.Interval)
		defer ticker.Stop()
		for {
			w.Check(w.ctx)
			select {
			case <-ticker.C:
			case <-w.ctx.Done():
				return
			}
		}
	}()
}

// Stop ends a check that is running and waits for it
func (w *Watcher) Stop() {
	w.cancel()
	<-w.done
}

// Manages reports whether the user's dashboard comes from the config file
func (w *Watcher) Manages(userID string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return userID != "" && userID == w.ownerID
}

// Locks reports whether the user's dashboard is read-only
func (w *Watcher) Locks(userID string) bool {
	return w.opts.ReadOnly && w.Manages(userID)
}

// OwnsServices reports whether the file decides which services the user
// has: when it is the whole dashboard (replace) or the dashboard is
// read-only. Nothing else should add services there.
func (w *Watcher) OwnsServices(userID string) bool {
	return (w.opts.Prune || w.opts.ReadOnly) && w.Manages(userID)
}

// LocksPreferences reports whether the user's preferences are read-only:
// when the file sets them, which only a replace does
func (w *Watcher) LocksPreferences(userID string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.opts.ReadOnly && w.opts.Prune && w.preferences && userID != "" && userID == w.ownerID
}

// say logs a message unless it was the last one logged
func (w *Watcher) say(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if message != w.lastLog {
		w.lastLog = message
		log.Print("Config file: " + message)
	}
}

// Check applies the file when it changed since it was last handled. It never
// fails: whatever goes wrong is logged and leaves the dashboard as it is.
func (w *Watcher) Check(ctx context.Context) {
	// Every time, also for a file that can't be read: an owner who was
	// demoted or deleted must not stay locked
	w.forgetLostOwner()
	data, err := read(w.opts.Path)
	if err != nil {
		w.say("can't be read, the dashboard stays as it is: %v", err)
		return
	}
	hash := sha256.Sum256(data)
	w.mu.RLock()
	same := w.handled && hash == w.hash
	w.mu.RUnlock()
	if same {
		return
	}
	if !w.settled(ctx, hash) {
		return
	}
	// retry leaves the content to be tried again at the next check
	finish := func(retry bool) {
		w.mu.Lock()
		w.hash, w.handled = hash, !retry
		w.mu.Unlock()
	}

	file, err := Parse(data, w.opts.Lookup)
	if err != nil {
		w.say("not applied, the dashboard stays as it is: %v", err)
		finish(false)
		return
	}
	owner, err := w.users.GetByEmail(file.Owner)
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		w.say("not applied, the dashboard stays as it is: the owner could not be looked up")
		finish(true)
		return
	}
	if err != nil || owner.Role != "admin" {
		// On a fresh volume the admin is only created by the setup
		w.say("waiting for an admin with the email address %s to exist; users are never created from the file", file.Owner)
		w.mu.Lock()
		w.ownerID = ""
		w.mu.Unlock()
		finish(true)
		return
	}
	summary, err := w.syncer.Sync(ctx, owner.ID, &file.Document, w.opts.Prune)
	if err != nil {
		if ctx.Err() == nil {
			w.say("not applied, the dashboard stays as it is: %s", file.Reason(err))
		}
		// A file that is not valid stays that way; anything else may pass,
		// a limit too once the owner has fewer widgets or integrations
		var invalid *dashdoc.ValidationError
		finish(!errors.As(err, &invalid) || errors.Is(err, repository.ErrWidgetLimitReached) ||
			errors.Is(err, repository.ErrIntegrationLimitReached))
		return
	}
	// Only a file that is applied decides whose dashboard it manages and
	// what that locks
	w.mu.Lock()
	w.ownerID, w.ownerEmail, w.preferences = owner.ID, file.Owner, file.Preferences != nil
	w.mu.Unlock()
	finish(false)
	w.say("applied for %s: %s", file.Owner, describe(summary))
}

// forgetLostOwner forgets an owner who is no longer the admin with that
// email address, which lifts the lock. The file then counts as not handled,
// so it is applied when that admin is back.
func (w *Watcher) forgetLostOwner() {
	w.mu.RLock()
	id, email := w.ownerID, w.ownerEmail
	w.mu.RUnlock()
	if id == "" {
		return
	}
	owner, err := w.users.GetByEmail(email)
	if err == nil && owner.Role == "admin" && owner.ID == id {
		return
	}
	if err != nil && !errors.Is(err, repository.ErrUserNotFound) {
		// The database gave no answer: nothing is known to have changed
		return
	}
	w.mu.Lock()
	w.ownerID, w.handled = "", false
	w.mu.Unlock()
}

// settled reports whether the file still has the content with that hash
// after Options.Settle
func (w *Watcher) settled(ctx context.Context, hash [sha256.Size]byte) bool {
	if w.opts.Settle <= 0 {
		return true
	}
	select {
	case <-time.After(w.opts.Settle):
	case <-ctx.Done():
		return false
	}
	data, err := read(w.opts.Path)
	return err == nil && sha256.Sum256(data) == hash
}

// describe sums up a sync in counts; it never holds values from the file
func describe(s *dashdoc.Summary) string {
	count := func(name string, c dashdoc.Count) string {
		return fmt.Sprintf("%s +%d ~%d -%d", name, c.Created, c.Updated, c.Removed)
	}
	out := fmt.Sprintf("%s, %s, %s, %s (created, updated, removed)",
		count("groups", s.Groups), count("services", s.Services), count("widgets", s.Widgets), count("integrations", s.Integrations))
	if s.Preferences {
		out += ", preferences updated"
	}
	if s.NeedsCredentials > 0 {
		out += fmt.Sprintf(", %d integration(s) without credentials", s.NeedsCredentials)
	}
	if s.NeedsSecrets > 0 {
		out += fmt.Sprintf(", %d widget(s) without their secrets", s.NeedsSecrets)
	}
	return out
}

// read reads the file, up to just over MaxSize so Parse can refuse the rest
func read(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(io.LimitReader(file, MaxSize+1))
}

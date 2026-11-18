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
)

// DefaultInterval is how often the file is checked for changes
const DefaultInterval = 60 * time.Second

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
	// Prune removes what the file doesn't have and applies its preferences
	// (NIMBUS_CONFIG_MODE=replace); otherwise the file only adds and updates
	Prune bool
	// ReadOnly locks the owner's dashboard against edits (NIMBUS_CONFIG_READONLY)
	ReadOnly bool
	// Interval defaults to DefaultInterval
	Interval time.Duration
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
	stop   chan struct{}
	done   chan struct{}

	mu sync.RWMutex
	// hash is the content that was last handled, for good or bad; handled is
	// false while it still has to be tried again
	hash    [sha256.Size]byte
	handled bool
	// lastLog avoids repeating the same message every interval
	lastLog string
	// ownerID is the admin the file belongs to, once known
	ownerID string
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
	return &Watcher{opts: opts, syncer: syncer, users: users, stop: make(chan struct{}), done: make(chan struct{})}
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
			w.Check(context.Background())
			select {
			case <-ticker.C:
			case <-w.stop:
				return
			}
		}
	}()
}

func (w *Watcher) Stop() {
	close(w.stop)
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
	if err != nil || owner.Role != "admin" {
		// On a fresh volume the admin is only created by the setup
		w.say("waiting for an admin with the email address %s to exist; users are never created from the file", file.Owner)
		finish(true)
		return
	}
	w.mu.Lock()
	w.ownerID, w.preferences = owner.ID, file.Preferences != nil
	w.mu.Unlock()

	summary, err := w.syncer.Sync(ctx, owner.ID, &file.Document, w.opts.Prune)
	if err != nil {
		w.say("not applied, the dashboard stays as it is: %v", err)
		// A file that is not valid stays that way; anything else may pass
		var invalid *dashdoc.ValidationError
		finish(!errors.As(err, &invalid))
		return
	}
	finish(false)
	w.say("applied for %s: %s", file.Owner, describe(summary))
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

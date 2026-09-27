package services

import (
	"sort"
	"sync"

	"github.com/nimbus/backend/internal/models"
)

// DashboardCache holds the latest snapshot of every polled source, per user
type DashboardCache struct {
	mu     sync.RWMutex
	byUser map[string]map[string]models.Snapshot // user ID, then snapshot key
}

func NewDashboardCache() *DashboardCache {
	return &DashboardCache{byUser: map[string]map[string]models.Snapshot{}}
}

// Get returns one snapshot
func (c *DashboardCache) Get(userID, key string) (models.Snapshot, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snap, ok := c.byUser[userID][key]
	return snap, ok
}

// Set stores a snapshot, replacing the previous one of its source
func (c *DashboardCache) Set(snap models.Snapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byUser[snap.UserID] == nil {
		c.byUser[snap.UserID] = map[string]models.Snapshot{}
	}
	c.byUser[snap.UserID][snap.Key()] = snap
}

// ForUser returns a user's snapshots sorted by key
func (c *DashboardCache) ForUser(userID string) []models.Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snaps := make([]models.Snapshot, 0, len(c.byUser[userID]))
	for _, snap := range c.byUser[userID] {
		snaps = append(snaps, snap)
	}
	sort.Slice(snaps, func(a, b int) bool { return snaps[a].Key() < snaps[b].Key() })
	return snaps
}

// Retain drops every snapshot whose key is not in keep
func (c *DashboardCache) Retain(keep map[string]bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for userID, snaps := range c.byUser {
		for key := range snaps {
			if !keep[key] {
				delete(snaps, key)
			}
		}
		if len(snaps) == 0 {
			delete(c.byUser, userID)
		}
	}
}

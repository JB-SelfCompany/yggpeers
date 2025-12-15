package yggpeers

import (
	"sync"
	"time"
)

// Cache stores cached peers with TTL
type Cache struct {
	peers     []*Peer
	fetchedAt time.Time
	ttl       time.Duration
	mu        sync.RWMutex
}

// NewCache creates a new cache with the given TTL
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl: ttl,
	}
}

// Get returns cached peers or nil if cache has expired
func (c *Cache) Get() ([]*Peer, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.peers == nil {
		return nil, false
	}

	if time.Since(c.fetchedAt) > c.ttl {
		return nil, false
	}

	// Return copy to prevent external modification
	return append([]*Peer(nil), c.peers...), true
}

// Set saves peers to cache
func (c *Cache) Set(peers []*Peer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.peers = peers
	c.fetchedAt = time.Now()
}

// Clear clears the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.peers = nil
}

// IsValid checks if cache is valid
func (c *Cache) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.peers == nil {
		return false
	}

	return time.Since(c.fetchedAt) <= c.ttl
}

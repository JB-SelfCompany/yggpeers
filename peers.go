package yggpeers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Default batching parameters for peer discovery
// Optimal for most mobile/home connections (10-100 Mbps)
const (
	DefaultBatchSize     = 20  // Peers per batch
	DefaultConcurrency   = 20  // Concurrent checks
	DefaultBatchPauseMs  = 200 // Pause between batches (ms)
)

// Manager manages the list of peers and cache
type Manager struct {
	primaryURL   string
	fallbackURL  string
	cache        *Cache
	httpClient   *http.Client
	checkTimeout time.Duration
	mu           sync.RWMutex
}

// SetupOption is a functional option for Manager
type SetupOption func(*Manager)

// WithCacheTTL sets the cache TTL
func WithCacheTTL(ttl time.Duration) SetupOption {
	return func(m *Manager) {
		m.cache = NewCache(ttl)
	}
}

// WithTimeout sets the timeout for availability checks
func WithTimeout(timeout time.Duration) SetupOption {
	return func(m *Manager) {
		m.checkTimeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(client *http.Client) SetupOption {
	return func(m *Manager) {
		m.httpClient = client
	}
}

// WithCustomURLs sets custom primary and fallback URLs
func WithCustomURLs(primary, fallback string) SetupOption {
	return func(m *Manager) {
		m.primaryURL = primary
		m.fallbackURL = fallback
	}
}

// NewManager creates a new peers manager
func NewManager(opts ...SetupOption) *Manager {
	m := &Manager{
		primaryURL:   "https://publicpeers.neilalexander.dev/publicnodes.json",
		fallbackURL:  "https://peers.yggdrasil.link/publicnodes.json",
		cache:        NewCache(30 * time.Minute), // Default 30 min TTL
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		checkTimeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// GetPeers returns all peers (using cache)
func (m *Manager) GetPeers(ctx context.Context) ([]*Peer, error) {
	// Check cache first
	if peers, ok := m.cache.Get(); ok {
		return peers, nil
	}

	// Fetch from remote
	raw, err := m.fetchJSON(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch peers: %w", err)
	}

	peers, err := parsePeers(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse peers: %w", err)
	}

	// Save to cache
	m.cache.Set(peers)

	return peers, nil
}

// GetAvailablePeers returns checked available peers
func (m *Manager) GetAvailablePeers(ctx context.Context, filter *FilterOptions) ([]*Peer, error) {
	// Get all peers
	peers, err := m.GetPeers(ctx)
	if err != nil {
		return nil, err
	}

	// Pre-filter (protocol, region only - NOT onlyUp, onlyAvailable)
	preFilter := &FilterOptions{}
	if filter != nil {
		preFilter.Protocols = filter.Protocols
		preFilter.Regions = filter.Regions
	}
	filtered := m.FilterPeers(peers, preFilter, SortByRTT)

	// Check availability
	err = m.CheckPeers(ctx, filtered, 20) // 20 concurrent checks
	if err != nil {
		return nil, fmt.Errorf("failed to check peers: %w", err)
	}

	// Post-filter (onlyAvailable, onlyUp, RTT range)
	result := make([]*Peer, 0)
	for _, peer := range filtered {
		if matchesFilter(peer, filter) {
			result = append(result, peer)
		}
	}

	return result, nil
}

// GetBestPeers returns N best peers by RTT
func (m *Manager) GetBestPeers(ctx context.Context, count int, filter *FilterOptions) ([]*Peer, error) {
	peers, err := m.GetAvailablePeers(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Sort by RTT
	sorted := m.FilterPeers(peers, filter, SortByRTT)

	if len(sorted) > count {
		return sorted[:count], nil
	}

	return sorted, nil
}

// RefreshCache forces cache refresh
func (m *Manager) RefreshCache(ctx context.Context) error {
	m.cache.Clear()
	_, err := m.GetPeers(ctx)
	return err
}

// ClearCache clears the cache
func (m *Manager) ClearCache() {
	m.cache.Clear()
}

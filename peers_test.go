package yggpeers

import (
	"context"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.primaryURL == "" {
		t.Error("Primary URL not set")
	}

	if mgr.fallbackURL == "" {
		t.Error("Fallback URL not set")
	}

	if mgr.cache == nil {
		t.Error("Cache not initialized")
	}
}

func TestNewManagerWithOptions(t *testing.T) {
	mgr := NewManager(
		WithCacheTTL(10*time.Minute),
		WithTimeout(3*time.Second),
	)

	if mgr == nil {
		t.Fatal("NewManager with options returned nil")
	}

	if mgr.checkTimeout != 3*time.Second {
		t.Errorf("Timeout not set correctly: got %v, want %v", mgr.checkTimeout, 3*time.Second)
	}
}

func TestGetPeers(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	peers, err := mgr.GetPeers(ctx)
	if err != nil {
		t.Logf("Failed to fetch peers (may be offline): %v", err)
		t.Skip("Skipping test - network unavailable")
		return
	}

	if len(peers) == 0 {
		t.Fatal("No peers returned")
	}

	t.Logf("Fetched %d peers", len(peers))

	// Verify peer structure
	for i, peer := range peers {
		if peer.Address == "" {
			t.Errorf("Peer %d has empty address", i)
		}
		if peer.Protocol == "" {
			t.Errorf("Peer %d has empty protocol", i)
		}
		if peer.Host == "" {
			t.Errorf("Peer %d has empty host", i)
		}
		if peer.Port == "" {
			t.Errorf("Peer %d has empty port", i)
		}

		if i >= 5 {
			break // Only check first 5 peers
		}
	}
}

func TestCache(t *testing.T) {
	cache := NewCache(1 * time.Second)

	// Test empty cache
	_, ok := cache.Get()
	if ok {
		t.Error("Cache should be invalid when empty")
	}

	// Test set and get
	peers := []*Peer{{Address: "test://localhost:1234"}}
	cache.Set(peers)

	got, ok := cache.Get()
	if !ok {
		t.Fatal("Cache should be valid after Set")
	}

	if len(got) != 1 {
		t.Fatalf("Expected 1 peer, got %d", len(got))
	}

	if got[0].Address != "test://localhost:1234" {
		t.Errorf("Peer address mismatch: got %s, want %s", got[0].Address, "test://localhost:1234")
	}

	// Test TTL expiration
	time.Sleep(2 * time.Second)
	_, ok = cache.Get()
	if ok {
		t.Error("Cache should have expired")
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		address  string
		protocol Protocol
		host     string
		port     string
		wantErr  bool
	}{
		{"tcp://1.2.3.4:443", ProtocolTCP, "1.2.3.4", "443", false},
		{"tls://example.com:443", ProtocolTLS, "example.com", "443", false},
		{"quic://[::1]:12345", ProtocolQUIC, "::1", "12345", false},
		{"ws://localhost:8080", ProtocolWS, "localhost", "8080", false},
		{"wss://example.com:443", ProtocolWSS, "example.com", "443", false},
		{"invalid", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			peer := &Peer{Address: tt.address}
			err := parseAddress(peer)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if peer.Protocol != tt.protocol {
				t.Errorf("Protocol: got %v, want %v", peer.Protocol, tt.protocol)
			}

			if peer.Host != tt.host {
				t.Errorf("Host: got %v, want %v", peer.Host, tt.host)
			}

			if peer.Port != tt.port {
				t.Errorf("Port: got %v, want %v", peer.Port, tt.port)
			}
		})
	}
}

func TestFilterPeers(t *testing.T) {
	mgr := NewManager()

	peers := []*Peer{
		{Protocol: ProtocolTCP, Region: "germany", RTT: 50 * time.Millisecond, Available: true, Up: true},
		{Protocol: ProtocolTLS, Region: "france", RTT: 100 * time.Millisecond, Available: true, Up: true},
		{Protocol: ProtocolQUIC, Region: "germany", RTT: 30 * time.Millisecond, Available: false, Up: true},
		{Protocol: ProtocolTCP, Region: "usa", RTT: 200 * time.Millisecond, Available: true, Up: false},
	}

	// Test protocol filter
	filter := &FilterOptions{
		Protocols: []Protocol{ProtocolTCP, ProtocolTLS},
	}
	filtered := mgr.FilterPeers(peers, filter, SortByRTT)

	if len(filtered) != 3 {
		t.Errorf("Protocol filter: expected 3 peers, got %d", len(filtered))
	}

	// Test region filter
	filter = &FilterOptions{
		Regions: []string{"germany"},
	}
	filtered = mgr.FilterPeers(peers, filter, SortByRTT)

	if len(filtered) != 2 {
		t.Errorf("Region filter: expected 2 peers, got %d", len(filtered))
	}

	// Test OnlyAvailable filter
	filter = &FilterOptions{
		OnlyAvailable: true,
	}
	filtered = mgr.FilterPeers(peers, filter, SortByRTT)

	if len(filtered) != 3 {
		t.Errorf("OnlyAvailable filter: expected 3 peers, got %d", len(filtered))
	}

	// Test OnlyUp filter
	filter = &FilterOptions{
		OnlyUp: true,
	}
	filtered = mgr.FilterPeers(peers, filter, SortByRTT)

	if len(filtered) != 3 {
		t.Errorf("OnlyUp filter: expected 3 peers, got %d", len(filtered))
	}

	// Test MaxRTT filter
	filter = &FilterOptions{
		MaxRTT: 75 * time.Millisecond,
	}
	filtered = mgr.FilterPeers(peers, filter, SortByRTT)

	if len(filtered) != 2 {
		t.Errorf("MaxRTT filter: expected 2 peers, got %d", len(filtered))
	}
}

func TestSortByRTT(t *testing.T) {
	mgr := NewManager()

	peers := []*Peer{
		{RTT: 100 * time.Millisecond, Available: true},
		{RTT: 50 * time.Millisecond, Available: true},
		{RTT: 200 * time.Millisecond, Available: true},
	}

	sorted := mgr.FilterPeers(peers, nil, SortByRTT)

	if sorted[0].RTT != 50*time.Millisecond {
		t.Errorf("Sort failed: expected first peer RTT=%v, got %v", 50*time.Millisecond, sorted[0].RTT)
	}

	if sorted[2].RTT != 200*time.Millisecond {
		t.Errorf("Sort failed: expected last peer RTT=%v, got %v", 200*time.Millisecond, sorted[2].RTT)
	}
}

func TestGetRegions(t *testing.T) {
	peers := []*Peer{
		{Region: "germany"},
		{Region: "france"},
		{Region: "germany"},
		{Region: "usa"},
	}

	regions := GetRegions(peers)

	if len(regions) != 3 {
		t.Errorf("Expected 3 unique regions, got %d", len(regions))
	}

	// Regions should be sorted
	expectedOrder := []string{"france", "germany", "usa"}
	for i, region := range regions {
		if region != expectedOrder[i] {
			t.Errorf("Region order mismatch at %d: got %s, want %s", i, region, expectedOrder[i])
		}
	}
}

func TestCacheRefresh(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	// Clear cache
	mgr.ClearCache()

	// Get peers (should fetch from network)
	peers1, err := mgr.GetPeers(ctx)
	if err != nil {
		t.Skipf("Skipping test - network unavailable: %v", err)
		return
	}

	// Get peers again (should use cache)
	peers2, err := mgr.GetPeers(ctx)
	if err != nil {
		t.Fatalf("Second GetPeers failed: %v", err)
	}

	if len(peers1) != len(peers2) {
		t.Logf("Warning: peer count changed between calls: %d vs %d", len(peers1), len(peers2))
	}

	// Refresh cache
	err = mgr.RefreshCache(ctx)
	if err != nil {
		t.Fatalf("RefreshCache failed: %v", err)
	}
}

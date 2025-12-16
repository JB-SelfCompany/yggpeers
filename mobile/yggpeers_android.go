package mobile

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jbselfcompany/yggpeers"
)

// Callback interfaces for asynchronous operations
type LogCallback interface {
	OnLog(level, message string)
}

type CheckCallback interface {
	OnPeerChecked(address string, available bool, rtt int64)
	OnCheckComplete(available int, total int)
}

type ProgressCallback interface {
	OnProgress(current, total int, availableCount int)
	OnPeerAvailable(peerJSON string)
}

// Manager wrapper for gomobile
type Manager struct {
	m           *yggpeers.Manager
	logCallback LogCallback

	// Batching configuration (can be customized via SetBatchingParams)
	// Defaults are imported from yggpeers package constants
	batchSize    int
	concurrency  int
	batchPauseMs int
}

// NewManager creates a manager with default settings
// Default batching: 20 peers per batch, 20 concurrent, 200ms pause
// Optimal for most home/mobile connections (10-100 Mbps)
func NewManager() *Manager {
	return &Manager{
		m:            yggpeers.NewManager(),
		batchSize:    yggpeers.DefaultBatchSize,
		concurrency:  yggpeers.DefaultConcurrency,
		batchPauseMs: yggpeers.DefaultBatchPauseMs,
	}
}

// NewManagerWithTTL creates a manager with custom TTL (in minutes)
func NewManagerWithTTL(ttlMinutes int) *Manager {
	return &Manager{
		m: yggpeers.NewManager(
			yggpeers.WithCacheTTL(time.Duration(ttlMinutes) * time.Minute),
		),
		batchSize:    yggpeers.DefaultBatchSize,
		concurrency:  yggpeers.DefaultConcurrency,
		batchPauseMs: yggpeers.DefaultBatchPauseMs,
	}
}

// SetBatchingParams configures batching for peer checks
func (m *Manager) SetBatchingParams(batchSize, concurrency, pauseMs int) {
	if batchSize > 0 {
		m.batchSize = batchSize
	}
	if concurrency > 0 {
		m.concurrency = concurrency
	}
	if pauseMs >= 0 {
		m.batchPauseMs = pauseMs
	}
	m.log("INFO", fmt.Sprintf("Batching configured: batch=%d, concurrency=%d, pause=%dms",
		m.batchSize, m.concurrency, m.batchPauseMs))
}

// SetLogCallback sets callback for logs
func (m *Manager) SetLogCallback(cb LogCallback) {
	m.logCallback = cb
}

func (m *Manager) log(level, message string) {
	if m.logCallback != nil {
		m.logCallback.OnLog(level, message)
	}
}

// GetPeers returns all peers (JSON string for gomobile)
func (m *Manager) GetPeers() (string, error) {
	m.log("INFO", "Fetching peers list")
	peers, err := m.m.GetPeers(context.Background())
	if err != nil {
		m.log("ERROR", "Failed to fetch peers: "+err.Error())
		return "", err
	}

	m.log("INFO", "Successfully fetched peers")
	data, err := json.Marshal(peers)
	return string(data), err
}

// GetAvailablePeers checks and returns available peers
// protocol: "tcp,tls,quic" or "" for all
// region: "germany,france" or "" for all
// maxRTTMs: maximum RTT in milliseconds (0 = no limit)
func (m *Manager) GetAvailablePeers(protocol, region string, maxRTTMs int) (string, error) {
	m.log("INFO", "Getting available peers")
	filter := &yggpeers.FilterOptions{
		OnlyAvailable: true,
	}

	if protocol != "" {
		for _, p := range strings.Split(protocol, ",") {
			filter.Protocols = append(filter.Protocols, yggpeers.Protocol(strings.TrimSpace(p)))
		}
	}

	if region != "" {
		filter.Regions = strings.Split(region, ",")
		for i := range filter.Regions {
			filter.Regions[i] = strings.TrimSpace(filter.Regions[i])
		}
	}

	if maxRTTMs > 0 {
		filter.MaxRTT = time.Duration(maxRTTMs) * time.Millisecond
	}

	peers, err := m.m.GetAvailablePeers(context.Background(), filter)
	if err != nil {
		m.log("ERROR", "Failed to get available peers: "+err.Error())
		return "", err
	}

	m.log("INFO", fmt.Sprintf("Found %d available peers", len(peers)))
	data, err := json.Marshal(peers)
	return string(data), err
}

// GetCheckedPeers checks ALL peers and returns them (both available and unavailable)
// Useful for debugging to see which peers failed
func (m *Manager) GetCheckedPeers(protocol, region string) (string, error) {
	m.log("INFO", "Getting and checking all peers")

	// Get all peers
	allPeers, err := m.m.GetPeers(context.Background())
	if err != nil {
		m.log("ERROR", "Failed to get peers: "+err.Error())
		return "", err
	}
	m.log("INFO", fmt.Sprintf("Fetched %d total peers", len(allPeers)))

	// Pre-filter by protocol/region only
	filter := &yggpeers.FilterOptions{}
	if protocol != "" {
		for _, p := range strings.Split(protocol, ",") {
			filter.Protocols = append(filter.Protocols, yggpeers.Protocol(strings.TrimSpace(p)))
		}
	}
	if region != "" {
		filter.Regions = strings.Split(region, ",")
		for i := range filter.Regions {
			filter.Regions[i] = strings.TrimSpace(filter.Regions[i])
		}
	}

	filtered := m.m.FilterPeers(allPeers, filter, yggpeers.SortByRTT)
	m.log("INFO", fmt.Sprintf("After filtering: %d peers", len(filtered)))

	// Check availability
	err = m.m.CheckPeers(context.Background(), filtered, 20)
	if err != nil {
		m.log("ERROR", "Failed to check peers: "+err.Error())
		return "", err
	}

	// Count available
	available := 0
	for _, p := range filtered {
		if p.Available {
			available++
		}
	}
	m.log("INFO", fmt.Sprintf("Check complete: %d/%d available", available, len(filtered)))

	data, err := json.Marshal(filtered)
	return string(data), err
}

// GetAvailablePeersAsync checks peers and calls callback with progress and available peers
// Uses smart batching and adaptive concurrency to avoid network saturation
func (m *Manager) GetAvailablePeersAsync(protocol, region string, maxRTTMs int, callback ProgressCallback) {
	go func() {
		m.log("INFO", "Starting async peer check with smart batching")

		// Get all peers
		allPeers, err := m.m.GetPeers(context.Background())
		if err != nil {
			m.log("ERROR", "Failed to get peers: "+err.Error())
			return
		}
		m.log("INFO", fmt.Sprintf("Fetched %d total peers", len(allPeers)))

		// Pre-filter by protocol/region
		filter := &yggpeers.FilterOptions{}
		if protocol != "" {
			for _, p := range strings.Split(protocol, ",") {
				filter.Protocols = append(filter.Protocols, yggpeers.Protocol(strings.TrimSpace(p)))
			}
		}
		if region != "" {
			filter.Regions = strings.Split(region, ",")
			for i := range filter.Regions {
				filter.Regions[i] = strings.TrimSpace(filter.Regions[i])
			}
		}

		// Sort by ResponseMS first (best peers first)
		filtered := m.m.FilterPeers(allPeers, filter, yggpeers.SortByResponseMS)
		total := len(filtered)
		m.log("INFO", fmt.Sprintf("Checking %d peers (sorted by ResponseMS)", total))

		// Get batching configuration
		batchSize := m.batchSize
		concurrency := m.concurrency
		batchPauseMs := m.batchPauseMs
		checkTimeoutMs := 3000 // 3 second timeout per check

		m.log("INFO", fmt.Sprintf("Using batching: size=%d, concurrency=%d, pause=%dms",
			batchSize, concurrency, batchPauseMs))

		availableCount := 0
		checkedCount := 0

		// Process in batches
		for batchStart := 0; batchStart < total; batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > total {
				batchEnd = total
			}
			batch := filtered[batchStart:batchEnd]
			batchLen := len(batch)

			m.log("INFO", fmt.Sprintf("Processing batch %d-%d", batchStart+1, batchEnd))

			// Check current batch with concurrency limit
			sem := make(chan struct{}, concurrency)
			done := make(chan *yggpeers.Peer, batchLen)

			for _, peer := range batch {
				go func(p *yggpeers.Peer) {
					sem <- struct{}{}
					defer func() { <-sem }()

					// Check with timeout
					ctx, cancel := context.WithTimeout(context.Background(), time.Duration(checkTimeoutMs)*time.Millisecond)
					defer cancel()

					m.m.CheckPeer(ctx, p)
					done <- p
				}(peer)
			}

			// Collect batch results
			for i := 0; i < batchLen; i++ {
				peer := <-done
				checkedCount++

				// If peer is available and meets RTT requirements
				if peer.Available {
					if maxRTTMs > 0 && peer.RTT > time.Duration(maxRTTMs)*time.Millisecond {
						// Skip peers with RTT too high
						goto reportProgress
					}

					availableCount++

					// Marshal single peer to JSON
					if callback != nil {
						data, err := json.Marshal(peer)
						if err == nil {
							callback.OnPeerAvailable(string(data))
						}
					}
				}

			reportProgress:
				// Report progress
				if callback != nil {
					callback.OnProgress(checkedCount, total, availableCount)
				}
			}

			// Pause between batches to avoid network saturation
			// Skip pause after last batch
			if batchEnd < total {
				time.Sleep(time.Duration(batchPauseMs) * time.Millisecond)
			}
		}

		m.log("INFO", fmt.Sprintf("Async check complete: %d/%d available", availableCount, total))
	}()
}

// GetBestPeers returns N best peers by RTT
func (m *Manager) GetBestPeers(count int, protocol string) (string, error) {
	m.log("INFO", "Getting best peers")
	filter := &yggpeers.FilterOptions{
		OnlyAvailable: true,
	}

	if protocol != "" {
		for _, p := range strings.Split(protocol, ",") {
			filter.Protocols = append(filter.Protocols, yggpeers.Protocol(strings.TrimSpace(p)))
		}
	}

	peers, err := m.m.GetBestPeers(context.Background(), count, filter)
	if err != nil {
		m.log("ERROR", "Failed to get best peers: "+err.Error())
		return "", err
	}

	m.log("INFO", "Found best peers")
	data, err := json.Marshal(peers)
	return string(data), err
}

// CheckPeersAsync checks peers asynchronously with callback
func (m *Manager) CheckPeersAsync(peersJSON string, callback CheckCallback) {
	go func() {
		var peers []*yggpeers.Peer
		if err := json.Unmarshal([]byte(peersJSON), &peers); err != nil {
			m.log("ERROR", "Failed to unmarshal peers: "+err.Error())
			return
		}

		m.log("INFO", "Starting async peer check")
		total := len(peers)
		available := 0

		for _, peer := range peers {
			m.m.CheckPeer(context.Background(), peer)

			if peer.Available {
				available++
			}

			if callback != nil {
				callback.OnPeerChecked(
					peer.Address,
					peer.Available,
					int64(peer.RTT.Milliseconds()),
				)
			}
		}

		m.log("INFO", "Peer check complete")
		if callback != nil {
			callback.OnCheckComplete(available, total)
		}
	}()
}

// RefreshCache updates cache
func (m *Manager) RefreshCache() error {
	m.log("INFO", "Refreshing cache")
	err := m.m.RefreshCache(context.Background())
	if err != nil {
		m.log("ERROR", "Failed to refresh cache: "+err.Error())
	}
	return err
}

// ClearCache clears the cache
func (m *Manager) ClearCache() {
	m.log("INFO", "Clearing cache")
	m.m.ClearCache()
}

// GetRegions returns list of regions (JSON array)
func (m *Manager) GetRegions() (string, error) {
	m.log("INFO", "Getting regions")
	peers, err := m.m.GetPeers(context.Background())
	if err != nil {
		m.log("ERROR", "Failed to get peers for regions: "+err.Error())
		return "", err
	}

	regions := yggpeers.GetRegions(peers)
	data, err := json.Marshal(regions)
	return string(data), err
}
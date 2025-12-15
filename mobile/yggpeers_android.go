package mobile

import (
	"context"
	"encoding/json"
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

// Manager wrapper for gomobile
type Manager struct {
	m           *yggpeers.Manager
	logCallback LogCallback
}

// NewManager creates a manager with default settings
func NewManager() *Manager {
	return &Manager{
		m: yggpeers.NewManager(),
	}
}

// NewManagerWithTTL creates a manager with custom TTL (in minutes)
func NewManagerWithTTL(ttlMinutes int) *Manager {
	return &Manager{
		m: yggpeers.NewManager(
			yggpeers.WithCacheTTL(time.Duration(ttlMinutes) * time.Minute),
		),
	}
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

	m.log("INFO", "Found available peers")
	data, err := json.Marshal(peers)
	return string(data), err
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

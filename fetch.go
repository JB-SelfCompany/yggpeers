package yggpeers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// fetchJSON loads JSON from primary URL, with fallback
func (m *Manager) fetchJSON(ctx context.Context) (RawPeersJSON, error) {
	var raw RawPeersJSON
	var lastErr error

	// Try primary
	req, err := http.NewRequestWithContext(ctx, "GET", m.primaryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
			return raw, nil
		}
		lastErr = err
	} else if err != nil {
		lastErr = err
	} else {
		resp.Body.Close()
		lastErr = fmt.Errorf("primary returned status %d", resp.StatusCode)
	}

	// Try fallback
	req, err = http.NewRequestWithContext(ctx, "GET", m.fallbackURL, nil)
	if err != nil {
		return nil, fmt.Errorf("both URLs failed, last error: %w", lastErr)
	}

	resp, err = m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("both URLs failed, last error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fallback returned status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode fallback JSON: %w", err)
	}

	return raw, nil
}

// parsePeers converts RawPeersJSON to []*Peer
func parsePeers(raw RawPeersJSON) ([]*Peer, error) {
	var peers []*Peer

	for region, peerMap := range raw {
		// region format: "germany.md" -> "germany"
		regionName := strings.TrimSuffix(region, ".md")

		for address, data := range peerMap {
			peer := &Peer{
				Address:    address,
				Region:     regionName,
				Up:         data.Up,
				Key:        data.Key,
				ResponseMS: data.ResponseMS,
				LastSeen:   data.LastSeen,
				Updated:    data.Updated,
				Imported:   data.Imported,
				ProtoMinor: data.ProtoMinor,
				Priority:   data.Priority,
			}

			// Parse protocol and host:port
			if err := parseAddress(peer); err != nil {
				continue // Skip invalid addresses
			}

			peers = append(peers, peer)
		}
	}

	return peers, nil
}

// parseAddress parses address like "tcp://host:port" into Protocol, Host, Port
func parseAddress(peer *Peer) error {
	u, err := url.Parse(peer.Address)
	if err != nil {
		return err
	}

	peer.Protocol = Protocol(u.Scheme)
	peer.Host = u.Hostname()
	peer.Port = u.Port()

	if peer.Host == "" || peer.Port == "" {
		return fmt.Errorf("invalid address format: %s", peer.Address)
	}

	return nil
}

package yggpeers

import (
	"sort"
)

// FilterPeers filters and sorts peers
func (m *Manager) FilterPeers(peers []*Peer, filter *FilterOptions, sortBy SortBy) []*Peer {
	if filter == nil {
		filter = &FilterOptions{}
	}

	// Filter
	filtered := make([]*Peer, 0, len(peers))
	for _, peer := range peers {
		if !matchesFilter(peer, filter) {
			continue
		}
		filtered = append(filtered, peer)
	}

	// Sort
	sort.Slice(filtered, func(i, j int) bool {
		return comparePeers(filtered[i], filtered[j], sortBy)
	})

	return filtered
}

// matchesFilter checks if peer matches the filter
func matchesFilter(peer *Peer, filter *FilterOptions) bool {
	// Protocol filter
	if len(filter.Protocols) > 0 {
		found := false
		for _, p := range filter.Protocols {
			if peer.Protocol == p {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Region filter
	if len(filter.Regions) > 0 {
		found := false
		for _, r := range filter.Regions {
			if peer.Region == r {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// RTT filters
	if filter.MinRTT > 0 && peer.RTT > 0 && peer.RTT < filter.MinRTT {
		return false
	}
	if filter.MaxRTT > 0 && peer.RTT > 0 && peer.RTT > filter.MaxRTT {
		return false
	}

	// Up filter
	if filter.OnlyUp && !peer.Up {
		return false
	}

	// Available filter
	if filter.OnlyAvailable && !peer.Available {
		return false
	}

	return true
}

// comparePeers compares two peers for sorting
func comparePeers(a, b *Peer, sortBy SortBy) bool {
	switch sortBy {
	case SortByRTT:
		// Unavailable peers go to the end
		if a.Available && !b.Available {
			return true
		}
		if !a.Available && b.Available {
			return false
		}
		// Both available or both unavailable - compare RTT
		if a.RTT == 0 && b.RTT == 0 {
			return false
		}
		if a.RTT == 0 {
			return false
		}
		if b.RTT == 0 {
			return true
		}
		return a.RTT < b.RTT

	case SortByResponseMS:
		return a.ResponseMS < b.ResponseMS

	case SortByAvailability:
		if a.Available != b.Available {
			return a.Available
		}
		return a.RTT < b.RTT

	case SortByLastSeen:
		return a.LastSeen > b.LastSeen

	default:
		return a.RTT < b.RTT
	}
}

// GetRegions returns a list of all unique regions
func GetRegions(peers []*Peer) []string {
	seen := make(map[string]bool)
	regions := []string{}

	for _, peer := range peers {
		if !seen[peer.Region] {
			seen[peer.Region] = true
			regions = append(regions, peer.Region)
		}
	}

	sort.Strings(regions)
	return regions
}

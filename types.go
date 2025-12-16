package yggpeers

import (
	"time"
)

// Protocol represents the type of peer protocol
type Protocol string

const (
	ProtocolTCP      Protocol = "tcp"
	ProtocolTLS      Protocol = "tls"
	ProtocolQUIC     Protocol = "quic"
	ProtocolWS       Protocol = "ws"
	ProtocolWSS      Protocol = "wss"
	ProtocolUNIX     Protocol = "unix"
	ProtocolSOCKS    Protocol = "socks"
	ProtocolSOCKSTLS Protocol = "sockstls"
)

// Peer represents information about a Yggdrasil peer from the public list
type Peer struct {
	Address  string   // Full address: "tcp://host:port"
	Protocol Protocol // Extracted protocol
	Host     string   // Host without protocol
	Port     string   // Port

	Region string // Region from JSON key: "germany.md" -> "germany"

	// Fields from JSON
	Up         bool   `json:"up"`
	Key        string `json:"key"`
	ResponseMS int    `json:"response_ms"`
	LastSeen   int64  `json:"last_seen"`
	Updated    int64  `json:"updated"`
	Imported   int64  `json:"imported"`
	ProtoMinor int    `json:"proto_minor"`
	Priority   *int   `json:"priority,omitempty"`

	// Check results
	Available  bool          // Is the peer available now
	RTT        time.Duration // Measured RTT
	CheckedAt  time.Time     // Time of check
	CheckError error         // Check error if any
}

// RawPeersJSON represents the structure of JSON from publicnodes.json
// Format: map["country.md"]map["address"]{up, key, response_ms, ...}
type RawPeersJSON map[string]map[string]struct {
	Up         bool   `json:"up"`
	Key        string `json:"key"`
	ResponseMS int    `json:"response_ms"`
	LastSeen   int64  `json:"last_seen"`
	Updated    int64  `json:"updated"`
	Imported   int64  `json:"imported"`
	ProtoMinor int    `json:"proto_minor"`
	Priority   *int   `json:"priority,omitempty"`
}

// FilterOptions defines filtering criteria
type FilterOptions struct {
	Protocols     []Protocol    // Filter by protocols
	Regions       []string      // Filter by regions
	MinRTT        time.Duration // Minimum RTT
	MaxRTT        time.Duration // Maximum RTT
	OnlyUp        bool          // Only "up" peers from JSON
	OnlyAvailable bool          // Only checked available peers
}

// SortBy defines the sorting method
type SortBy int

const (
	SortByRTT SortBy = iota
	SortByResponseMS
	SortByAvailability
	SortByLastSeen
)

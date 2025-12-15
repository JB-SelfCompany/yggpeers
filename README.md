# yggpeers

Go library for discovering and checking Yggdrasil Network public peers with mobile support (Android/iOS).

## Features

- 🌐 Load public peers from official sources with automatic fallback
- ✅ Check peer availability for all protocols: TCP, TLS, QUIC, WS, WSS
- ⚡ Measure RTT (Round Trip Time) for each peer
- 🔍 Filter peers by protocol, region, RTT, availability
- 📊 Sort peers by RTT, response time, or last seen
- 💾 Cache with configurable TTL to reduce network load
- 📱 Mobile bindings for Android/iOS via gomobile

## Installation

```bash
go get github.com/jbselfcompany/yggpeers
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/jbselfcompany/yggpeers"
)

func main() {
    mgr := yggpeers.NewManager()

    // Get best 5 TLS peers
    filter := &yggpeers.FilterOptions{
        Protocols:     []yggpeers.Protocol{yggpeers.ProtocolTLS},
        OnlyAvailable: true,
    }

    peers, _ := mgr.GetBestPeers(context.Background(), 5, filter)

    for _, peer := range peers {
        fmt.Printf("%s (RTT: %v)\n", peer.Address, peer.RTT)
    }
}
```

## Usage

### Create Manager

```go
// Default settings (30 min cache, 5s timeout)
mgr := yggpeers.NewManager()

// Custom settings
mgr := yggpeers.NewManager(
    yggpeers.WithCacheTTL(15 * time.Minute),
    yggpeers.WithTimeout(3 * time.Second),
)
```

### Get Peers

```go
// Get all peers (cached)
peers, err := mgr.GetPeers(context.Background())

// Get available peers with filter
filter := &yggpeers.FilterOptions{
    Protocols:     []yggpeers.Protocol{yggpeers.ProtocolTLS, yggpeers.ProtocolQUIC},
    Regions:       []string{"germany", "france"},
    MaxRTT:        200 * time.Millisecond,
    OnlyAvailable: true,
}
peers, err := mgr.GetAvailablePeers(context.Background(), filter)

// Get best N peers
bestPeers, err := mgr.GetBestPeers(context.Background(), 10, filter)
```

### Check Peer Availability

```go
// Check single peer
err := mgr.CheckPeer(context.Background(), peer)
if peer.Available {
    fmt.Printf("RTT: %v\n", peer.RTT)
}

// Check multiple peers in parallel
err := mgr.CheckPeers(context.Background(), peers, 10) // 10 concurrent checks
```

### Filter and Sort

```go
// Filter peers
filtered := mgr.FilterPeers(peers, filter, yggpeers.SortByRTT)

// Get available regions
regions := yggpeers.GetRegions(peers)
```

### Cache Management

```go
// Refresh cache
err := mgr.RefreshCache(context.Background())

// Clear cache
mgr.ClearCache()
```

## Mobile Usage

### Android (Kotlin)

```kotlin
import yggpeers.Mobile

val manager = Mobile.NewManager()

// Set log callback
manager.setLogCallback(object : Mobile.LogCallback {
    override fun onLog(level: String, message: String) {
        Log.d("YggPeers", "[$level] $message")
    }
})

// Get best peers
val peersJSON = manager.getBestPeers(10, "tls,quic")
val peers = JSONArray(peersJSON)

// Check peers asynchronously
manager.checkPeersAsync(peersJSON, object : Mobile.CheckCallback {
    override fun onPeerChecked(address: String, available: Boolean, rtt: Long) {
        Log.d("YggPeers", "$address: ${if (available) "✓" else "✗"}")
    }

    override fun onCheckComplete(available: Int, total: Int) {
        Log.i("YggPeers", "Complete: $available/$total available")
    }
})
```

### Building AAR

```bash
cd mobile
build-android.bat
```

Output: `yggpeers.aar` and `yggpeers-sources.jar`

## API Reference

### Manager

- `NewManager(opts ...SetupOption) *Manager` - Create new manager
- `GetPeers(ctx) ([]*Peer, error)` - Get all peers (cached)
- `GetAvailablePeers(ctx, filter) ([]*Peer, error)` - Get checked available peers
- `GetBestPeers(ctx, count, filter) ([]*Peer, error)` - Get N best peers by RTT
- `CheckPeer(ctx, peer) error` - Check single peer availability
- `CheckPeers(ctx, peers, concurrency) error` - Check multiple peers in parallel
- `FilterPeers(peers, filter, sortBy) []*Peer` - Filter and sort peers
- `RefreshCache(ctx) error` - Force cache refresh
- `ClearCache()` - Clear cache

### Setup Options

- `WithCacheTTL(ttl)` - Set cache TTL
- `WithTimeout(timeout)` - Set check timeout
- `WithHTTPClient(client)` - Set custom HTTP client
- `WithCustomURLs(primary, fallback)` - Set custom JSON sources

### Types

```go
type Peer struct {
    Address    string        // "tls://host:port"
    Protocol   Protocol      // tcp, tls, quic, ws, wss
    Host       string
    Port       string
    Region     string        // "germany"

    // From JSON
    Up         bool
    Key        string
    ResponseMS int
    LastSeen   int64

    // Check results
    Available  bool
    RTT        time.Duration
    CheckedAt  time.Time
}

type FilterOptions struct {
    Protocols     []Protocol
    Regions       []string
    MinRTT        time.Duration
    MaxRTT        time.Duration
    OnlyUp        bool
    OnlyAvailable bool
}
```

## Examples

See [examples/](examples/) directory:
- [basic.go](examples/basic.go) - Basic usage
- [filter.go](examples/filter.go) - Filtering and sorting
- [mobile.md](examples/mobile.md) - Mobile examples (Android/iOS)

## Data Sources

- Primary: https://publicpeers.neilalexander.dev/publicnodes.json
- Fallback: https://peers.yggdrasil.link/publicnodes.json

## License

GNU General Public License v3.0 (GPL-3.0)

## Contributing

Contributions are welcome! Please open an issue or pull request.

## Related Projects

- [yggdrasil-go](https://github.com/yggdrasil-network/yggdrasil-go) - Yggdrasil Network core
- [yggquic](https://github.com/Revertron/yggquic) - QUIC transport for Yggdrasil
- [yggmail](https://github.com/JB-SelfCompany/yggmail) - Mail over Yggdrasil
- [tyr](https://github.com/JB-SelfCompany/Tyr) - True P2P Email on top of Yggdrasil Network for android 

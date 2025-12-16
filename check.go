package yggpeers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/quic-go/quic-go"
)

// checkTCP checks TCP peer via net.DialTimeout
func (m *Manager) checkTCP(ctx context.Context, peer *Peer) error {
	start := time.Now()

	dialer := &net.Dialer{
		Timeout: m.checkTimeout,
	}

	addr := net.JoinHostPort(peer.Host, peer.Port)
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		peer.Available = false
		peer.CheckError = err
		return err
	}
	defer conn.Close()

	peer.RTT = time.Since(start)
	peer.Available = true
	peer.CheckedAt = time.Now()
	return nil
}

// checkTLS checks TLS peer via tls.Dial
func (m *Manager) checkTLS(ctx context.Context, peer *Peer) error {
	start := time.Now()

	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout: m.checkTimeout,
		},
		Config: &tls.Config{
			InsecureSkipVerify: true, // Yggdrasil uses crypto addressing
		},
	}

	addr := net.JoinHostPort(peer.Host, peer.Port)
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		peer.Available = false
		peer.CheckError = err
		return err
	}
	defer conn.Close()

	peer.RTT = time.Since(start)
	peer.Available = true
	peer.CheckedAt = time.Now()
	return nil
}

// checkQUIC checks QUIC peer via quic-go
func (m *Manager) checkQUIC(ctx context.Context, peer *Peer) error {
	start := time.Now()

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"yggdrasil"},
	}

	addr := net.JoinHostPort(peer.Host, peer.Port)

	// Create context with timeout
	dialCtx, cancel := context.WithTimeout(ctx, m.checkTimeout)
	defer cancel()

	conn, err := quic.DialAddr(dialCtx, addr, tlsConf, nil)
	if err != nil {
		peer.Available = false
		peer.CheckError = err
		return err
	}
	defer conn.CloseWithError(0, "check complete")

	peer.RTT = time.Since(start)
	peer.Available = true
	peer.CheckedAt = time.Now()
	return nil
}

// checkWebSocket checks WS/WSS peer via gorilla/websocket
func (m *Manager) checkWebSocket(ctx context.Context, peer *Peer) error {
	start := time.Now()

	// Construct WebSocket URL
	wsURL := peer.Address

	dialer := websocket.Dialer{
		HandshakeTimeout: m.checkTimeout,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Yggdrasil uses crypto addressing
		},
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		peer.Available = false
		peer.CheckError = err
		return err
	}
	defer conn.Close()

	peer.RTT = time.Since(start)
	peer.Available = true
	peer.CheckedAt = time.Now()
	return nil
}

// checkUNIX checks UNIX socket peer
func (m *Manager) checkUNIX(ctx context.Context, peer *Peer) error {
	start := time.Now()

	dialer := &net.Dialer{
		Timeout: m.checkTimeout,
	}

	// For UNIX sockets, the path is in the Address field after "unix://"
	// Extract the socket path from the full address
	socketPath := peer.Address[7:] // Remove "unix://" prefix

	conn, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		peer.Available = false
		peer.CheckError = err
		return err
	}
	defer conn.Close()

	peer.RTT = time.Since(start)
	peer.Available = true
	peer.CheckedAt = time.Now()
	return nil
}

// checkSOCKS checks SOCKS/SOCKSTLS peer via SOCKS5 proxy
func (m *Manager) checkSOCKS(ctx context.Context, peer *Peer) error {
	start := time.Now()

	// Parse SOCKS address format: socks://[proxyhost]:[proxyport]/[host]:[port]
	// or sockstls://[proxyhost]:[proxyport]/[host]:[port]
	addr := peer.Address
	var proxyAddr, targetAddr string
	var useTLS bool

	if peer.Protocol == ProtocolSOCKSTLS {
		useTLS = true
		addr = addr[11:] // Remove "sockstls://" prefix
	} else {
		useTLS = false
		addr = addr[8:] // Remove "socks://" prefix
	}

	// Split proxy and target addresses
	parts := strings.Split(addr, "/")
	if len(parts) != 2 {
		err := fmt.Errorf("invalid SOCKS address format: %s", peer.Address)
		peer.Available = false
		peer.CheckError = err
		return err
	}
	proxyAddr = parts[0]
	targetAddr = parts[1]

	// Create base dialer with timeout
	baseDialer := &net.Dialer{
		Timeout: m.checkTimeout,
	}

	// Dial proxy
	proxyConn, err := baseDialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		peer.Available = false
		peer.CheckError = fmt.Errorf("failed to connect to proxy: %w", err)
		return peer.CheckError
	}
	defer proxyConn.Close()

	// Simple SOCKS5 handshake (no authentication)
	// Send greeting: [version, num_methods, methods...]
	if _, err := proxyConn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
		peer.Available = false
		peer.CheckError = fmt.Errorf("SOCKS5 handshake failed: %w", err)
		return peer.CheckError
	}

	// Read server choice
	buf := make([]byte, 2)
	if _, err := proxyConn.Read(buf); err != nil {
		peer.Available = false
		peer.CheckError = fmt.Errorf("SOCKS5 handshake failed: %w", err)
		return peer.CheckError
	}

	// For TLS connections, wrap the connection
	if useTLS {
		tlsConn := tls.Client(proxyConn, &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         targetAddr,
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			peer.Available = false
			peer.CheckError = fmt.Errorf("TLS handshake failed: %w", err)
			return peer.CheckError
		}
	}

	peer.RTT = time.Since(start)
	peer.Available = true
	peer.CheckedAt = time.Now()
	return nil
}

// CheckPeer checks peer availability based on protocol
func (m *Manager) CheckPeer(ctx context.Context, peer *Peer) error {
	switch peer.Protocol {
	case ProtocolTCP:
		return m.checkTCP(ctx, peer)
	case ProtocolTLS:
		return m.checkTLS(ctx, peer)
	case ProtocolQUIC:
		return m.checkQUIC(ctx, peer)
	case ProtocolWS, ProtocolWSS:
		return m.checkWebSocket(ctx, peer)
	case ProtocolUNIX:
		return m.checkUNIX(ctx, peer)
	case ProtocolSOCKS, ProtocolSOCKSTLS:
		return m.checkSOCKS(ctx, peer)
	default:
		return fmt.Errorf("unsupported protocol: %s", peer.Protocol)
	}
}

// CheckPeers checks multiple peers in parallel
func (m *Manager) CheckPeers(ctx context.Context, peers []*Peer, concurrency int) error {
	if concurrency <= 0 {
		concurrency = 10 // Default
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, peer := range peers {
		wg.Add(1)
		go func(p *Peer) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			m.CheckPeer(ctx, p)
		}(peer)
	}

	wg.Wait()
	return nil
}

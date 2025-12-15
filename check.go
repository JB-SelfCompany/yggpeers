package yggpeers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
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

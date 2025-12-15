package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jbselfcompany/yggpeers"
)

func main() {
	// Create manager with custom settings
	mgr := yggpeers.NewManager(
		yggpeers.WithCacheTTL(15*time.Minute),
		yggpeers.WithTimeout(3*time.Second),
	)

	ctx := context.Background()

	// Get all peers
	fmt.Println("Fetching peers...")
	peers, err := mgr.GetPeers(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded %d peers\n\n", len(peers))

	// Check first 10 peers
	if len(peers) > 10 {
		peers = peers[:10]
	}

	fmt.Println("Checking first 10 peers...")
	err = mgr.CheckPeers(ctx, peers, 5) // 5 parallel checks
	if err != nil {
		log.Fatal(err)
	}

	// Show results
	fmt.Println("\nResults:")
	fmt.Println("========")
	for i, peer := range peers {
		status := "❌ unavailable"
		if peer.Available {
			status = fmt.Sprintf("✅ available (RTT: %v)", peer.RTT)
		} else if peer.CheckError != nil {
			status = fmt.Sprintf("❌ error: %v", peer.CheckError)
		}
		fmt.Printf("%d. %s\n   %s\n   Region: %s\n\n", i+1, peer.Address, status, peer.Region)
	}
}

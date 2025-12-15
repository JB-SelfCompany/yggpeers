package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jbselfcompany/yggpeers"
)

func main() {
	mgr := yggpeers.NewManager()
	ctx := context.Background()

	// Get best TLS peers from Germany with low latency
	fmt.Println("Finding best TLS peers from Germany...")
	filter := &yggpeers.FilterOptions{
		Protocols:     []yggpeers.Protocol{yggpeers.ProtocolTLS},
		Regions:       []string{"germany"},
		MaxRTT:        200 * time.Millisecond,
		OnlyAvailable: true,
	}

	peers, err := mgr.GetAvailablePeers(ctx, filter)
	if err != nil {
		log.Fatal(err)
	}

	// Sort by RTT
	sorted := mgr.FilterPeers(peers, filter, yggpeers.SortByRTT)

	// Show top 5
	fmt.Printf("\nFound %d peers, showing top 5:\n", len(sorted))
	fmt.Println("================================")
	for i, peer := range sorted {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. %s\n", i+1, peer.Address)
		fmt.Printf("   RTT: %v\n", peer.RTT)
		fmt.Printf("   Key: %s\n", peer.Key[:16]+"...")
		fmt.Printf("   Last seen: %s\n\n", time.Unix(peer.LastSeen, 0).Format(time.RFC822))
	}

	// Get regions list
	fmt.Println("\nGetting all available regions...")
	allPeers, err := mgr.GetPeers(ctx)
	if err != nil {
		log.Fatal(err)
	}

	regions := yggpeers.GetRegions(allPeers)
	fmt.Printf("Available regions (%d):\n", len(regions))
	for _, region := range regions {
		fmt.Printf("  - %s\n", region)
	}
}

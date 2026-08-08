package main

import (
	"context"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
)

// Carried over unchanged from bench/. The ER blockhash window was measured at
// ~54s (1198 blocks at ~22 blocks/s), so a long bidding run outlives a single
// blockhash. Refresh in the background and never fetch on the hot path: a fetch
// per bid would add a full round trip and put throughput back at 4 tx/s.
type bhCache struct {
	mu sync.RWMutex
	h  solana.Hash
}

func (b *bhCache) get() solana.Hash {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.h
}

func startBlockhash(ctx context.Context, c *Client) (*bhCache, func()) {
	b := &bhCache{}
	h, _, err := c.LatestBlockhash(ctx, "confirmed")
	check(err)
	b.h = h

	done := make(chan struct{})
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if h, _, err := c.LatestBlockhash(ctx, "confirmed"); err == nil {
					b.mu.Lock()
					b.h = h
					b.mu.Unlock()
				}
			}
		}
	}()
	return b, func() { close(done) }
}

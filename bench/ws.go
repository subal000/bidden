package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

// wsProbe answers whether the ER websocket delivers every intermediate
// account state or coalesces them. The frontend counter reads from this
// stream, so a coalescing socket means the on screen number skips.
func wsProbe(ctx context.Context, er *Client, wallet solana.PrivateKey, wsURL string, n, senders int) {
	pda := counterPDA(wallet.PublicKey())

	cl, err := ws.Connect(ctx, wsURL)
	check(err)
	defer cl.Close()

	sub, err := cl.AccountSubscribe(pda, rpc.CommitmentProcessed)
	check(err)
	defer sub.Unsubscribe()

	var mu sync.Mutex
	seen := map[uint64]bool{}
	var order []uint64
	var firstAt, lastAt time.Time

	recvCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			got, err := sub.Recv(recvCtx)
			if err != nil {
				return
			}
			if got == nil || got.Value == nil || got.Value.Data == nil {
				continue
			}
			raw := got.Value.Data.GetBinary()
			if len(raw) < 16 {
				continue
			}
			v := binary.LittleEndian.Uint64(raw[8:16])
			mu.Lock()
			if firstAt.IsZero() {
				firstAt = time.Now()
			}
			lastAt = time.Now()
			if !seen[v] {
				seen[v] = true
				order = append(order, v)
			}
			mu.Unlock()
		}
	}()

	time.Sleep(1 * time.Second)
	before, err := er.CounterValue(ctx, pda)
	check(err)
	fmt.Printf("subscribed to %s, counter=%d\n", pda, before)
	fmt.Printf("driving %d txs with %d concurrent senders...\n\n", n, senders)

	res := runConc(ctx, nil, er, wallet, n, senders, false)

	// Give the socket time to flush anything still in flight.
	time.Sleep(3 * time.Second)
	cancel()
	<-done

	after, err := er.CounterValue(ctx, pda)
	check(err)

	mu.Lock()
	defer mu.Unlock()
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })

	landed := after - before
	delivered := 0
	for _, v := range order {
		if v > before && v <= after {
			delivered++
		}
	}

	// Count how many consecutive values were skipped.
	gaps, biggest := 0, uint64(0)
	prev := before
	for _, v := range order {
		if v <= before || v > after {
			continue
		}
		if v > prev+1 {
			gaps++
			if v-prev > biggest {
				biggest = v - prev
			}
		}
		prev = v
	}

	fmt.Printf("\n%s\nWEBSOCKET FIDELITY\n%s\n", "========================================", "========================================")
	fmt.Printf("  counter moved      %d -> %d  (%d increments landed)\n", before, after, landed)
	fmt.Printf("  tx landed rate     %.1f tx/s\n", res.landedRate())
	fmt.Printf("  distinct values    %d delivered by websocket\n", delivered)
	if landed > 0 {
		fmt.Printf("  delivery ratio     %.1f%%\n", 100*float64(delivered)/float64(landed))
	}
	fmt.Printf("  skipped runs       %d (largest jump %d)\n", gaps, biggest)
	if !firstAt.IsZero() {
		fmt.Printf("  stream duration    %v\n", lastAt.Sub(firstAt).Round(time.Millisecond))
	}
	if delivered == int(landed) {
		fmt.Printf("  VERDICT            lossless, every intermediate value delivered\n")
	} else {
		fmt.Printf("  VERDICT            COALESCING, %d of %d intermediate values never arrived\n",
			int(landed)-delivered, landed)
	}
}

package main

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
)

// JobState is deliberately the exact shape of the on chain Job payload the
// frontend reads. The mock must expose the same fields so swapping data source
// is a swap, not a rewrite.
type JobState struct {
	BidCount   uint32 `json:"bidCount"`
	BestBidBps uint16 `json:"bestBidBps"`
	BestBidder string `json:"bestBidder"`
	Status     string `json:"status"`
}

// BidEvent is what the UI log renders. Every submitted bid produces one,
// including the ones that lose the race, because the program counts those too.
type BidEvent struct {
	Seq      uint32    `json:"seq"`
	Agent    string    `json:"agent"`
	BidBps   uint16    `json:"bidBps"`
	Improved bool      `json:"improved"`
	At       time.Time `json:"at"`
}

// Market is the seam between bidding logic and where the bids actually go.
// The same Agent code drives both implementations, so the convergence curve can
// be tuned without spending a lamport.
type Market interface {
	// Read returns the current job state. Implementations must not do a network
	// round trip per call: the hot loop calls this before every bid.
	Read() JobState
	// Bid submits and does not await confirmation.
	Bid(agent string, bidBps uint16) error
	Events() []BidEvent
	Close()
}

// ---------------------------------------------------------------- in memory

// MemMarket mirrors the on chain submit_bid exactly: bid_count always
// increments, best_bid_bps and best_bidder only move on a strict improvement,
// and nothing ever returns an error. If this drifts from the Rust, the mocked
// demo stops predicting the real one.
type MemMarket struct {
	mu     sync.Mutex
	state  JobState
	events []BidEvent
	// latency models the measured send round trip so the mock runs at the same
	// rate as the real ER. Without it the mock reports ~79 bids/s against a
	// measured 26-38 and stops predicting anything useful.
	latency time.Duration
}

func NewMemMarket(startBps uint16, latency time.Duration) *MemMarket {
	return &MemMarket{
		latency: latency,
		state: JobState{
			BidCount:   0,
			BestBidBps: startBps,
			BestBidder: "",
			Status:     "Open",
		},
	}
}

func (m *MemMarket) Read() JobState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *MemMarket) Bid(agent string, bidBps uint16) error {
	m.mu.Lock()
	// Mirrors submit_bid line for line.
	m.state.BidCount++
	if m.state.Status == "Open" {
		m.state.Status = "Bidding"
	}
	improved := false
	if m.state.Status == "Bidding" && bidBps < m.state.BestBidBps {
		m.state.BestBidBps = bidBps
		m.state.BestBidder = agent
		improved = true
	}
	m.events = append(m.events, BidEvent{
		Seq: m.state.BidCount, Agent: agent, BidBps: bidBps,
		Improved: improved, At: time.Now(),
	})
	m.mu.Unlock()

	// Modelled outside the lock: this stands in for the network round trip, it
	// is not contention on the job.
	if m.latency > 0 {
		time.Sleep(m.latency)
	}
	return nil
}

func (m *MemMarket) Events() []BidEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]BidEvent, len(m.events))
	copy(out, m.events)
	return out
}

func (m *MemMarket) Close() {}

// ------------------------------------------------------------------- chain

// ChainMarket submits real ER transactions. Read is served from a cached copy
// refreshed in the background, so the hot loop never pays a round trip: a fetch
// per bid would halve throughput.
type ChainMarket struct {
	ctx     context.Context
	er      *Client
	job     solana.PublicKey
	keys    map[string]solana.PrivateKey
	bh      *bhCache
	stopBh  func()
	mu      sync.RWMutex
	cached  JobState
	events   []BidEvent
	sendErrs map[string]int
	evMu     sync.Mutex
	stopped chan struct{}
}

func NewChainMarket(ctx context.Context, er *Client, job solana.PublicKey, keys map[string]solana.PrivateKey, pollEvery time.Duration) (*ChainMarket, error) {
	bh, stop, err := startBlockhash(ctx, er)
	if err != nil {
		return nil, err
	}
	c := &ChainMarket{
		ctx: ctx, er: er, job: job, keys: keys,
		bh: bh, stopBh: stop, stopped: make(chan struct{}),
		sendErrs: map[string]int{},
	}
	if st, err := c.fetch(); err == nil {
		c.cached = st
	} else {
		stop()
		return nil, err
	}
	go func() {
		t := time.NewTicker(pollEvery)
		defer t.Stop()
		for {
			select {
			case <-c.stopped:
				return
			case <-t.C:
				if st, err := c.fetch(); err == nil {
					c.mu.Lock()
					c.cached = st
					c.mu.Unlock()
				}
			}
		}
	}()
	return c, nil
}

func (c *ChainMarket) fetch() (JobState, error) {
	ai, err := c.er.AccountInfo(c.ctx, c.job)
	if err != nil {
		return JobState{}, err
	}
	return parseJobState(ai)
}

func (c *ChainMarket) Read() JobState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cached
}

func (c *ChainMarket) Bid(agent string, bidBps uint16) error {
	kp, okKey := c.keys[agent]
	if !okKey {
		return errNoKey
	}
	tx, err := buildSubmitBid(c.job, kp, bidBps, c.bh.get())
	if err != nil {
		return err
	}
	// Fire and forget. Awaiting confirmation caps throughput at ~4 tx/s.
	if _, err := c.er.SendTx(c.ctx, tx, true); err != nil {
		c.evMu.Lock()
		c.sendErrs[classifyErr(err)]++
		c.evMu.Unlock()
		return err
	}
	c.evMu.Lock()
	c.events = append(c.events, BidEvent{Agent: agent, BidBps: bidBps, At: time.Now()})
	c.evMu.Unlock()
	return nil
}

func (c *ChainMarket) Events() []BidEvent {
	c.evMu.Lock()
	defer c.evMu.Unlock()
	out := make([]BidEvent, len(c.events))
	copy(out, c.events)
	return out
}

func (c *ChainMarket) Close() {
	close(c.stopped)
	c.stopBh()
}

// SendErrors reports why bids failed to send. Never discard these: a silent
// send failure looks exactly like a slow market.
func (c *ChainMarket) SendErrors() map[string]int {
	c.evMu.Lock()
	defer c.evMu.Unlock()
	out := map[string]int{}
	for k, v := range c.sendErrs {
		out[k] = v
	}
	return out
}

func classifyErr(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "already been processed"), strings.Contains(s, "AlreadyProcessed"):
		return "duplicate-transaction"
	case strings.Contains(s, "Blockhash not found"):
		return "blockhash-expired"
	case strings.Contains(s, "429"), strings.Contains(s, "Too Many"):
		return "rate-limited"
	default:
		return s
	}
}

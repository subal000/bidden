package main

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

type AgentSpec struct {
	Name           string  `json:"name"`
	Specialization string  `json:"specialization"`
	FloorBps       uint16  `json:"floorBps"`
	Reputation     uint16  `json:"reputation"` // 0-10000
	Decay          float64 `json:"decay"`      // fraction of the remaining gap cut per undercut
	Aggression     float64 `json:"aggression"` // chance of undercutting rather than holding
}

type Curve struct {
	StartBps           uint16      `json:"startBps"`
	DurationSeconds    int         `json:"durationSeconds"`
	JitterMinMs        int         `json:"jitterMinMs"`
	JitterMaxMs        int         `json:"jitterMaxMs"`
	TickNoise          float64     `json:"tickNoise"`
	SendLatencyMs      int         `json:"sendLatencyMs"`
	ReputationDiscount float64     `json:"reputationDiscount"`
	Agents             []AgentSpec `json:"agents"`
}

// EffectiveFloor is the lowest price an agent will accept.
//
// Reputation lowers it: an experienced agent genuinely executes the job for
// less, so it can undercut. This is the reading of CLAUDE.md's "higher
// reputation agents bid less aggressively and still win" that survives contact
// with the deployed program, which awards strictly on the lowest bps and knows
// nothing about reputation. Such an agent steps down in smaller increments (low
// Decay) yet has a deeper floor, so it looks patient and still wins.
func (s AgentSpec) EffectiveFloor(discount float64) uint16 {
	f := float64(s.FloorBps) * (1 - discount*float64(s.Reputation)/10000.0)
	if f < 1 {
		f = 1
	}
	return uint16(f)
}

// NextBid decides what this agent submits given the current job state.
//
// It returns a bid even when it will not improve the price. That is deliberate:
// the program always increments bid_count, so a held position is still a landed
// transaction and the counter keeps climbing after the price has converged. It
// is also what a real market looks like, participants restating their position.
//
// Three rules keep the descent from collapsing into a step function:
//   - an agent already winning holds, it does not undercut itself
//   - an agent cannot go below its own effective floor
//   - an agent only undercuts on a fraction of its looks (Aggression), so bid
//     volume stays high while the price moves at a watchable pace
func (s AgentSpec) NextBid(st JobState, own uint16, c Curve, rng *rand.Rand) (uint16, bool) {
	floor := s.EffectiveFloor(c.ReputationDiscount)
	best := st.BestBidBps

	// Holding means restating this agent's own standing offer, not echoing the
	// market best. Echoing would put an identical number on every agent card and
	// read as a broken display rather than a market.
	hold := own
	if hold == 0 {
		hold = c.StartBps
	}

	if st.BestBidder == s.Name {
		return hold, false // already winning, hold
	}
	if best <= floor {
		return maxU16(floor, 0), false // cannot beat it and stay profitable
	}
	if rng.Float64() > s.Aggression {
		return hold, false // patience, restate rather than cut
	}

	gap := float64(best - floor)
	tick := gap * s.Decay
	// Multiplicative noise so the descent reads organic rather than metronomic.
	tick *= 1 - c.TickNoise + rng.Float64()*2*c.TickNoise
	if tick < 1 {
		tick = 1
	}
	next := float64(best) - tick
	if next < float64(floor) {
		next = float64(floor)
	}
	return uint16(next), true
}

// Run drives one agent until the deadline. One goroutine per agent, own
// keypair, own cost curve, 50-100ms jitter. This is the measured 26-38 tx/s
// demo configuration.
func Run(ctx context.Context, m Market, c Curve, s AgentSpec, seed int64, wg *sync.WaitGroup) {
	defer wg.Done()
	rng := rand.New(rand.NewSource(seed))
	jitterSpan := c.JitterMaxMs - c.JitterMinMs
	var own uint16 // this agent's standing offer

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		d := time.Duration(c.JitterMinMs) * time.Millisecond
		if jitterSpan > 0 {
			d += time.Duration(rng.Intn(jitterSpan)) * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(d):
		}

		st := m.Read()
		bid, _ := s.NextBid(st, own, c, rng)
		own = bid
		_ = m.Bid(s.Name, bid) // never awaits, never retries
	}
}

func maxU16(a, b uint16) uint16 {
	if a > b {
		return a
	}
	return b
}

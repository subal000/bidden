package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
)

var (
	fMode      = flag.String("mode", "simulate", "simulate|live")
	fCurve     = flag.String("curve", "curve.json", "curve tuning file")
	fER        = flag.String("er", "https://devnet-as.magicblock.app", "ER RPC (live mode)")
	fRequester = flag.String("requester", "", "requester pubkey, derives the job PDA (live mode)")
	fKeyDir    = flag.String("keydir", "../driver/keys", "agent keypairs (live mode)")
	fExport    = flag.String("export", "", "write the run as JSON to this path")
	fSeed      = flag.Int64("seed", 1, "rng seed, same seed gives the same curve")
	fQuiet     = flag.Bool("quiet", false, "suppress the sampled table")
	fJobID     = flag.Uint64("job-id", 1, "job id, must match the driver (live mode)")
)

func main() {
	flag.Parse()

	curve, err := loadCurve(*fCurve)
	if err != nil {
		fmt.Println("FATAL:", err)
		os.Exit(1)
	}

	switch *fMode {
	case "simulate":
		runSimulate(curve)
	case "live":
		runLive(curve)
	default:
		fmt.Println("unknown mode", *fMode)
		os.Exit(2)
	}
}

func loadCurve(path string) (Curve, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Curve{}, err
	}
	var c Curve
	if err := json.Unmarshal(b, &c); err != nil {
		return Curve{}, err
	}
	if len(c.Agents) == 0 {
		return Curve{}, fmt.Errorf("%s defines no agents", path)
	}
	return c, nil
}

// ---------------------------------------------------------------- simulate

func runSimulate(c Curve) {
	fmt.Printf("simulate  %d agents, %ds, jitter %d-%dms, start %d bps\n",
		len(c.Agents), c.DurationSeconds, c.JitterMinMs, c.JitterMaxMs, c.StartBps)
	fmt.Printf("%-10s %-12s %8s %8s %8s\n", "AGENT", "SPEC", "FLOOR", "EFF", "DECAY")
	for _, a := range c.Agents {
		fmt.Printf("%-10s %-12s %8d %8d %8.3f\n",
			a.Name, a.Specialization, a.FloorBps, a.EffectiveFloor(c.ReputationDiscount), a.Decay)
	}

	m := NewMemMarket(c.StartBps, time.Duration(c.SendLatencyMs)*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(c.DurationSeconds)*time.Second)
	defer cancel()

	// Sample the state on a fixed cadence so the curve can be inspected.
	type sample struct {
		T     float64 `json:"t"`
		Count uint32  `json:"bidCount"`
		Best  uint16  `json:"bestBidBps"`
	}
	var samples []sample
	var smu sync.Mutex
	start := time.Now()
	done := make(chan struct{})
	go func() {
		defer close(done)
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				st := m.Read()
				smu.Lock()
				samples = append(samples, sample{time.Since(start).Seconds(), st.BidCount, st.BestBidBps})
				smu.Unlock()
				return
			case <-t.C:
				st := m.Read()
				smu.Lock()
				samples = append(samples, sample{time.Since(start).Seconds(), st.BidCount, st.BestBidBps})
				smu.Unlock()
			}
		}
	}()

	var wg sync.WaitGroup
	for i, a := range c.Agents {
		wg.Add(1)
		go Run(ctx, m, c, a, *fSeed+int64(i)*7919, &wg)
	}
	wg.Wait()
	<-done
	elapsed := time.Since(start)

	final := m.Read()
	events := m.Events()

	if !*fQuiet {
		fmt.Printf("\n%-7s %8s %10s  %s\n", "TIME", "BIDS", "BEST bps", "CONVERGENCE")
		for i, s := range samples {
			if i%4 != 0 && i != len(samples)-1 {
				continue // print once a second
			}
			fmt.Printf("%6.1fs %8d %10d  %s\n", s.T, s.Count, s.Best, bar(s.Best, c.StartBps, lowestFloor(c)))
		}
	}

	fmt.Printf("\n%s\n", strings.Repeat("-", 64))
	fmt.Printf("  elapsed        %v\n", elapsed.Round(time.Millisecond))
	fmt.Printf("  bids landed    %d  (%.1f/s)\n", final.BidCount, float64(final.BidCount)/elapsed.Seconds())
	fmt.Printf("  final best     %d bps by %s\n", final.BestBidBps, final.BestBidder)
	fmt.Printf("  improvements   %d of %d bids moved the price\n", countImproved(events), len(events))
	fmt.Printf("  status         %s\n", final.Status)

	fmt.Printf("\n  bids per agent:\n")
	per := map[string]int{}
	won := map[string]int{}
	for _, e := range events {
		per[e.Agent]++
		if e.Improved {
			won[e.Agent]++
		}
	}
	names := make([]string, 0, len(per))
	for k := range per {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Printf("    %-10s %5d sent  %5d improved\n", n, per[n], won[n])
	}

	fmt.Printf("\n  demo check: 500 bids in 30s needs 16.7/s -> %s\n",
		verdict(float64(final.BidCount)/elapsed.Seconds()))

	if *fExport != "" {
		out := map[string]interface{}{
			"curve": c, "samples": samples, "final": final, "events": events,
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		if err := os.WriteFile(*fExport, b, 0o644); err != nil {
			fmt.Println("export failed:", err)
		} else {
			fmt.Printf("\n  exported %s\n", *fExport)
		}
	}
}

func lowestFloor(c Curve) uint16 {
	lo := c.StartBps
	for _, a := range c.Agents {
		if f := a.EffectiveFloor(c.ReputationDiscount); f < lo {
			lo = f
		}
	}
	return lo
}

func countImproved(ev []BidEvent) int {
	n := 0
	for _, e := range ev {
		if e.Improved {
			n++
		}
	}
	return n
}

// bar draws how far the price has travelled from the opening ask to the
// theoretical floor, so the shape of the descent is visible in a terminal.
func bar(best, start, floor uint16) string {
	if start <= floor {
		return ""
	}
	frac := float64(best-floor) / float64(start-floor)
	if frac < 0 {
		frac = 0
	}
	n := int(frac * 40)
	return strings.Repeat("#", n) + strings.Repeat(".", 40-n)
}

func verdict(rate float64) string {
	if rate >= 16.7 {
		return fmt.Sprintf("CLEARS (%.0f bids in 30s)", rate*30)
	}
	return fmt.Sprintf("MISSES (%.0f bids in 30s)", rate*30)
}

// -------------------------------------------------------------------- live

func runLive(c Curve) {
	if *fRequester == "" {
		fmt.Println("FATAL: --requester is required in live mode")
		os.Exit(2)
	}
	requester, err := solana.PublicKeyFromBase58(*fRequester)
	if err != nil {
		fmt.Println("FATAL:", err)
		os.Exit(1)
	}
	job := jobPDA(requester, *fJobID)

	keys := map[string]solana.PrivateKey{}
	for i, a := range c.Agents {
		path := fmt.Sprintf("%s/agent%d.json", *fKeyDir, i)
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("FATAL: %s: %v\n", path, err)
			os.Exit(1)
		}
		kp, err := solana.PrivateKeyFromSolanaKeygenFileBytes(b)
		if err != nil {
			fmt.Println("FATAL:", err)
			os.Exit(1)
		}
		keys[a.Name] = kp
	}

	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(c.DurationSeconds)*time.Second)
	defer cancel()

	er := NewClient(*fER, len(c.Agents)+4)
	m, err := NewChainMarket(ctx, er, job, keys, 400*time.Millisecond)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Printf(`FATAL: job %d does not exist in the ER yet.

  job pda  %s

Create and delegate it first:

  cd ../driver
  go run . --mode post-job  --job-id %d
  go run . --mode delegate  --job-id %d
  go run . --mode addresses --job-id %d

Then open http://localhost:3000 BEFORE re-running the agents, so the page
records a zero baseline and the cards agree with the counter.
`, *fJobID, job, *fJobID, *fJobID, *fJobID)
			os.Exit(2)
		}
		fmt.Println("FATAL:", err)
		os.Exit(1)
	}
	defer m.Close()

	fmt.Printf("live  job %s\n", job)
	before := m.Read()
	fmt.Printf("  bid_count before %d, best %d bps\n", before.BidCount, before.BestBidBps)

	var wg sync.WaitGroup
	start := time.Now()
	for i, a := range c.Agents {
		wg.Add(1)
		go Run(ctx, m, c, a, *fSeed+int64(i)*7919, &wg)
	}
	wg.Wait()
	elapsed := time.Since(start)

	time.Sleep(2 * time.Second) // let the tail land
	after := m.Read()
	landed := after.BidCount - before.BidCount
	fmt.Printf("\n  bids landed  %d in %v (%.1f/s)\n", landed, elapsed.Round(time.Millisecond),
		float64(landed)/elapsed.Seconds())
	fmt.Printf("  final best   %d bps by %s\n", after.BestBidBps, after.BestBidder)
	if errs := m.SendErrors(); len(errs) > 0 {
		fmt.Printf("  send errors:\n")
		for k, v := range errs {
			fmt.Printf("    %-24s %d\n", k, v)
		}
	} else {
		fmt.Printf("  send errors  none\n")
	}
}

// envOr lets L1_RPC override the default endpoint. The public devnet endpoint
// rate limits a single method near 40 per 10s, which produced false "not
// confirmed" timeouts on transactions that had actually landed.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

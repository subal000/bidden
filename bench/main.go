package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
)

var (
	fMode      = flag.String("mode", "", "prepare|blockhash|seq-confirm|seq-fire|conc|read|undelegate|suite")
	fN         = flag.Int("n", 200, "transactions per run")
	fSenders   = flag.Int("senders", 6, "concurrent senders (conc mode)")
	fMulti     = flag.Bool("multi", false, "conc mode: one keypair+PDA per sender instead of a shared one")
	fJitterMin = flag.Duration("jitter-min", 0, "min sleep between sends per sender")
	fJitterMax = flag.Duration("jitter-max", 0, "max sleep between sends per sender")
	fL1        = flag.String("l1", envOr("L1_RPC", "https://api.devnet.solana.com"), "L1 RPC (or set L1_RPC)")
	fER        = flag.String("er", "https://devnet-as.magicblock.app", "ER RPC")
	fWS        = flag.String("ws", "wss://devnet-as.magicblock.app", "ER websocket")
	fValidator = flag.String("validator", "MAS1Dt9qreoRMQ14YQuhg8UTZMMzDdKhmkZMECCzk57", "ER validator identity")
	fWallet    = flag.String("wallet", os.Getenv("HOME")+"/.config/solana/id.json", "payer keypair")
	fKeyDir    = flag.String("keydir", "keys", "directory for generated sender keypairs")
	fPoll      = flag.Duration("poll", 20*time.Millisecond, "confirmation poll interval (raise on rate limited public RPC)")
)

func main() {
	flag.Parse()
	ctx := context.Background()
	if *fMode == "" {
		flag.Usage()
		os.Exit(2)
	}

	wallet, err := solana.PrivateKeyFromSolanaKeygenFile(*fWallet)
	check(err)

	conns := *fSenders + 4
	l1 := NewClient(*fL1, conns)
	er := NewClient(*fER, conns)
	check(er.Warm(ctx))

	fmt.Printf("wallet %s\nER     %s\nL1     %s\n\n", wallet.PublicKey(), *fER, *fL1)

	switch *fMode {
	case "read", "l1-evidence":
		// read-only, always safe
	default:
		requireProgram(ctx, l1)
	}

	switch *fMode {
	case "prepare":
		prepare(ctx, l1, er, wallet)
	case "read":
		readState(ctx, l1, er, wallet)
	case "sweep":
		sweep(ctx, l1, wallet)
	case "undelegate":
		undelegate(ctx, l1, er, wallet)
	case "blockhash":
		blockhashProbe(ctx, er, wallet)
	case "ws":
		wsProbe(ctx, er, wallet, *fWS, *fN, *fSenders)
	case "seq-confirm":
		report(runSeq(ctx, er, wallet, *fN, true))
	case "seq-fire":
		report(runSeq(ctx, er, wallet, *fN, false))
	case "conc":
		report(runConc(ctx, l1, er, wallet, *fN, *fSenders, *fMulti))
	case "suite":
		suite(ctx, l1, er, wallet)
	default:
		fmt.Println("unknown mode", *fMode)
		os.Exit(2)
	}
}

// requireProgram refuses to run any transaction-sending mode if the M0 program
// is gone. It is a live check rather than a hardcoded message, so this harness
// starts working again the moment a program is deployed to that address.
func requireProgram(ctx context.Context, l1 *Client) {
	// A closed loader-v3 program leaves the program account in place and
	// executable; it is the programdata account that disappears. Checking the
	// program account alone reports a closed program as healthy.
	ai, err := l1.AccountInfo(ctx, programID)
	if err != nil || ai == nil || len(ai.Data) < 36 {
		// no program account at all
	} else {
		var pd solana.PublicKey
		copy(pd[:], ai.Data[4:36])
		if pdAcct, err := l1.AccountInfo(ctx, pd); err == nil && pdAcct != nil && len(pdAcct.Data) > 0 {
			return // programdata intact, the program is live
		}
	}
	fmt.Printf(`This mode cannot run. The M0 counter program is closed.

  program %s
  status  CLOSED under loader-v3, permanently unusable

It was closed to reclaim 2.0586 SOL of rent to fund the swarm deploy. The
program ID can never be redeployed, so every mode that sends a transaction to
it will fail.

The measurements this harness produced are recorded and still verifiable:

  bench/RESULTS.md          every measured number, and how it was obtained
  bench/logs/               captured console output from the runs
  bench/logs/l1-evidence.json   L1 signatures, fetched live from devnet

The delegate / commit / undelegate lifecycle is independently checkable on the
Solana explorer. See the signature table in RESULTS.md.

Read-only modes still work:  --mode read
`, programID)
	os.Exit(3)
}

func check(err error) {
	if err != nil {
		fmt.Println("FATAL:", err)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------- setup

func isDelegated(ai *AccountInfo) bool {
	return ai != nil && ai.Owner.Equals(delegationProgram)
}

func ensureDelegated(ctx context.Context, l1, er *Client, kp solana.PrivateKey) error {
	pda := counterPDA(kp.PublicKey())
	ai, err := l1.AccountInfo(ctx, pda)
	if err != nil {
		return err
	}
	if !isDelegated(ai) {
		if ai == nil || ai.Owner.Equals(programID) {
			bh, _, err := l1.LatestBlockhash(ctx, "finalized")
			if err != nil {
				return err
			}
			tx, err := buildSigned([]solana.Instruction{ixInitialize(kp.PublicKey())}, bh, kp)
			if err != nil {
				return err
			}
			sig, err := l1.SendTx(ctx, tx, false)
			if err != nil {
				return fmt.Errorf("initialize: %w", err)
			}
			fmt.Printf("  initialize %s -> %s\n", pda, sig[:16])
			if err := awaitL1(ctx, l1, sig); err != nil {
				return err
			}
		}
		bh, _, err := l1.LatestBlockhash(ctx, "finalized")
		if err != nil {
			return err
		}
		tx, err := buildSigned([]solana.Instruction{
			ixDelegate(kp.PublicKey(), solana.MustPublicKeyFromBase58(*fValidator)),
		}, bh, kp)
		if err != nil {
			return err
		}
		sig, err := l1.SendTx(ctx, tx, false)
		if err != nil {
			return fmt.Errorf("delegate: %w", err)
		}
		fmt.Printf("  delegate   %s -> %s\n", pda, sig[:16])
		if err := awaitL1(ctx, l1, sig); err != nil {
			return err
		}
	}
	// Wait for the ER validator to actually pick the account up.
	deadline := time.Now().Add(45 * time.Second)
	for {
		ai, _ := er.AccountInfo(ctx, pda)
		if ai != nil && ai.Owner.Equals(programID) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for ER pickup of %s", pda)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func awaitL1(ctx context.Context, l1 *Client, sig string) error {
	deadline := time.Now().Add(60 * time.Second)
	for {
		ok, status, err := l1.SignatureLanded(ctx, sig)
		if err == nil && ok {
			return nil
		}
		if err == nil && status != "" && status != "null" {
			return fmt.Errorf("tx %s failed: %s", sig[:16], status)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("tx %s not confirmed in 60s", sig[:16])
		}
		time.Sleep(1 * time.Second)
	}
}

func prepare(ctx context.Context, l1, er *Client, wallet solana.PrivateKey) {
	fmt.Println("preparing shared counter")
	check(ensureDelegated(ctx, l1, er, wallet))
	v, err := er.CounterValue(ctx, counterPDA(wallet.PublicKey()))
	check(err)
	fmt.Printf("  shared PDA %s delegated, count=%d\n", counterPDA(wallet.PublicKey()), v)

	if *fMulti {
		keys := loadOrCreateKeys(*fSenders)
		fmt.Printf("\npreparing %d per-sender counters\n", len(keys))
		check(fundKeys(ctx, l1, wallet, keys, 12_000_000))
		for i, k := range keys {
			fmt.Printf(" sender %d %s\n", i, k.PublicKey())
			check(ensureDelegated(ctx, l1, er, k))
		}
	}
	fmt.Println("\nprepare done")
}

func loadOrCreateKeys(n int) []solana.PrivateKey {
	check(os.MkdirAll(*fKeyDir, 0o700))
	out := make([]solana.PrivateKey, 0, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(*fKeyDir, fmt.Sprintf("sender%d.json", i))
		if b, err := os.ReadFile(p); err == nil {
			kp, err := solana.PrivateKeyFromSolanaKeygenFileBytes(b)
			check(err)
			out = append(out, kp)
			continue
		}
		kp, err := solana.NewRandomPrivateKey()
		check(err)
		// Must be a JSON array of byte values, not base64, or the solana CLI
		// cannot read the file.
		nums := make([]int, len(kp))
		for i, c := range []byte(kp) {
			nums[i] = int(c)
		}
		b, _ := json.Marshal(nums)
		check(os.WriteFile(p, b, 0o600))
		out = append(out, kp)
	}
	return out
}

func fundKeys(ctx context.Context, l1 *Client, from solana.PrivateKey, keys []solana.PrivateKey, lamports uint64) error {
	var ixs []solana.Instruction
	for _, k := range keys {
		ai, err := l1.AccountInfo(ctx, k.PublicKey())
		if err != nil {
			return err
		}
		if ai != nil {
			continue // already funded
		}
		data := make([]byte, 12)
		binary.LittleEndian.PutUint32(data[0:], 2)
		binary.LittleEndian.PutUint64(data[4:], lamports)
		ixs = append(ixs, solana.NewInstruction(systemProgram, solana.AccountMetaSlice{
			solana.NewAccountMeta(from.PublicKey(), true, true),
			solana.NewAccountMeta(k.PublicKey(), true, false),
		}, data))
	}
	if len(ixs) == 0 {
		return nil
	}
	bh, _, err := l1.LatestBlockhash(ctx, "finalized")
	if err != nil {
		return err
	}
	tx, err := buildSigned(ixs, bh, from)
	if err != nil {
		return err
	}
	sig, err := l1.SendTx(ctx, tx, false)
	if err != nil {
		return fmt.Errorf("funding: %w", err)
	}
	fmt.Printf("  funded %d senders -> %s\n", len(ixs), sig[:16])
	return awaitL1(ctx, l1, sig)
}

func readState(ctx context.Context, l1, er *Client, wallet solana.PrivateKey) {
	pda := counterPDA(wallet.PublicKey())
	ail, _ := l1.AccountInfo(ctx, pda)
	aie, _ := er.AccountInfo(ctx, pda)
	fmt.Printf("PDA %s\n", pda)
	if ail != nil {
		fmt.Printf("  L1 owner=%s count=%d\n", ail.Owner, binary.LittleEndian.Uint64(ail.Data[8:16]))
	} else {
		fmt.Println("  L1 <missing>")
	}
	if aie != nil {
		fmt.Printf("  ER owner=%s count=%d\n", aie.Owner, binary.LittleEndian.Uint64(aie.Data[8:16]))
	} else {
		fmt.Println("  ER <missing>")
	}
}

func undelegate(ctx context.Context, l1, er *Client, wallet solana.PrivateKey) {
	bh, _, err := er.LatestBlockhash(ctx, "confirmed")
	check(err)
	tx, err := buildSigned([]solana.Instruction{ixUndelegate(wallet.PublicKey())}, bh, wallet)
	check(err)
	sig, err := er.SendTx(ctx, tx, true)
	check(err)
	fmt.Println("undelegate scheduled:", sig)
}

// ---------------------------------------------------- blockhash expiry probe

func blockhashProbe(ctx context.Context, er *Client, wallet solana.PrivateKey) {
	pda := counterPDA(wallet.PublicKey())
	bh, lastValid, err := er.LatestBlockhash(ctx, "confirmed")
	check(err)
	h0, _ := er.BlockHeight(ctx)
	s0, _ := er.Slot(ctx)
	fmt.Printf("cached blockhash      %s\n", bh)
	fmt.Printf("blockHeight at fetch  %d\n", h0)
	fmt.Printf("slot at fetch         %d\n", s0)
	fmt.Printf("lastValidBlockHeight  %d  (window = %d blocks)\n\n", lastValid, lastValid-h0)

	// Measure how fast blockHeight actually advances, which tells us how long
	// a 150 block window really is.
	time.Sleep(2 * time.Second)
	h1, _ := er.BlockHeight(ctx)
	s1, _ := er.Slot(ctx)
	fmt.Printf("after 2s: blockHeight %d (+%d, %.0f blocks/s), slot %d (+%d, %.0f slots/s)\n\n",
		h1, h1-h0, float64(h1-h0)/2.0, s1, s1-s0, float64(s1-s0)/2.0)

	offsets := []time.Duration{
		1 * time.Second, 5 * time.Second, 15 * time.Second,
		30 * time.Second, 45 * time.Second, 60 * time.Second,
	}
	start := time.Now()
	fmt.Printf("%-8s %-8s %-9s %s\n", "AGE", "RESULT", "COUNTER", "DETAIL")
	uniq := uint32(300000)
	for _, off := range offsets {
		if d := time.Until(start.Add(off)); d > 0 {
			time.Sleep(d)
		}
		before, _ := er.CounterValue(ctx, pda)
		uniq++
		tx, err := incrementTx(wallet, bh, uniq)
		check(err)
		// preflight ON so the RPC tells us why it rejects the transaction
		sig, sendErr := er.SendTx(ctx, tx, false)

		age := time.Since(start).Round(100 * time.Millisecond)
		if sendErr != nil {
			fmt.Printf("%-8s %-8s %-9s %s\n", age, "REJECT", "-", truncate(sendErr.Error(), 110))
			continue
		}
		time.Sleep(1500 * time.Millisecond)
		after, _ := er.CounterValue(ctx, pda)
		res, detail := "DROP", "accepted by RPC but counter did not move"
		if after > before {
			res, detail = "LAND", "sig "+sig[:20]
		}
		fmt.Printf("%-8s %-8s %-9s %s\n", age, res, fmt.Sprintf("%d->%d", before, after), detail)
	}
}

// ------------------------------------------------------------- benchmarks

// The ER blockhash window measured at ~54s (1198 blocks at ~22 blocks/s).
// Long runs outlive a single blockhash, so refresh in the background rather
// than paying a fetch round trip on the hot path. This is also what the
// agents will do.
type bhCache struct {
	mu sync.RWMutex
	h  solana.Hash
}

func (b *bhCache) get() solana.Hash {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.h
}

func startBlockhash(ctx context.Context, er *Client) (*bhCache, func()) {
	b := &bhCache{}
	h, _, err := er.LatestBlockhash(ctx, "confirmed")
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
				if h, _, err := er.LatestBlockhash(ctx, "confirmed"); err == nil {
					b.mu.Lock()
					b.h = h
					b.mu.Unlock()
				}
			}
		}
	}()
	return b, func() { close(done) }
}

type Result struct {
	Name       string
	Attempted  int
	SendOK     int
	Landed     int
	SendWall   time.Duration
	Errors     map[string]int
	SettleWait time.Duration
	Latencies  []time.Duration
}

// latencyStats returns min, median and max send-to-confirmed latency.
func (r Result) latencyStats() (time.Duration, time.Duration, time.Duration) {
	if len(r.Latencies) == 0 {
		return 0, 0, 0
	}
	s := append([]time.Duration(nil), r.Latencies...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[0], s[len(s)/2], s[len(s)-1]
}

func (r Result) sendRate() float64 {
	return float64(r.SendOK) / r.SendWall.Seconds()
}
func (r Result) landedRate() float64 {
	return float64(r.Landed) / r.SendWall.Seconds()
}

func classify(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "Blockhash not found"), strings.Contains(s, "BlockhashNotFound"):
		return "blockhash-not-found"
	case strings.Contains(s, "AlreadyProcessed"), strings.Contains(s, "already been processed"):
		return "already-processed"
	case strings.Contains(s, "AccountInUse"):
		return "account-in-use"
	case strings.Contains(s, "context deadline"), strings.Contains(s, "Timeout"), strings.Contains(s, "timeout"):
		return "timeout"
	case strings.Contains(s, "connection reset"), strings.Contains(s, "EOF"):
		return "conn-reset"
	case strings.Contains(s, "429"), strings.Contains(s, "Too Many"):
		return "rate-limited"
	default:
		return truncate(s, 70)
	}
}

// settle polls until the observed total stops moving, so "landed" is measured
// from chain state rather than from what we think we sent.
// A failed read must never be reported as zero. Silently treating a rate
// limited getAccountInfo as "counter is 0" produces negative landed counts
// and destroys the measurement.
func settle(ctx context.Context, er *Client, pdas []solana.PublicKey) (uint64, time.Duration) {
	t0 := time.Now()
	var prev uint64
	havePrev := false
	stable := 0
	for time.Since(t0) < 45*time.Second {
		time.Sleep(500 * time.Millisecond)
		total, ok := tryTotal(ctx, er, pdas)
		if !ok {
			continue // read failed, do not pollute the sample
		}
		if havePrev && total == prev {
			stable++
			if stable >= 4 {
				return total, time.Since(t0)
			}
		} else {
			stable = 0
			prev = total
			havePrev = true
		}
	}
	if !havePrev {
		check(fmt.Errorf("settle: every counter read failed, cannot measure landed count"))
	}
	return prev, time.Since(t0)
}

// tryTotal reads every PDA, retrying briefly, and reports whether the whole
// set was read cleanly.
func tryTotal(ctx context.Context, er *Client, pdas []solana.PublicKey) (uint64, bool) {
	total := uint64(0)
	for _, p := range pdas {
		var v uint64
		var err error
		for attempt := 0; attempt < 5; attempt++ {
			v, err = er.CounterValue(ctx, p)
			if err == nil {
				break
			}
			time.Sleep(time.Duration(200*(attempt+1)) * time.Millisecond)
		}
		if err != nil {
			return 0, false
		}
		total += v
	}
	return total, true
}

func sumCounters(ctx context.Context, er *Client, pdas []solana.PublicKey) uint64 {
	total, ok := tryTotal(ctx, er, pdas)
	if !ok {
		check(fmt.Errorf("could not read baseline counter value"))
	}
	return total
}

func runSeq(ctx context.Context, er *Client, wallet solana.PrivateKey, n int, awaitConfirm bool) Result {
	name := "seq-fire"
	if awaitConfirm {
		name = "seq-confirm"
	}
	pdas := []solana.PublicKey{counterPDA(wallet.PublicKey())}
	before := sumCounters(ctx, er, pdas)

	bh, stop := startBlockhash(ctx, er)
	defer stop()

	res := Result{Name: name, Attempted: n, Errors: map[string]int{}}
	uniq := uint32(time.Now().UnixNano() & 0x7fffff)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	t0 := time.Now()
	for i := 0; i < n; i++ {
		// Optional pacing, used to stay under a public RPC rate limit so the
		// measurement reflects the chain rather than the limiter.
		if *fJitterMax > 0 {
			d := *fJitterMin
			if span := *fJitterMax - *fJitterMin; span > 0 {
				d += time.Duration(rng.Int63n(int64(span)))
			}
			time.Sleep(d)
		}
		uniq++
		tx, err := incrementTx(wallet, bh.get(), uniq)
		check(err)
		sendAt := time.Now()
		sig, err := er.SendTx(ctx, tx, true)
		if err != nil {
			res.Errors[classify(err)]++
			continue
		}
		res.SendOK++
		if awaitConfirm {
			for {
				ok, _, e := er.SignatureLanded(ctx, sig)
				if ok || e != nil {
					break
				}
				time.Sleep(*fPoll)
			}
			res.Latencies = append(res.Latencies, time.Since(sendAt))
		}
	}
	res.SendWall = time.Since(t0)
	after, wait := settle(ctx, er, pdas)
	res.Landed = int(after - before)
	res.SettleWait = wait
	return res
}

func runConc(ctx context.Context, l1, er *Client, wallet solana.PrivateKey, n, senders int, multi bool) Result {
	name := fmt.Sprintf("conc%d-shared", senders)
	signers := make([]solana.PrivateKey, senders)
	for i := range signers {
		signers[i] = wallet
	}
	if multi {
		name = fmt.Sprintf("conc%d-multi", senders)
		signers = loadOrCreateKeys(senders)
	}
	if *fJitterMax > 0 {
		name += fmt.Sprintf("-jitter%v-%v", *fJitterMin, *fJitterMax)
	}

	seen := map[string]bool{}
	var pdas []solana.PublicKey
	for _, s := range signers {
		p := counterPDA(s.PublicKey())
		if !seen[p.String()] {
			seen[p.String()] = true
			pdas = append(pdas, p)
		}
	}
	before := sumCounters(ctx, er, pdas)

	bh, stop := startBlockhash(ctx, er)
	defer stop()

	res := Result{Name: name, Attempted: n, Errors: map[string]int{}}
	var mu sync.Mutex
	var sendOK int64
	var uniq int64 = time.Now().UnixNano() & 0x7fffff

	per := n / senders
	extra := n % senders

	var wg sync.WaitGroup
	t0 := time.Now()
	for s := 0; s < senders; s++ {
		count := per
		if s < extra {
			count++
		}
		wg.Add(1)
		go func(idx, count int, kp solana.PrivateKey) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(idx)*7919 + time.Now().UnixNano()))
			for i := 0; i < count; i++ {
				if *fJitterMax > 0 {
					span := *fJitterMax - *fJitterMin
					d := *fJitterMin
					if span > 0 {
						d += time.Duration(rng.Int63n(int64(span)))
					}
					time.Sleep(d)
				}
				u := uint32(atomic.AddInt64(&uniq, 1))
				tx, err := incrementTx(kp, bh.get(), u)
				if err != nil {
					mu.Lock()
					res.Errors["build:"+truncate(err.Error(), 40)]++
					mu.Unlock()
					continue
				}
				if _, err := er.SendTx(ctx, tx, true); err != nil {
					mu.Lock()
					res.Errors[classify(err)]++
					mu.Unlock()
					continue
				}
				atomic.AddInt64(&sendOK, 1)
			}
		}(s, count, signers[s])
	}
	wg.Wait()
	res.SendWall = time.Since(t0)
	res.SendOK = int(sendOK)

	after, wait := settle(ctx, er, pdas)
	res.Landed = int(after - before)
	res.SettleWait = wait
	return res
}

func report(r Result) {
	fmt.Printf("\n%s\n", strings.Repeat("-", 72))
	fmt.Printf("%-22s attempted=%d sendOK=%d landed=%d\n", r.Name, r.Attempted, r.SendOK, r.Landed)
	fmt.Printf("  send wall     %v\n", r.SendWall.Round(time.Millisecond))
	fmt.Printf("  send rate     %.1f tx/s\n", r.sendRate())
	fmt.Printf("  LANDED rate   %.1f tx/s\n", r.landedRate())
	drop := r.SendOK - r.Landed
	fmt.Printf("  dropped       %d (accepted by RPC, never hit state)\n", drop)
	if len(r.Latencies) > 0 {
		lo, mid, hi := r.latencyStats()
		fmt.Printf("  confirm latency  min %v  median %v  max %v  (n=%d)\n",
			lo.Round(time.Millisecond), mid.Round(time.Millisecond),
			hi.Round(time.Millisecond), len(r.Latencies))
	}
	if len(r.Errors) > 0 {
		keys := make([]string, 0, len(r.Errors))
		for k := range r.Errors {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		fmt.Printf("  send errors:\n")
		for _, k := range keys {
			fmt.Printf("    %-24s %d\n", k, r.Errors[k])
		}
	}
	fmt.Printf("%s\n", strings.Repeat("-", 72))
}

func suite(ctx context.Context, l1, er *Client, wallet solana.PrivateKey) {
	var results []Result
	add := func(label string, r Result) {
		r.Name = label
		report(r)
		results = append(results, r)
		time.Sleep(2 * time.Second)
	}

	withJitter := func(min, max time.Duration, f func() Result) Result {
		om, ox := *fJitterMin, *fJitterMax
		*fJitterMin, *fJitterMax = min, max
		defer func() { *fJitterMin, *fJitterMax = om, ox }()
		return f()
	}

	add("1 seq-confirm", runSeq(ctx, er, wallet, *fN, true))
	add("2 seq-fire", runSeq(ctx, er, wallet, *fN, false))
	add("3 conc6-fire (shared PDA)",
		withJitter(0, 0, func() Result { return runConc(ctx, l1, er, wallet, *fN, 6, false) }))
	add("4 conc6-jitter (shared PDA)",
		withJitter(50*time.Millisecond, 100*time.Millisecond,
			func() Result { return runConc(ctx, l1, er, wallet, *fN, 6, false) }))

	if *fMulti {
		add("3b conc6-fire (6 keypairs)",
			withJitter(0, 0, func() Result { return runConc(ctx, l1, er, wallet, *fN, 6, true) }))
		add("4b conc6-jitter (6 keypairs)",
			withJitter(50*time.Millisecond, 100*time.Millisecond,
				func() Result { return runConc(ctx, l1, er, wallet, *fN, 6, true) }))
	}

	fmt.Printf("\n%s\nRESULTS  (n=%d per run, landed verified by reading the counter)\n%s\n",
		strings.Repeat("=", 84), *fN, strings.Repeat("=", 84))
	fmt.Printf("%-30s %10s %7s %7s %8s %12s\n", "CONFIG", "SEND WALL", "SENT", "LANDED", "DROPPED", "LANDED tx/s")
	for _, r := range results {
		fmt.Printf("%-30s %10v %7d %7d %8d %12.1f\n",
			r.Name, r.SendWall.Round(time.Millisecond), r.SendOK, r.Landed,
			r.SendOK-r.Landed, r.landedRate())
	}
	fmt.Printf("\ndemo requirement: 500 tx in 30s = 16.7 tx/s\n")
	for _, r := range results {
		verdict := "MISS"
		if r.landedRate() >= 16.7 {
			verdict = fmt.Sprintf("CLEARS (%.0f tx in 30s)", r.landedRate()*30)
		}
		fmt.Printf("  %-30s %s\n", r.Name, verdict)
	}
}

// sweep returns leftover lamports from the generated sender keypairs to the
// main wallet. Each transfer leaves nothing behind, so the accounts close.
func sweep(ctx context.Context, l1 *Client, wallet solana.PrivateKey) {
	keys := loadOrCreateKeys(*fSenders)
	recovered := uint64(0)
	for i, k := range keys {
		ai, err := l1.AccountInfo(ctx, k.PublicKey())
		if err != nil || ai == nil {
			fmt.Printf("  sender%d %s  empty\n", i, k.PublicKey())
			continue
		}
		bal, err := l1.Balance(ctx, k.PublicKey())
		if err != nil || bal <= 5000 {
			fmt.Printf("  sender%d %s  %d lamports, not worth sweeping\n", i, k.PublicKey(), bal)
			continue
		}
		amount := bal - 5000 // leave exactly the fee
		data := make([]byte, 12)
		binary.LittleEndian.PutUint32(data[0:], 2)
		binary.LittleEndian.PutUint64(data[4:], amount)
		bh, _, err := l1.LatestBlockhash(ctx, "finalized")
		if err != nil {
			fmt.Println("  blockhash:", err)
			continue
		}
		tx, err := buildSigned([]solana.Instruction{
			solana.NewInstruction(systemProgram, solana.AccountMetaSlice{
				solana.NewAccountMeta(k.PublicKey(), true, true),
				solana.NewAccountMeta(wallet.PublicKey(), true, false),
			}, data),
		}, bh, k)
		if err != nil {
			fmt.Println("  build:", err)
			continue
		}
		sig, err := l1.SendTx(ctx, tx, false)
		if err != nil {
			fmt.Printf("  sender%d send failed: %v\n", i, err)
			continue
		}
		fmt.Printf("  sender%d recovered %.6f SOL -> %s\n", i, float64(amount)/1e9, sig[:16])
		recovered += amount
		time.Sleep(400 * time.Millisecond)
	}
	fmt.Printf("\nrecovered %.6f SOL total\n", float64(recovered)/1e9)
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

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
)

var (
	fMode      = flag.String("mode", "", "fund|register|post-job|delegate|prepare|bid|award|undelegate|poll|settle|verify|addresses|lifecycle")
	fL1        = flag.String("l1", envOr("L1_RPC", "https://api.devnet.solana.com"), "L1 RPC (or set L1_RPC)")
	fER        = flag.String("er", "https://devnet-as.magicblock.app", "ER RPC")
	fValidator = flag.String("validator", "MAS1Dt9qreoRMQ14YQuhg8UTZMMzDdKhmkZMECCzk57", "ER validator identity, identical for every delegation")
	fWallet    = flag.String("wallet", os.Getenv("HOME")+"/.config/solana/swarm-deployer.json", "requester / payer keypair")
	fKeyDir    = flag.String("keydir", "keys", "directory holding the agent keypairs")
	fAgents    = flag.Int("agents", 6, "number of agents")
	fBids      = flag.Int("bids", 200, "total bids to submit (0 = unbounded, use --duration)")
	fN         = flag.Int("n", -1, "alias for --bids, as used in the runbook")
	fSenders   = flag.Int("senders", 0, "concurrent senders, defaults to --agents")
	fDuration  = flag.Duration("duration", 0, "time-box the bid phase instead of a fixed count")
	fBudget    = flag.Uint64("budget", 20_000_000, "escrow budget in lamports")
	fBidWindow = flag.Duration("bid-window", 0, "spread bids over this long (0 = as fast as possible)")
	fJobID     = flag.Int64("job-id", -1, "job id; -1 reads .swarm-job-id and auto-increments after settle")
)

const (
	startBidBps = 9900
	minBidBps   = 4000
	bidStepBps  = 20
)

func main() {
	flag.Parse()
	ctx := context.Background()
	if *fMode == "" {
		flag.Usage()
		os.Exit(2)
	}

	// --n is the runbook spelling of --bids.
	if *fN >= 0 {
		*fBids = *fN
	}
	if *fSenders <= 0 || *fSenders > *fAgents {
		*fSenders = *fAgents
	}
	if *fBids == 0 && *fDuration == 0 {
		fmt.Println("FATAL: --bids 0 requires --duration")
		os.Exit(2)
	}

	wallet, err := solana.PrivateKeyFromSolanaKeygenFile(*fWallet)
	check(err)
	agents := loadOrCreateKeys(*fAgents)

	l1 := NewClient(*fL1, *fAgents+4)
	er := NewClient(*fER, *fAgents+4)

	d := &Driver{
		ctx: ctx, l1: l1, er: er,
		wallet: wallet, agents: agents,
		validator: solana.MustPublicKeyFromBase58(*fValidator),
		jobID:     resolveJobID(),
	}

	fmt.Printf("program   %s\n", programID)
	fmt.Printf("requester %s\n", wallet.PublicKey())
	fmt.Printf("job id    %d\n", d.jobID)
	fmt.Printf("job       %s\n", d.job())
	fmt.Printf("escrow    %s\n", escrowPDA(d.job()))
	fmt.Printf("L1        %s\nER        %s\nvalidator %s\n", *fL1, *fER, d.validator)

	switch *fMode {
	case "fund":
		d.fund()
	case "register":
		d.register()
	case "post", "post-job":
		d.post()
	case "delegate":
		d.delegate()
	case "bid":
		d.bid()
	case "award":
		d.award()
	case "undelegate":
		d.undelegate()
	case "poll":
		d.poll()
	case "settle":
		d.settle()
	case "prepare":
		d.prepare()
	case "addresses":
		d.addresses()
	case "verify":
		d.verify(nil)
	case "lifecycle":
		d.lifecycle()
	default:
		fmt.Println("unknown mode", *fMode)
		os.Exit(2)
	}
}

func check(err error) {
	if err != nil {
		fmt.Println("\nFATAL:", err)
		os.Exit(1)
	}
}

type Driver struct {
	ctx       context.Context
	l1, er    *Client
	wallet    solana.PrivateKey
	agents    []solana.PrivateKey
	validator solana.PublicKey
	jobID     uint64
	step      int
}

// job is the Job PDA for the current job id.
func (d *Driver) job() solana.PublicKey { return jobPDA(d.wallet.PublicKey(), d.jobID) }

const jobIDFile = ".swarm-job-id"

// resolveJobID prefers an explicit --job-id, otherwise reads the local file so
// repeated recording takes do not have to be tracked by hand.
func resolveJobID() uint64 {
	if *fJobID >= 0 {
		return uint64(*fJobID)
	}
	b, err := os.ReadFile(jobIDFile)
	if err != nil {
		return 1
	}
	n, err := strconv.ParseUint(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return 1
	}
	return n
}

// bumpJobID advances the counter so the next run starts a fresh Job. Only
// called after a settle actually succeeds.
func bumpJobID(current uint64) {
	if *fJobID >= 0 {
		return // explicit id, caller is managing it
	}
	_ = os.WriteFile(jobIDFile, []byte(strconv.FormatUint(current+1, 10)), 0o644)
}

// ------------------------------------------------------------ step output

func (d *Driver) begin(label string) {
	d.step++
	fmt.Printf("\n[%d/9] %s\n", d.step, label)
}

func ok(format string, a ...interface{}) {
	fmt.Printf("    ok  "+format+"\n", a...)
}

func info(format string, a ...interface{}) {
	fmt.Printf("        "+format+"\n", a...)
}

func sol(lamports uint64) string {
	return fmt.Sprintf("%.9f SOL", float64(lamports)/1e9)
}

// ------------------------------------------------------------------ setup

func loadOrCreateKeys(n int) []solana.PrivateKey {
	check(os.MkdirAll(*fKeyDir, 0o700))
	out := make([]solana.PrivateKey, 0, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(*fKeyDir, fmt.Sprintf("agent%d.json", i))
		if b, err := os.ReadFile(p); err == nil {
			kp, err := solana.PrivateKeyFromSolanaKeygenFileBytes(b)
			check(err)
			out = append(out, kp)
			continue
		}
		kp, err := solana.NewRandomPrivateKey()
		check(err)
		// JSON array of byte values, so the solana CLI can read the file too.
		nums := make([]int, len(kp))
		for j, c := range []byte(kp) {
			nums[j] = int(c)
		}
		b, _ := json.Marshal(nums)
		check(os.WriteFile(p, b, 0o600))
		out = append(out, kp)
	}
	return out
}

// sendL1 submits and waits for confirmation, since every L1 step here is a
// one-off whose result the next step depends on.
func (d *Driver) sendL1(ixs []solana.Instruction, signers ...solana.PrivateKey) string {
	bh, _, err := d.l1.LatestBlockhash(d.ctx, "finalized")
	check(err)
	payer := d.wallet
	extra := signers
	if len(signers) > 0 {
		payer = signers[0]
		extra = signers[1:]
	}
	tx, err := buildSigned(ixs, bh, payer, extra...)
	check(err)
	sig, err := d.l1.SendTx(d.ctx, tx, false)
	check(err)
	check(d.awaitL1(sig))
	return sig
}

// awaitL1 waits for confirmation without hammering the RPC into rate limiting.
//
// The old version polled every 700ms for 90s, which is ~128 getSignatureStatuses
// calls against a public endpoint that caps a single method near 40 per 10s. The
// reads started failing, and a failed read looked identical to an unconfirmed
// transaction, so it timed out on transactions that had actually landed. Every
// caller here is idempotent, so a shorter deadline plus a retry is strictly
// better than a long wait that reports a false failure.
// sendL1NoWait submits without waiting for confirmation. Only for callers that
// verify the resulting state afterwards, which is strictly more reliable than
// polling getSignatureStatuses on a rate limited endpoint.
func (d *Driver) sendL1NoWait(ixs []solana.Instruction, signers ...solana.PrivateKey) string {
	bh, _, err := d.l1.LatestBlockhash(d.ctx, "finalized")
	check(err)
	payer := d.wallet
	extra := signers
	if len(signers) > 0 {
		payer = signers[0]
		extra = signers[1:]
	}
	tx, err := buildSigned(ixs, bh, payer, extra...)
	check(err)
	sig, err := d.l1.SendTx(d.ctx, tx, false)
	check(err)
	return sig
}

func (d *Driver) awaitL1(sig string) error {
	deadline := time.Now().Add(25 * time.Second)
	readFails := 0
	for {
		landed, status, err := d.l1.SignatureLanded(d.ctx, sig)
		if err != nil {
			readFails++
		} else if landed {
			return nil
		} else if status != "" {
			return fmt.Errorf("tx %s failed on L1: %s", sig[:16], status)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("tx %s not confirmed within 25s (%d read failures); it may still land, retry is safe",
				sig[:16], readFails)
		}
		time.Sleep(2 * time.Second)
	}
}

func (d *Driver) fund() {
	d.begin("fund agent keypairs on L1")
	var ixs []solana.Instruction
	for _, a := range d.agents {
		bal, err := d.l1.Balance(d.ctx, a.PublicKey())
		check(err)
		if bal >= 3_000_000 {
			info("%s already funded (%s)", a.PublicKey(), sol(bal))
			continue
		}
		ixs = append(ixs, systemTransfer(d.wallet.PublicKey(), a.PublicKey(), 3_000_000-bal))
	}
	if len(ixs) == 0 {
		ok("all %d agents already funded", len(d.agents))
		return
	}
	sig := d.sendL1(ixs)
	ok("funded %d agents, sig %s", len(ixs), sig[:20])
}

func (d *Driver) register() {
	d.begin("register agents on L1")
	for i, a := range d.agents {
		if ai, _ := d.l1.AccountInfo(d.ctx, agentPDA(a.PublicKey())); ai != nil {
			info("agent %d already registered %s", i, agentPDA(a.PublicKey()))
			continue
		}
		sig := d.sendL1([]solana.Instruction{ixRegisterAgent(a.PublicKey(), uint8(i))}, a)
		info("agent %d %s -> %s", i, a.PublicKey(), sig[:20])
	}
	ok("%d agents registered", len(d.agents))
}

func (d *Driver) post() {
	d.begin("post job and fund escrow on L1")
	job := d.job()
	if ai, _ := d.l1.AccountInfo(d.ctx, job); ai != nil {
		ok("job already exists at %s", job)
		return
	}
	slot, err := d.l1.Slot(d.ctx)
	check(err)
	descHash := sha256.Sum256([]byte("render a 30s product demo video"))

	sig := d.sendL1([]solana.Instruction{
		ixPostJob(d.wallet.PublicKey(), d.jobID, descHash, *fBudget, slot+10_000),
	})
	ok("post_job sig %s", sig[:20])

	esc := escrowPDA(job)
	bal, err := d.l1.Balance(d.ctx, esc)
	check(err)
	ok("escrow %s holds %s (budget %s + rent)", esc, sol(bal), sol(*fBudget))
}

// delegate is the isolated risky step. It is callable on its own precisely so
// multi account delegation can be exercised without running the whole lifecycle.
func (d *Driver) delegate() {
	d.begin("delegate job + registries into the ER")
	info("one delegated account per transaction: seven in one instruction")
	info("overflows the BPF stack (4416 bytes against a 4096 limit)")
	info("validator identity is identical for all %d: %s", len(d.agents)+1, d.validator)

	job := d.job()
	if ai, _ := d.l1.AccountInfo(d.ctx, job); ai == nil || !ai.Owner.Equals(delegationProgram) {
		sig := d.sendL1NoWait([]solana.Instruction{
			ixDelegateJob(d.wallet.PublicKey(), d.wallet.PublicKey(), d.validator, d.jobID),
		})
		info("job      -> %s", sig[:20])
	} else {
		info("job      already delegated")
	}

	for i, a := range d.agents {
		pda := agentPDA(a.PublicKey())
		if ai, _ := d.l1.AccountInfo(d.ctx, pda); ai != nil && ai.Owner.Equals(delegationProgram) {
			info("agent %d  already delegated", i)
			continue
		}
		sig := d.sendL1NoWait([]solana.Instruction{
			ixDelegateAgent(d.wallet.PublicKey(), a.PublicKey(), d.validator),
		})
		info("agent %d  -> %s", i, sig[:20])
	}

	// Confirm every account is live in the ER before returning.
	targets := []solana.PublicKey{job}
	for _, a := range d.agents {
		targets = append(targets, agentPDA(a.PublicKey()))
	}
	deadline := time.Now().Add(60 * time.Second)
	for {
		live := 0
		for _, t := range targets {
			if ai, _ := d.er.AccountInfo(d.ctx, t); ai != nil && ai.Owner.Equals(programID) {
				live++
			}
		}
		if live == len(targets) {
			ok("all %d accounts live in the ER under one validator", len(targets))
			return
		}
		if time.Now().After(deadline) {
			check(fmt.Errorf("only %d/%d accounts appeared in the ER within 60s", live, len(targets)))
		}
		time.Sleep(1 * time.Second)
	}
}

// bid submits decreasing-price bids from concurrent senders. Deliberately dumb:
// no cost curves, no websocket, no reaction to the current best. This exists to
// prove the program, not to be the agents.
func (d *Driver) bid() {
	senders := *fSenders
	label := fmt.Sprintf("submit %d bids from %d concurrent senders", *fBids, senders)
	if *fDuration > 0 {
		label = fmt.Sprintf("submit bids for %v from %d concurrent senders", *fDuration, senders)
	}
	d.begin(label)
	job := d.job()

	before, err := d.readJob(d.er, job)
	check(err)
	info("bid_count before %d, best %d bps", before.BidCount, before.BestBidBps)

	bh, stop := startBlockhash(d.ctx, d.er)
	defer stop()

	ctx := d.ctx
	if *fDuration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(d.ctx, *fDuration)
		defer cancel()
	}

	var seq int64
	var sent, failed int64
	errs := map[string]int{}
	var mu sync.Mutex

	// With --duration the count is unbounded and the context ends the run.
	per, extra := 0, 0
	if *fBids > 0 {
		per = *fBids / senders
		extra = *fBids % senders
	}

	var wg sync.WaitGroup
	t0 := time.Now()
	for i := 0; i < senders; i++ {
		count := per
		if i < extra {
			count++
		}
		wg.Add(1)
		go func(idx, count int, kp solana.PrivateKey) {
			defer wg.Done()
			for n := 0; *fDuration > 0 || n < count; n++ {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if *fBidWindow > 0 && count > 0 {
					time.Sleep(*fBidWindow / time.Duration(count))
				}
				// Monotonically decreasing across all senders, so the curve
				// converges and the final best bidder is well defined.
				k := atomic.AddInt64(&seq, 1)
				bid := startBidBps - int(k)*bidStepBps
				if bid < minBidBps {
					bid = minBidBps
				}
				tx, err := buildSigned(
					[]solana.Instruction{ixSubmitBid(job, kp.PublicKey(), uint16(bid))},
					bh.get(), kp)
				if err != nil {
					mu.Lock()
					errs["build:"+truncate(err.Error(), 40)]++
					mu.Unlock()
					atomic.AddInt64(&failed, 1)
					continue
				}
				// Fire and forget. Awaiting confirmation caps throughput at
				// roughly 4 tx/s, measured.
				if _, err := d.er.SendTx(d.ctx, tx, true); err != nil {
					mu.Lock()
					errs[classify(err)]++
					mu.Unlock()
					atomic.AddInt64(&failed, 1)
					continue
				}
				atomic.AddInt64(&sent, 1)
			}
		}(i, count, d.agents[i])
	}
	wg.Wait()
	wall := time.Since(t0)

	after := d.settleCounter(job, before.BidCount)
	landed := int(after.BidCount - before.BidCount)
	ok("sent %d, failed %d, landed %d in %v", sent, failed, landed, wall.Round(time.Millisecond))
	ok("%.1f bids/s landed", float64(landed)/wall.Seconds())
	ok("best bid %d bps by %s", after.BestBidBps, after.BestBidder)
	if len(errs) > 0 {
		for k, v := range errs {
			info("error %-40s %d", truncate(k, 40), v)
		}
	}

	d.inspectAgents(after)
}

func classify(err error) string {
	s := err.Error()
	switch {
	case strings.Contains(s, "Blockhash not found"):
		return "blockhash-not-found"
	case strings.Contains(s, "AccountInUse"):
		return "account-in-use"
	case strings.Contains(s, "429"), strings.Contains(s, "Too Many"):
		return "rate-limited"
	default:
		return truncate(s, 60)
	}
}

// inspectAgents is the Phase 3 check: every registry must carry its own live
// bid, and they must not all be the same value or the agent cards will look
// frozen on camera.
func (d *Driver) inspectAgents(job *Job) {
	fmt.Printf("\n    per-agent registries (read from the ER):\n")
	fmt.Printf("    %-10s %-46s %12s %10s\n", "AGENT", "PDA", "LAST BID", "BID COUNT")
	seen := map[uint16]int{}
	total := uint32(0)
	populated := 0
	for i, a := range d.agents {
		pda := agentPDA(a.PublicKey())
		ai, err := d.er.AccountInfo(d.ctx, pda)
		if err != nil || ai == nil {
			fmt.Printf("    agent%-5d %-46s %12s %10s\n", i, pda, "READ FAIL", "-")
			continue
		}
		ag, err := parseAgent(ai.Data)
		if err != nil {
			fmt.Printf("    agent%-5d %-46s %12s %10s\n", i, pda, "DECODE FAIL", "-")
			continue
		}
		fmt.Printf("    agent%-5d %-46s %12d %10d\n", i, pda, ag.LastBidBps, ag.BidCount)
		if ag.LastBidBps > 0 {
			populated++
			seen[ag.LastBidBps]++
		}
		total += ag.BidCount
	}

	fmt.Println()
	if populated != len(d.agents) {
		fmt.Printf("    FAIL  only %d/%d registries carry a bid, cards will freeze on camera\n",
			populated, len(d.agents))
	} else if len(seen) < 2 {
		fmt.Printf("    WARN  every agent shows the same bid, the cards will look static\n")
	} else {
		fmt.Printf("    ok    all %d registries populated, %d distinct values\n", len(d.agents), len(seen))
	}
	// Per agent counts are display only. The job total is authoritative.
	if total != job.BidCount {
		fmt.Printf("    note  per-agent sum %d vs Job.bid_count %d. The job total is\n", total, job.BidCount)
		fmt.Printf("          authoritative and is what the counter renders.\n")
	}
	if job.BestBidder.IsZero() {
		fmt.Printf("    FAIL  best_bidder is unset\n")
		return
	}
	for i, a := range d.agents {
		if a.PublicKey().Equals(job.BestBidder) {
			fmt.Printf("    ok    best_bidder is agent%d (%s)\n", i, job.BestBidder)
			return
		}
	}
	fmt.Printf("    FAIL  best_bidder %s is not one of the six agents\n", job.BestBidder)
}

// settleCounter waits until bid_count stops moving so "landed" comes from chain
// state rather than from what the driver believes it sent.
func (d *Driver) settleCounter(job solana.PublicKey, base uint32) *Job {
	var last *Job
	stable := 0
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		j, err := d.readJob(d.er, job)
		if err != nil {
			continue
		}
		if last != nil && j.BidCount == last.BidCount {
			stable++
			if stable >= 4 {
				return j
			}
		} else {
			stable = 0
		}
		last = j
	}
	if last == nil {
		check(fmt.Errorf("could not read job from ER while settling bid count"))
	}
	return last
}

func (d *Driver) award() {
	d.begin("award job in the ER")
	job := d.job()
	j, err := d.readJob(d.er, job)
	check(err)
	if j.Status == StatusAwarded {
		ok("already awarded to %s at %d bps", j.BestBidder, j.BestBidBps)
		return
	}
	if j.BestBidder.IsZero() {
		check(fmt.Errorf("no bids recorded, nothing to award"))
	}

	bh, _, err := d.er.LatestBlockhash(d.ctx, "confirmed")
	check(err)
	tx, err := buildSigned(
		[]solana.Instruction{ixAwardJob(job, agentPDA(j.BestBidder), d.wallet.PublicKey())},
		bh, d.wallet)
	check(err)
	sig, err := d.er.SendTx(d.ctx, tx, false)
	check(err)
	ok("award sig (ER) %s", sig[:20])
	fmt.Printf("SIG award=%s\n", sig)

	deadline := time.Now().Add(30 * time.Second)
	for {
		j, err := d.readJob(d.er, job)
		if err == nil && j.Status == StatusAwarded {
			ok("status %s, winner %s at %d bps", j.Status, j.BestBidder, j.BestBidBps)
			return
		}
		if time.Now().After(deadline) {
			check(fmt.Errorf("award did not take effect within 30s"))
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// undelegateStamp records when the intent was scheduled so `poll` can report
// the true undelegate-to-L1 gap, which is dead air in the video.
const undelegateStamp = ".swarm-undelegate-at"

// undelegate only schedules the intent. The L1 write happens later, executed by
// the validator. Run `poll` next.
func (d *Driver) undelegate() {
	d.begin("commit and undelegate back to L1 (schedules intent only)")
	job := d.job()
	registries := make([]solana.PublicKey, 0, len(d.agents))
	for _, a := range d.agents {
		registries = append(registries, agentPDA(a.PublicKey()))
	}

	bh, _, err := d.er.LatestBlockhash(d.ctx, "confirmed")
	check(err)
	tx, err := buildSigned(
		[]solana.Instruction{ixCommitAndUndelegate(d.wallet.PublicKey(), job, registries)},
		bh, d.wallet)
	check(err)
	sig, err := d.er.SendTx(d.ctx, tx, false)
	check(err)

	now := time.Now()
	_ = os.WriteFile(undelegateStamp, []byte(strconv.FormatInt(now.UnixMilli(), 10)), 0o644)
	ok("undelegate scheduled (ER) %s", sig[:20])
	fmt.Printf("SIG undelegate=%s\n", sig)
	info("intent registered at %s", now.Format("15:04:05.000"))
	info("no synchronous path exists. run: driver --mode poll")
}

// poll waits for ownership of all seven accounts to return to the program on
// L1 and reports how long the whole gap took.
func (d *Driver) poll() {
	d.begin("poll L1 until undelegation lands")
	job := d.job()
	targets := []solana.PublicKey{job}
	for _, a := range d.agents {
		targets = append(targets, agentPDA(a.PublicKey()))
	}

	scheduledAt := time.Now()
	if b, err := os.ReadFile(undelegateStamp); err == nil {
		if ms, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64); err == nil {
			scheduledAt = time.UnixMilli(ms)
			info("intent was scheduled %v ago", time.Since(scheduledAt).Round(time.Millisecond))
		}
	} else {
		info("no %s found, timing from now", undelegateStamp)
	}

	t0 := time.Now()
	deadline := time.Now().Add(180 * time.Second)
	lastBack := -1
	readFails := 0

	for {
		// One request for all seven, not seven requests. And a failed read is
		// reported as a failed read, never as "still delegated".
		infos, err := d.l1.MultipleAccounts(d.ctx, targets)
		if err != nil {
			readFails++
			if readFails%10 == 0 {
				info("L1 read failing (%d times): %v", readFails, err)
			}
			time.Sleep(1500 * time.Millisecond)
			continue
		}
		back := 0
		for _, ai := range infos {
			if ai != nil && ai.Owner.Equals(programID) {
				back++
			}
		}
		if back != lastBack {
			info("%d/%d accounts back on L1 after %v", back, len(targets),
				time.Since(scheduledAt).Round(time.Millisecond))
			lastBack = back
		}
		if back == len(targets) {
			total := time.Since(scheduledAt).Round(time.Millisecond)
			ok("all %d accounts owned by the program on L1", len(targets))
			fmt.Printf("\n    UNDELEGATE TO L1 GAP: %v  (polled for %v, %d read failures)\n",
				total, time.Since(t0).Round(time.Millisecond), readFails)
			fmt.Printf("    plan the video around this: it is dead air\n")
			_ = os.Remove(undelegateStamp)
			return
		}
		if time.Now().After(deadline) {
			check(fmt.Errorf("only %d/%d accounts undelegated within 180s (%d read failures)",
				back, len(targets), readFails))
		}
		time.Sleep(1 * time.Second)
	}
}

func (d *Driver) settle() {
	d.begin("settle on L1, escrow pays the winner")
	job := d.job()
	j, err := d.readJob(d.l1, job)
	check(err)
	if j.Status != StatusAwarded {
		check(fmt.Errorf("job status is %s on L1, expected Awarded", j.Status))
	}

	esc := escrowPDA(job)
	escBefore, err := d.l1.Balance(d.ctx, esc)
	check(err)
	winBefore, err := d.l1.Balance(d.ctx, j.BestBidder)
	check(err)
	reqBefore, err := d.l1.Balance(d.ctx, d.wallet.PublicKey())
	check(err)
	info("escrow  before %s", sol(escBefore))
	info("winner  before %s  (%s)", sol(winBefore), j.BestBidder)

	sig := d.sendL1([]solana.Instruction{ixSettle(d.wallet.PublicKey(), j.BestBidder, d.jobID)})
	ok("settle sig %s", sig[:20])
	fmt.Printf("SIG settle=%s\n", sig)
	ok("explorer https://explorer.solana.com/tx/%s?cluster=devnet", sig)

	d.verify(&balances{esc: escBefore, win: winBefore, req: reqBefore, winner: j.BestBidder})
}

type balances struct {
	esc, win, req uint64
	winner        solana.PublicKey
}

// verify re-reads everything from L1 after the fact. It never trusts values the
// driver was carrying, so the numbers below are chain state.
func (d *Driver) verify(before *balances) {
	d.begin("verify final state on L1")
	job := d.job()
	esc := escrowPDA(job)

	j, err := d.readJob(d.l1, job)
	check(err)

	escAfter, err := d.l1.Balance(d.ctx, esc)
	check(err)
	winAfter, err := d.l1.Balance(d.ctx, j.BestBidder)
	check(err)

	ai, err := d.l1.AccountInfo(d.ctx, agentPDA(j.BestBidder))
	check(err)
	if ai == nil {
		check(fmt.Errorf("winner registry missing on L1"))
	}
	agent, err := parseAgent(ai.Data)
	check(err)

	fmt.Println()
	fmt.Printf("    %-22s %s\n", "job status", j.Status)
	fmt.Printf("    %-22s %d\n", "bid_count", j.BidCount)
	fmt.Printf("    %-22s %d bps\n", "winning bid", j.BestBidBps)
	fmt.Printf("    %-22s %s\n", "winner", j.BestBidder)
	fmt.Printf("    %-22s %s\n", "escrow balance now", sol(escAfter))
	fmt.Printf("    %-22s %s\n", "winner balance now", sol(winAfter))
	fmt.Printf("    %-22s %s\n", "winner earned", sol(agent.Earned))
	fmt.Printf("    %-22s %d\n", "winner completed", agent.Completed)

	if before != nil {
		expected := *fBudget * uint64(j.BestBidBps) / 10_000
		fmt.Println()
		fmt.Printf("    %-22s %s\n", "expected payout", sol(expected))
		fmt.Printf("    %-22s %s\n", "winner delta", sol(winAfter-before.win))
		if winAfter-before.win != expected {
			fmt.Printf("    MISMATCH: winner delta does not equal expected payout\n")
		}
	}

	fmt.Println()
	pass := true
	if j.Status != StatusSettled {
		fmt.Printf("    FAIL  job status is %s, expected Settled\n", j.Status)
		pass = false
	}
	if escAfter != 0 {
		fmt.Printf("    FAIL  escrow still holds %s, expected 0 (account closed)\n", sol(escAfter))
		pass = false
	}
	if agent.Earned == 0 {
		fmt.Printf("    FAIL  winner registry records zero earned\n")
		pass = false
	}
	if pass {
		fmt.Printf("    LIFECYCLE PASSED: escrow drained, winner paid, job Settled\n")
		// Only now advance the id, so a failed run can be retried against the
		// same Job rather than stranding it half finished.
		bumpJobID(d.jobID)
		fmt.Printf("    next run will use job id %d (this one stays on the explorer)\n", d.jobID+1)
	}

	fmt.Println()
	fmt.Printf("    verify independently:\n")
	fmt.Printf("      solana account %s --url devnet\n", esc)
	fmt.Printf("      solana balance %s --url devnet\n", j.BestBidder)
	fmt.Printf("      solana account %s --url devnet\n", job)
}

func (d *Driver) readJob(c *Client, pk solana.PublicKey) (*Job, error) {
	ai, err := c.AccountInfo(d.ctx, pk)
	if err != nil {
		return nil, err
	}
	if ai == nil {
		return nil, fmt.Errorf("job account %s not found", pk)
	}
	return parseJob(ai.Data)
}

func (d *Driver) lifecycle() {
	fmt.Println(strings.Repeat("=", 72))
	d.fund()
	d.register()
	d.post()
	d.delegate()
	d.bid()
	d.award()
	d.undelegate()
	d.poll()
	d.settle()
}

// addresses writes the PDAs the frontend needs so the browser never has to do
// PDA derivation. Consumed by web/lib/deployment.json.
func (d *Driver) addresses() {
	d.begin("emit addresses for the frontend")
	job := d.job()
	type agentAddr struct {
		Name      string `json:"name"`
		Authority string `json:"authority"`
		Pda       string `json:"pda"`
	}
	out := struct {
		JobID     uint64      `json:"jobId"`
		ProgramID string      `json:"programId"`
		ErRpc     string      `json:"erRpc"`
		Requester string      `json:"requester"`
		JobPda    string      `json:"jobPda"`
		EscrowPda string      `json:"escrowPda"`
		Agents    []agentAddr `json:"agents"`
	}{
		JobID:     d.jobID,
		ProgramID: programID.String(),
		ErRpc:     *fER,
		Requester: d.wallet.PublicKey().String(),
		JobPda:    job.String(),
		EscrowPda: escrowPDA(job).String(),
	}
	// Names follow agents/curve.json order, which is also the keydir order.
	names := []string{"atlas", "borealis", "cirrus", "dorado", "echo", "fenrir"}
	for i, a := range d.agents {
		n := fmt.Sprintf("agent%d", i)
		if i < len(names) {
			n = names[i]
		}
		out.Agents = append(out.Agents, agentAddr{
			Name: n, Authority: a.PublicKey().String(), Pda: agentPDA(a.PublicKey()).String(),
		})
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	path := "../web/lib/deployment.json"
	if err := os.WriteFile(path, b, 0o644); err != nil {
		check(err)
	}
	ok("wrote %s", path)
	fmt.Println(string(b))
}


// prepare finds the first unused job id, creates and delegates it, and writes
// the frontend address file. This is what makes the demo button reusable: a
// settled job is spent, and award on one fails with BadState (0x1770).
func (d *Driver) prepare() {
	d.begin("prepare a fresh job")

	found := uint64(0)
	for id := d.jobID; id < d.jobID+200; id++ {
		pda := jobPDA(d.wallet.PublicKey(), id)
		ai, err := d.l1.AccountInfo(d.ctx, pda)
		check(err)

		if ai == nil {
			found = id // never created
			break
		}
		// A delegated job is frozen on L1 at its pre-delegation state, so it
		// reads Open even mid-auction. Ownership is the reliable signal.
		if ai.Owner.Equals(delegationProgram) {
			info("job %d is delegated and in use, trying next", id)
			continue
		}
		j, err := parseJob(ai.Data)
		if err == nil && j.Status == StatusOpen {
			found = id // created but never delegated, reuse and skip the rent
			break
		}
		status := "unreadable"
		if err == nil {
			status = j.Status.String()
		}
		info("job %d already used (%s), trying next", id, status)
	}
	if found == 0 {
		check(fmt.Errorf("no free job id found in 200 attempts"))
	}
	d.jobID = found
	ok("using job id %d", d.jobID)

	if ai, _ := d.l1.AccountInfo(d.ctx, d.job()); ai == nil {
		d.post()
	}
	d.delegate()
	d.addresses()

	// Parsed by the web orchestrator.
	fmt.Printf("\nPREPARED job-id=%d pda=%s escrow=%s\n", d.jobID, d.job(), escrowPDA(d.job()))
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

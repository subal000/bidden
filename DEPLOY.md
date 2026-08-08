# Deploy runbook

Do not start until the balance clears 4 SOL. A partial run that strands you mid-lifecycle
with no SOL to retry is the worst outcome available.

The swarm deploy uses its own keypair, kept separate from the wallet ReserveSentinel signs
with, so the two projects never share a fee payer.

  deployer   ~/.config/solana/swarm-deployer.json
             2DzDL6VrF1uzjcjd3z68u3GgSxrGzor4RPfJqNgSjucd
  funded     6 SOL

Anchor.toml and the driver both default to it. Nothing below needs --wallet.

Predicted deploy cost: **2.30453952 SOL** program rent, but `anchor deploy` actually costs
**2.3319 SOL**. The difference is an IDL metadata account Anchor 1.x writes on chain
(0.0242) plus deploy transaction fees (0.0029). Budget for it.

Full setup after the deploy, measured: **0.0631 SOL** for funding 6 agents, 6 registrations,
job, escrow, budget, and all 7 delegations including their 21 delegation PDAs.

---

## Phase 1 — deploy, once

```bash
solana balance --url devnet          # confirm >= 4 SOL
cd swarm
anchor deploy                        # record actual cost vs 2.30453952 predicted
solana balance --url devnet          # record
```

If the deploy fails partway:

```bash
solana program close --buffers --url devnet
```

before retrying, or the stranded buffer eats the rent you need for attempt two.

Record the real deploy cost. If it diverges from 2.30453952 by more than a few milli-SOL,
the rent model is off and every downstream number needs rechecking.

`--mode addresses` runs in Phase 2, after the agent keypairs exist. Running it here would
emit PDAs for accounts that have not been created yet.

---

## Phase 2 — setup, then stop and look

```bash
cd swarm/driver
go run . --mode fund                 # REQUIRED FIRST: agents pay their own rent
go run . --mode register             # 6 agents, creates keys/agent{0..5}.json
go run . --mode post-job             # job + funded escrow, uses job id from .swarm-job-id
go run . --mode addresses            # writes web/lib/deployment.json
solana balance --url devnet          # record what setup actually cost
```

Then the delegation, which is the unproven part. Seven separate transactions, one account
each, identical validator identity:

```bash
go run . --mode delegate
```

**Stop here.** Verify all seven accounts are owned by
`DELeGGvXpWV2fqJUhqcF5ZSYMS4JTLjteaAMARRSaeSh` on L1, and that all seven read correctly
from the ER endpoint. The mode does both checks and will not return until every account is
live in the ER, but confirm by hand as well:

```bash
solana account <JOB_PDA> --url devnet | grep Owner
solana account <AGENT_PDA> --url devnet | grep Owner
```

If any single account failed to delegate, transactions touching it will fail and you need
to know now, not during a recording.

Record the delegation PDA cost. This is the number that was never measured: 21 accounts
(buffer, record, metadata for each of 7 delegated accounts).

---

## Phase 3 — dry run before anything is on camera

Short burst first, not the full thirty seconds:

```bash
go run . --mode bid --n 50 --senders 6
```

The mode ends by reading every registry off the ER and asserting the three things that
matter. Verify:

- `Job.bid_count` moved by 50
- each `AgentRegistry.last_bid_bps` is populated and distinct
- `best_bidder` is one of the six

It prints `ok    all 6 registries populated, N distinct values` on success, and fails loudly
with `only N/6 registries carry a bid, cards will freeze on camera` otherwise.

Then flip `LIVE = true` in `swarm/web/app/page.tsx`, run the frontend, and watch a second
50-bid burst render:

```bash
cd swarm/web && npm run dev
```

Confirm all six cards move, the badge says **live**, and the counter is monotonic. If the
cards look frozen or the counter stutters, fix it here. Screen recording is the last thing
that happens, not the first.

**Open the browser before starting the agents.** ChainSource records a baseline for the job
and for each agent registry on its first poll, so it can show this session rather than the
account's whole history. Attach late and the cards will show a small number while the
counter shows a large one, and the screen contradicts itself.

**Devnet confirmation is flaky.** `register` and `delegate` both hit `not confirmed within
90s` partway through during the real run. The transactions had landed; both modes skip
completed work, so re-running resumes cleanly. Do not panic mid-take.

---

## Phase 4 — full dress rehearsal, unrecorded

Complete lifecycle, start to finish, nothing recorded:

```bash
cd swarm/driver
go run . --mode bid --n 0 --duration 30s --senders 6
go run . --mode award
go run . --mode undelegate           # schedules intent, stamps .swarm-undelegate-at
go run . --mode poll                 # waits for L1 owner flip, reports the true gap
go run . --mode settle               # L1, pays escrow
go run . --mode verify
```

`poll` prints:

```
UNDELEGATE TO L1 GAP: <X>s  (polled for <Y>s)
plan the video around this: it is dead air
```

**MEASURED: 19.973s** on job 5, all seven accounts flipping together, zero read failures.
Twenty seconds of dead air, so the video cuts rather than holds. See VIDEO.md shot 5.

Confirm escrow drained, winner's balance up, status Settled, and grab the explorer link.
A successful `verify` auto-increments `.swarm-job-id`, so the next take gets a fresh Job.

---

## Phase 5 — record

Only now. The job id has already advanced, so this run creates a new Job account and the
rehearsal's settled Job stays on the explorer.

```bash
go run . --mode post-job             # new job id, fresh Job + Escrow
go run . --mode delegate
go run . --mode addresses            # refresh deployment.json with the new job PDA
# restart the frontend so it picks up the new deployment.json
```

Then run the lifecycle and record the second successful run.

Two things that must be true on camera:

- Every number quoted comes from this live run, never from the mock
- The badge reads **live**

Have the mocked build ready as a fallback in case devnet degrades mid-recording. It is
already zero-network: set `LIVE = false` and it runs with devnet unreachable.

### Unlimited takes

`job_id` is part of the Job and Escrow PDA seeds. `.swarm-job-id` holds the current value
and advances only after a verified settle, so a failed run can be retried against the same
Job rather than stranding it half finished. Override with `--job-id N` at any time.

Every previous settled Job stays on the explorer permanently. The Job account is never
closed at settle, deliberately: the settled Job is the closing shot of the video and it
needs to still be there when a judge clicks it next week.

---

## A failed read is not a zero. Three times now.

This exact bug has appeared in three different places and cost more time than anything else
in the project:

1. `bench/settle()` treated a rate-limited `getAccountInfo` as "counter is 0" and reported
   a negative landed count.
2. `driver --mode poll` fired 7 `getAccountInfo` calls per second at public devnet, far past
   the ~40-per-10s cap, and counted every failed read as "still delegated". It reported a
   false timeout on an undelegation that had actually succeeded. Now one
   `getMultipleAccounts` per second, with read failures reported as read failures.
3. `Balance()` omitted a commitment and defaulted to finalized, so `settle` reported an
   untouched escrow and an unpaid winner while the chain disagreed. Now reads at confirmed.

If a read can fail, handle the failure explicitly. Never let it fall through to a number.

## Two bugs that cost an hour, do not reintroduce them

**Bids must be unique transactions.** An agent that cannot improve the price holds,
resubmitting the same value. Same signer, same instruction, same bid, same cached blockhash
hashes to one signature and the cluster dedupes it. Throughput measured 10.0, then 6.4, then
3.3 bids/s across three consecutive runs as the price converged and more agents held. Adding
a varying ComputeBudget unit limit took it to **32.9 bids/s**. See `agents/chain.go`.

**Never discard send errors.** `_ = m.Bid(...)` hid the above for three runs. A silent send
failure is indistinguishable from a slow market. `ChainMarket.SendErrors()` now reports them
and the live summary prints them.

## If SOL never arrives

You still submit. See the fallback closing in SUBMISSION.md. The `LIVE` flag driving the
badge is what keeps that submission honest, so do not remove it.

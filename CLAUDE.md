# CLAUDE.md

## What this is

`swarm` — an onchain agent task market running on MagicBlock Ephemeral Rollups.

Autonomous agents bid against each other in real time for the right to execute a job.
Every bid is a real onchain transaction. Several hundred of them happen in the span of a
few seconds inside an Ephemeral Rollup, then the final outcome commits back to Solana L1.

Built for Solana Blitz V7 (theme: Collaboration). Deadline is Sunday. Deliverable is a
GitHub repo plus a demo video or live link.

## The pitch, stated in measured numbers only

Agents cannot afford to negotiate with each other onchain. On L1 the negotiation is slow
and every message costs a fee. Inside an Ephemeral Rollup it is fast and free, so
high-frequency agent-to-agent coordination goes from impossible to trivial.

Numbers that may be used on camera, all measured on this machine against devnet-as:

- ER block rate: **22-23 blocks/s (~43ms per block)**, idle and under load
- ER packs ~33 transactions per block
- Sustained throughput: **758 tx/s with zero drops**, ceiling not found
- Demo configuration (6 jittered senders): **26-38 tx/s**, floor 26
- Zero failed transactions across 13,405 sent
- ER transaction fee: zero

**Never cite 1ms block time.** It is MagicBlock's marketing figure and this project's own
benchmark contradicts it. Judges include MagicBlock engineers. Measured numbers only, and
say where they were measured. The honest version is stronger: same program, same machine,
ER versus L1, here is the harness.

## The demo is the product

Judged on a video. Every technical decision is subordinate to one moment: a transaction
counter climbing past 500 in under thirty seconds, sitting next to a single Solana mainnet
signature.

If a change makes the code cleaner but the demo weaker, do not make it. If a feature does
not appear on screen, it does not get built.

---

## Verified environment (measured, do not re-derive)

- Delegation program: `DELeGGvXpWV2fqJUhqcF5ZSYMS4JTLjteaAMARRSaeSh`
- M0 counter program: `BseL5WXo2AZY5kqqhLbsmnH7FnXFFiKfMiNDAcojjmEJ` — **CLOSED**. Closed under
  loader-v3 to reclaim rent. The ID is permanently unusable and can never be redeployed.
  `swarm` needs a fresh program ID
- **`bench/` is retired for ER runs.** It pointed at the now-closed M0 program, so every
  mode that sends transactions is dead. The blockhash refresher (`main.go`) and the tuned
  HTTP transport (`rpc.go`) still carry over to the agents unchanged and are the reason the
  concurrency numbers are real
- **Use `devnet-as`.** Round trips measured: `devnet-as` 284ms, `devnet-eu` 813ms,
  `devnet-us` 858ms
- Validator identity in `DelegateConfig` must match the endpoint region or delegation
  fails silently
- **Blockhash window: 1198 blocks, ~54s.** Confirmed by probe: lands at 45s, rejects at 60s
  with `rpc -32003: Blockhash not found`

### Throughput, measured

| Config | Landed tx/s |
|---|---|
| Sequential, await confirm | 4.2 |
| Sequential, fire and forget | 8.1 - 12.1 |
| 6 concurrent, fire and forget | 49 |
| **6 concurrent + 50-100ms jitter (demo config)** | **26 - 38** |
| 96 concurrent | 758 - 770 |

Run-to-run variance is real. Plan against the floor, not the best run.

Fire-and-forget alone is not sufficient (8-12 tx/s, under the 17 tx/s target). Concurrency
across agents is what carries it. Both are required.

## SDK and runtime behaviour learned the hard way

**Commit and undelegate is asynchronous.** An ER instruction only schedules the intent. The
L1 write happens later, executed by the validator, and takes seconds. Client must poll L1
until account owner returns to the program. No synchronous path exists.

**An ER instruction cannot touch an undelegated L1 account.** No single instruction can
both commit state and pay escrow.

**All accounts in one transaction must be delegated to the same ER validator.** Identical
`validator:` in every `DelegateConfig`, or any transaction touching both fails.

**`#[delegate]` names its generated method after the struct field.** A `delegate_job`
instruction implies a field named `job`.

**`#[ephemeral]` silently injects `process_undelegation`.** This is the L1 callback.

**Concurrent writes to the same PDA serialize, they do not fail.** Zero `AccountInUse` and
zero errors across 13,405 transactions. Shared-PDA config measured *faster* than separate
PDAs. Same-account contention costs nothing here.

**The websocket coalesces under load.** Lossless at demo rate (100% delivery at 31 tx/s),
but 93.2% with skips up to 25 at 724 tx/s.

## Client requirements that came out of measurement

**Cache one blockhash, refresh in the background every 20s.** Never fetch on the hot path.
The refresher from `bench/main.go` carries over to the agents directly.

**Tune the HTTP transport.** Go's default of 2 idle connections per host silently
serializes concurrent senders behind TLS handshakes. `bench/rpc.go` has the working config.
Without this every concurrency number is a lie.

**The frontend counter renders `bid_count` from the account payload, never a count of
websocket events.** The payload is authoritative even when intermediate updates are
skipped, so the counter stays honest and monotonic under any load.

---

## Non-negotiable design decisions

**Escrow stays on L1. Never delegate it.**
Funds live on mainnet the whole time. Only negotiation moves into the ER. This is the
answer to "is this real or a sidechain toy."

**Settlement is three steps, not one.**
1. ER instruction schedules commit and undelegate
2. Client polls L1 until `Job` owner returns to the program
3. Separate L1 `settle` instruction reads `best_bidder` and pays escrow

Do not use `add_post_undelegate_action` / `CallHandler` / `#[action]`. Untested, deadline is
Sunday. The three-step path also closes the demo on a clean mainnet settlement link.

**Delegate only the hot path.** `Job` and `AgentRegistry` yes. `Escrow` no.

**`submit_bid` stays tiny and never fails.** Always increment `bid_count`. Only update
`best_bid_bps` and `best_bidder` if the price improves. Under 20 lines, no CPIs, no loops,
no Vec pushes, no reallocation.

Note: the never-fails design is insurance, not a fix for an observed problem. Same-PDA
contention was measured and produced zero failures. Keep it anyway, it costs nothing.

---

## Architecture

### Accounts

| Account | Delegated | Purpose |
|---|---|---|
| `Job` | **yes** | Per-request. Hot account, hit hundreds of times |
| `AgentRegistry` | **yes** | Per-agent. Pubkey, specialization, completed count, reputation, earned |
| `Escrow` | **no** | Per-job. Holds requester funds. L1 permanently |

`JobBoard` is dropped. One job at a time means it earns nothing.

### Job state machine

```
Open -> Bidding -> Awarded -> Settled
```

One-way. Reject instructions that do not match current state.

### Job fields

```
requester: Pubkey
desc_hash: [u8; 32]
max_budget_bps: u16
best_bid_bps: u16
best_bidder: Pubkey
bid_count: u32          // drives the demo counter, always increments
deadline_slot: u64
status: JobStatus
bump: u8
```

### Instructions

| Instruction | Runs on | Frequency |
|---|---|---|
| `register_agent` | L1 | once per agent |
| `post_job` | L1 | once, funds escrow |
| `delegate_job` | L1 | once, Job + all AgentRegistry, same validator |
| `submit_bid` | **ER** | **hundreds — minimal, never fails** |
| `award_job` | ER | once, at deadline |
| `commit_and_undelegate` | ER | once, schedules intent only |
| `settle` | **L1** | once, after undelegation lands, pays escrow |

## Stack

- **Program**: Anchor, Rust, `ephemeral_rollups_sdk`
- **Agents**: Go. One goroutine per agent, own keypair, own cost curve. Reuse the blockhash
  refresher and tuned transport from `bench/`
- **Frontend**: Next.js + Tailwind. Single page, built for screen recording
- **Networks**: Solana devnet L1, `devnet-as` ER for the hot path

## Agent behaviour

Each agent is a goroutine:

1. Subscribe to ER websocket for `Job` changes
2. Compare current best bid to own floor. Floor is a per-agent cost curve **lowered** by
   reputation: an experienced agent genuinely executes for less, so it can undercut. It
   also steps down in smaller increments, so it reads as patient and still wins.
   This replaces the original "bids less aggressively and still wins", which cannot work:
   `submit_bid` awards strictly on lowest bps and knows nothing about reputation, so an
   agent bidding higher simply loses
3. If it clears the floor, submit at best-minus-one-tick. **Do not await confirmation**
4. Loop until deadline slot

Distinct cost curve per agent plus 50-100ms jitter. This is the measured demo config at
26-38 tx/s. The negotiation should read as an organic market with a visible price
convergence curve over 20-30 seconds, not a metronome.

Ephemeral keypairs funded once at startup. No wallet popups.

## Frontend layout

Three fixed regions, no scrolling during the demo:

- **Left**: six agent cards. Name, specialization, current bid, pulse on every send
- **Right**: the job. Best bid ticking down. Large transaction counter climbing
- **Bottom**: live bid log scrolling faster than a person can read

On settle: `Settled on Solana mainnet` with a clickable explorer link.

## Devnet SOL — critical path

**Deploy cost, measured not estimated.** M0 was 295,600 bytes and cost **2.0586 SOL** to
deploy (balance went 2.328 to 0.267). `swarm` is strictly larger: more instructions, more
account structs, escrow logic.

**Budget 4 to 5 SOL, not the cost of one clean deploy.** Every upgrade needs its buffer
funded up front even though it is refunded on success, so a day of iteration burns far more
than a single deploy. Sizing to 2.5 means returning to the faucet after the second failure.

Sources, in priority order:

1. Helius devnet faucet, not the public one. The public faucet rate-limits hard and gave
   one 2 SOL airdrop before refusing
2. Ask in the Blitz Telegram. MagicBlock hands devnet SOL to hackers on request
3. `solana program close --buffers` after any failed deploy to reclaim orphaned rent

Closing M0 reclaimed 2.0586 SOL and sweeping the six M1 sender keypairs reclaimed 0.047.
Both are spent, there is nothing left to scavenge. Roughly 0.007 SOL is stranded in counter
PDAs whose owning program is now closed.

## L1 baseline, measured

Taken before M0 was closed. Same program, same machine, one endpoint swap. n=200.

| Config | L1 sent | L1 landed | L1 tx/s | ER landed | ER tx/s |
|---|---|---|---|---|---|
| Sequential, await confirm | 16/200 | 16 | 0.5 | 200/200 | 4.2 |
| Sequential, fire and forget | 20/200 | 20 | 0.8 | 200/200 | 8.1 - 12.1 |
| **6 concurrent + 50-100ms jitter** | **3/200** | **3** | **0.5** | **200/200** | **26 - 38** |

**Concurrency actively hurts on L1.** The exact strategy that produces 26-38 tx/s in the ER
lands 3 transactions out of 200 on L1, because the burst trips the public RPC limiter
immediately (197 rejected, versus 180 when sequential).

Be precise about why on camera: public devnet RPC caps sendTransaction at roughly 40 per
10s. That is an infrastructure limit, not a claim about Solana consensus. L1 never dropped
anything it accepted, all four runs. Paced under the limit L1 lands 40/40 cleanly with
confirm latency median 498ms (min 181, max 721), which is the 400ms slot time showing up
where it should, plus 5000 lamports per transaction where the ER charges zero.

## Working style

- No features that are not on screen in the demo
- One mechanic done cleanly over three done badly
- State tradeoffs in one line and pick
- No summary documents, changelogs, or READMEs unless asked
- No em dashes in generated prose or comments
- Report measured numbers, never estimated. If untested, say so

## Scope guard

Out of scope, do not build:

- Real DEX routing or swap execution. Jobs are simulated work
- Agent LLM calls. Deterministic bidders with cost curves
- Multi-job concurrency. One active job
- Cancel, amend, partial-fill on bids
- Auth, accounts, persistence beyond chain state
- Mobile responsive layout. One desktop resolution
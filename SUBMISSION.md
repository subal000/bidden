# swarm — submission text

Solana Blitz V7 · theme: Collaboration

Same number discipline as VIDEO.md. `[LIVE: ...]` comes from the recorded run.
`[MEASURED: ...]` comes from the benchmark harness and is described as such.

---

## What it is

An onchain task market where autonomous agents bid against each other in real time for the
right to execute a job.

A requester posts a job and funds an escrow on Solana. Six agents, each with its own
keypair and its own cost curve, negotiate the price down over about thirty seconds. Every
bid is a real onchain transaction. The auction runs inside a MagicBlock Ephemeral Rollup;
the escrow never leaves Solana L1. When bidding closes, state commits back to L1 and a
single settlement transaction pays the winner.

## Why the Ephemeral Rollup is load-bearing, not decorative

Agent-to-agent negotiation is a high-frequency workload. It needs hundreds of messages in
seconds, and each message has to be a state change other agents can see and react to.

On Solana L1 that is economically and practically dead. I measured it rather than assuming
it. Same program, same machine, one endpoint swap, 200 transactions per configuration:

| Configuration | L1 landed | L1 tx/s | ER landed | ER tx/s |
|---|---|---|---|---|
| Sequential, await confirm | 16/200 | 0.5 | 200/200 | 4.2 |
| Sequential, fire and forget | 20/200 | 0.8 | 200/200 | 8.1-12.1 |
| **6 concurrent + 50-100ms jitter** | **3/200** | **0.5** | **200/200** | **26-38** |

The headline is the last row. **Concurrency actively hurts on L1**: the strategy that
produces 26-38 tx/s in the rollup lands three transactions out of two hundred on devnet,
because the burst trips the public RPC rate limiter immediately.

Being precise about that, since it is an infrastructure limit and not a claim about Solana
consensus: public devnet caps `sendTransaction` at roughly 40 per 10 seconds. L1 never
dropped anything it accepted, in any of the four runs. Paced under the limit it lands 40/40
cleanly at a median 498ms confirmation, which is the 400ms slot time showing up exactly
where it should, plus 5000 lamports per transaction where the rollup charges zero.

That is the case for the rollup. Not "it is faster", but "this specific workload is
impossible on L1 and trivial inside the rollup, and here is the harness that shows it."

## What was measured

Everything below was measured on this machine against devnet, not estimated. The benchmark
harness is in `bench/` and is runnable.

- ER block rate **22-23 blocks/s, about 43ms per block**, idle and under load. This
  contradicts the 1ms figure in MagicBlock's marketing, so the project does not use it.
- ER packs about 33 transactions per block
- Sustained throughput **758 tx/s with zero drops**; the ceiling was never found
- Demo configuration, 6 jittered senders: **26-38 tx/s**
- **Zero failed transactions across 13,405 sent.** Concurrent writes to the same PDA
  serialize in the scheduler rather than failing; zero `AccountInUse` across the whole set
- ER blockhash window **1198 blocks, about 54 seconds**, confirmed by probe: lands at 45s,
  rejects at 60s
- ER websocket is lossless at demo rate (100% delivery at 31 tx/s) and coalesces under
  stress (93.2% at 724 tx/s), which is why the UI counter reads `bid_count` from the
  account payload and never counts websocket events
- Full delegation lifecycle verified end to end on devnet: initialize, delegate, mutate in
  the rollup, commit, undelegate, verify on L1

Those 13,405 transactions are **M0 and M1 benchmark traffic against a counter program,
since closed to reclaim its rent and fund this deploy**. Saying so up front because it is
checkable.

**Two kinds of evidence, and they are not the same.** ER transactions are not on any public
explorer: MagicBlock ships RPC endpoints and a router, not an explorer. ER history is
queryable through the ER RPC itself (verified: `getTransaction` and `getSignaturesForAddress`
both work, and a transaction still resolved 11.2 hours after it was sent), but there is no
permanent public record and the retention window is untested beyond that. So the 13,405
figure is **our measurement**, evidenced by the harness in `bench/` and the captured output
in `bench/logs/`. It is not a citation.

What *is* independently verifiable is the L1 side, and that happens to be the part worth
proving. The delegate / commit / undelegate lifecycle is entirely L1-visible.

## Design decisions worth defending

**Escrow is never delegated.** Funds stay on L1 for the entire auction. Only the job and
the agent registries move into the rollup, and only for the thirty seconds bidding takes.
This is the answer to "is this real or a sidechain toy".

**Settlement is three steps, not one.** The rollup instruction only schedules the commit
and undelegate; the L1 write is executed later by the validator. An ER instruction cannot
touch an undelegated L1 account, so no single instruction can both commit state and pay
escrow. The client polls L1 until ownership returns, then a separate L1 `settle`
instruction reads `best_bidder` and pays out.

**`submit_bid` is 15 lines, three accounts, and cannot return an error.** A losing race is
a landed transaction with an unchanged best bid, so agents need no retry logic and the
counter stays honest. It writes each agent's own delegated registry, which is what lets the
UI show every agent's live bid rather than only the current leader's.

**Delegation is one account per transaction.** Delegating the job plus six registries in a
single instruction overflows the BPF stack (4416 bytes against a 4096 limit) and needs 40
accounts, about 1433 bytes against the 1232 transaction limit. Both were found at compile
time rather than on chain. Splitting across transactions is safe; splitting the validator
identity is not, so a single constant feeds every call.

---

# Closing section — pick one

## Version A — live deployed demo

The program is deployed to Solana devnet at `BjMyKMPtFoWk7wXdSh4iz421H8PFWV1LbWcsJPKCzrhb`.
The video is a single unedited live run: [LIVE: N] bids inside the Ephemeral Rollup in
thirty seconds, settled with one transaction on Solana L1.

Settlement: [LIVE: explorer link]
Settled job account: [LIVE: explorer link]

Both are permanent. The job account is deliberately not closed at settlement so the final
state stays inspectable.

**What is honestly incomplete:**

- One job at a time. No multi-job concurrency, no job board.
- Agents are deterministic bidders with cost curves, not LLM agents. The negotiation is
  real; the intelligence is not the point of this submission.
- Jobs are simulated work. No agent actually renders anything.
- The frontend is built for one desktop resolution and a screen recording.
- I did not get MagicBlock's local ephemeral validator running, so every test cost real
  devnet SOL. The base layer starts fine with an explicit `--gossip-port`; the ER binary
  exits with code 1 and no output, and I timeboxed the investigation rather than sinking
  an evening into it.

## Version B — simulation plus benchmark fallback

**The program is written, builds clean, and is not deployed.** A 330,984 byte Anchor
program needs 2.3045 SOL of rent, plus roughly 21 more rent-exempt accounts for the job,
escrow, six agent registries, and the delegation program's buffer, record and metadata PDAs
for every delegated account. I could not get enough devnet SOL together before the
deadline. The public faucet rate-limited after a single 2 SOL airdrop.

So the demo video runs against a local in-browser simulation of the deployed program, and
it is labelled **mock** on screen for its entire duration. The badge is driven by the same
flag that selects the data source, so it cannot say live when it is not.

**Independently verifiable on the Solana explorer right now.** The M0 counter program used
the same delegation lifecycle as `swarm`. Its PDA has 125 L1 transactions; three prove the
cycle:

| What | Signature |
|---|---|
| `Initialize` on L1 | `4z4SAu5uQpCJ…FADp8` |
| `Delegate` — ownership moves to `DELeGG…` | `3ar1DAAjzTUj…w3pwf4` |
| **`ProcessUndelegation`** — the ER validator writing committed state back to L1 | `3Xqxsz989EFo…C9hSA` |

The third is the one that matters. It is the delegation program executing the commit on L1,
which is the part of the design that could not be assumed. The
[counter account](https://explorer.solana.com/address/Foo2tpFY3zhbpJSJsRA5eDuZGVh2z7xfANW5VD9HKKdd?cluster=devnet)
is still there, owned by the program, holding its final committed value of 13,524. Full
table with clickable links in `bench/RESULTS.md`.

**Measured, evidenced by the harness rather than by explorer links.** Everything ER-side:
13,405 landed transactions with zero failures, the ER versus L1 comparison table above, the
758 tx/s ceiling, the 43ms block rate, the 54-second blockhash window. Raw console output in
`bench/logs/`. These are measurements we made and can show our working for; they are not
things a judge can independently click.

**Runnable.** `bench/` carries the tuned HTTP transport and background blockhash refresher
that make the concurrency numbers meaningful rather than an artifact of Go's default
2-idle-connections-per-host. Its read-only modes still work. Every transaction-sending mode
exits with an explanation that the program was closed and points at the recorded results,
because a harness that fails with an opaque RPC error is worse than useless.

The simulation is not a mockup of a UI. It runs the same bidding logic as the Go agents,
against an in-memory market that mirrors `submit_bid` line for line, and both read the same
tuning file. The Go and TypeScript implementations independently converge to the same final
bid, which is the check that they have not drifted.

**What is honestly incomplete:** the program has never executed on chain. Every claim about
throughput and the delegation lifecycle comes from the counter program in `bench/`, not from
`swarm` itself. I would rather submit that plainly than record a simulation and let it read
as live.

Everything else in Version A's incomplete list also applies.

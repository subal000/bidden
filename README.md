# Bidden

**Agents are bidden. Then they bid.**

An onchain reverse auction where autonomous agents undercut each other in real time for the
right to execute a job. Bidding runs inside a MagicBlock Ephemeral Rollup. The escrow never
leaves Solana L1.

Built for Solana Blitz V7 · theme: Collaboration

---

## The number that makes the case

Agent-to-agent negotiation is a high-frequency workload: hundreds of messages in seconds,
each one a state change the other agents can see and react to.

Same program, same machine, one endpoint swapped. 200 transactions per configuration:

| Configuration | Solana L1 | Ephemeral Rollup |
|---|---|---|
| Sequential, await confirm | 16/200 landed · 0.5 tx/s | 200/200 · 4.2 tx/s |
| Sequential, fire and forget | 20/200 landed · 0.8 tx/s | 200/200 · 8-12 tx/s |
| **6 concurrent + 50-100ms jitter** | **3/200 landed · 0.5 tx/s** | **200/200 · 26-38 tx/s** |

**Concurrency actively hurts on L1.** The exact strategy that produces 26-38 tx/s in the
rollup lands three transactions out of two hundred on devnet, because the burst trips the
public RPC limiter immediately.

To be precise, since this is an infrastructure limit and not a claim about Solana consensus:
public devnet caps `sendTransaction` near 40 per 10s. L1 never dropped anything it accepted
in any run. Paced under the limit it lands 40/40 at a median 498ms confirmation — the 400ms
slot time showing up where it should — plus 5,000 lamports per transaction where the rollup
charges zero.

Method, raw output and the harness: [`bench/RESULTS.md`](bench/RESULTS.md)

## How it works

```
post_job               L1      requester funds an escrow on Solana
delegate               L1      Job + 6 agent registries move into the rollup
submit_bid             Rollup  agents undercut each other, ~1,100 bids in 30s
award_job              Rollup  lowest bid wins
commit_and_undelegate  Rollup  schedules the commit back to L1 (async, ~20s)
settle                 L1      escrow pays the winner, one signature on Solana
```

Six agents, each with its own keypair, cost floor and reputation. Reputation *lowers* an
agent's floor rather than making it bid higher, because the program awards strictly on the
lowest bid and knows nothing about reputation.

### Why MagicBlock is load-bearing, not decorative

Only the hot path is delegated. The `Job` account and each `AgentRegistry` move into the
rollup for the thirty seconds bidding takes. **The escrow is never delegated** — it sits on
Solana L1 for the entire auction. If the rollup vanished mid-auction the funds would be
exactly where they started.

Settlement is three steps, not one, because an ER instruction cannot touch an undelegated L1
account: the rollup schedules the commit, the client polls L1 until ownership returns, then a
separate `settle` instruction pays out.

## Live on devnet

| | |
|---|---|
| Program | [`BjMyKMPtFoWk7wXdSh4iz421H8PFWV1LbWcsJPKCzrhb`](https://explorer.solana.com/address/BjMyKMPtFoWk7wXdSh4iz421H8PFWV1LbWcsJPKCzrhb?cluster=devnet) |
| Integration proof | [`ProcessUndelegation`](https://explorer.solana.com/tx/3EQrQzNPqYXwkV68AThZRHVeYSQth4KnMN8UTC8xKspHeqd3SauhNkJydsM1sWBWW1VeeYDZ9UtFJfBqpkaHMNEe?cluster=devnet) — MagicBlock's delegation program writing committed rollup state back to Solana |
| Settlement | [`Settle`](https://explorer.solana.com/tx/5XscLBpUvrxAFq7DP8yXqdW6oUjogBsspKhRno1JypaBF4BQWBTDJEsGznWhLZbPrefmHNuzk62uTp5PtZ6XnbHV?cluster=devnet) — escrow paying the winning agent |

## Running it

```bash
cd swarm/web
npm install
npm run dev          # open localhost:3000, press "Run job"
```

One button runs the whole lifecycle in about seventy seconds: it creates a fresh job,
delegates seven accounts, runs a thirty second auction, awards, commits back to L1 and
settles. Needs a funded devnet keypair at `~/.config/solana/swarm-deployer.json` and
`L1_RPC` set — see [`.env.example`](.env.example).

Set `LIVE = false` in `swarm/web/app/demo/page.tsx` to run the whole demo against an
in-browser simulation with no network calls at all.

## Repo layout

| Path | What |
|---|---|
| `swarm/programs/swarm` | The Anchor program |
| `swarm/driver` | Go driver for the full lifecycle |
| `swarm/agents` | Six bidding agents with distinct cost curves |
| `swarm/web` | Next.js frontend |
| `bench` | Throughput harness and measured results |
| `m0` | Throwaway counter program used to prove the delegation lifecycle first |

The `swarm/` directory keeps its original name; `Bidden` is the product name.

## Measured, not estimated

- ER block rate **22-23 blocks/s (~43ms)**, identical idle and under 758 tx/s of load. This
  contradicts the 1ms marketing figure, so this project does not use it. Throughput comes
  from packing ~33 transactions per block.
- Sustained **758 tx/s with zero drops**; the ceiling was never found.
- **Zero failed transactions across 13,405 sent.** Concurrent writes to the same PDA
  serialize rather than failing.
- Blockhash window **1,198 blocks (~54s)**, confirmed by probe.
- Undelegate-to-L1 gap **19-21s**, consistent across four runs.

Rollup transactions are on no public explorer, so those throughput figures are our
measurement, evidenced by the harness. The L1 side is independently verifiable.

## Honestly incomplete

- One job at a time. No multi-job concurrency, no job board.
- Agents are deterministic bidders with cost curves, not LLM agents. The negotiation is
  real; the intelligence is not the point.
- Jobs are simulated work. No agent actually renders anything.
- The `Run job` button shells out to Go binaries, so it only works locally. A hosted deploy
  renders live chain state but cannot start a run.
- Mobile layout has an unresolved horizontal overflow. The demo targets one desktop
  resolution.

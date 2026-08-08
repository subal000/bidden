# Bidden — pitch + demo video

One video, both jobs. **Target 2:50.** Hard ceiling 3:00.

The deck carries the pitch, the app carries the demo. Eleven shots, four windows.

---

## Before you press record

| # | Window | URL | Notes |
|---|---|---|---|
| **A** | Browser tab 1 | `localhost:3000/demo` | Badge must read **live · devnet** |
| **B** | Browser tab 2 | `localhost:3000/slides` | Press `F` for full screen |
| **C** | Browser tab 3 | `localhost:3000/benchmark` | Scrolled to the ER vs L1 table |
| **D** | Browser tab 4 | Solana Explorer | Open the settle tx **after** the run |

```bash
cd swarm/web && npm run dev      # .env.local already sets NEXT_PUBLIC_LIVE=true
```

**Do one throwaway run first.** It settles the previous job, warms the Go build cache and
confirms devnet is healthy. You do not want to find a flaky RPC on take three.

Hide the bookmarks bar. Close every other tab. `Cmd+Shift+5`, record a **selection** so the
dock stays out. Hide the cursor except when clicking.

Each run costs ~0.023 SOL. At 3.2 SOL you have roughly 140 takes.

## Number discipline

- `[LIVE: ...]` — read off the take you record. **Nothing from the mock fills one.**
- `[MEASURED: ...]` — from the benchmark harness. Speak it as benchmark measurement.

**Never say 1ms.** Measured block rate is 22-23 blocks/s, about 43ms.
**Never say rollup transactions are on the explorer.** Only the L1 side is.

---

# The script

### 1 · 0:00-0:12 · **WINDOW A** — the hook

Auction already mid-flight. Counter climbing, curve falling, cards pulsing. No cursor.
Start recording once the counter has passed a few hundred so it moves from frame one.

> Six autonomous agents are bidding against each other for a job. Right now, on Solana.
>
> Every one of these is a real transaction. [LIVE: N] of them in the last ten seconds.
>
> On Solana L1, that same negotiation lands three out of two hundred.

The pitch lands at twelve seconds. Everything after is evidence.

---

### ▶ **SWITCH TO WINDOW B** · slides, full screen · `?s=2`

### 2 · 0:12-0:32 · **SLIDE 2** — the problem

> Agents that work together have to agree a price. Verifiably, so nobody has to be trusted.
>
> But negotiation is chatty. Hundreds of messages in seconds, each one a state change the
> others react to. At four hundred millisecond slots with a fee per message, that is
> economically dead on a base layer.
>
> So every agent marketplace today runs its matching off-chain and asks you to trust it.

### → **ARROW** to slide 3

### 3 · 0:32-0:50 · **SLIDE 3** — the product

> Bidden is a live reverse auction, settled on Solana.
>
> A requester posts a job and funds an escrow. Six agents are summoned to it, each with its
> own keypair, cost floor and reputation. They undercut each other for thirty seconds.
>
> Lowest bid wins, and the winner is paid from escrow. There is no matching engine to trust.
> The auction is the chain.

### → **ARROW** to slide 4

### 4 · 0:50-1:08 · **SLIDE 4** — why MagicBlock

> Here is the split that makes it work.
>
> The job and the six agent registries delegate into a MagicBlock Ephemeral Rollup for the
> thirty seconds bidding takes. Forty three millisecond blocks, zero fees.
>
> The escrow is never delegated. It sits on Solana the entire time. If the rollup vanished
> mid-auction, the money would be exactly where it started. That is the difference between
> using a rollup and trusting one.

### → **ARROW** twice, to slide 6 (skip the Proof slide, the real page is better)

### 5 · 1:08-1:12 · **SLIDE 6** — demo divider

Hold two seconds, no narration. Then cut.

---

### ▶ **SWITCH TO WINDOW A** — the live run

### 6 · 1:12-1:44 · the auction

Cursor visible, click **Run job**. Then hide the cursor. Leave silence in the middle and let
the motion carry it.

> Here is one auction, start to finish.
>
> Seven accounts delegate into the rollup, and six agents start undercutting each other.
>
> *(pause ~8 seconds, let the counter climb and the curve fall)*
>
> [LIVE: N] bids in thirty seconds. Every one a real transaction, and none of them cost
> anything.

---

### ▶ **SWITCH TO WINDOW C** — the measurement

### 7 · 1:44-2:08 · the proof

Scroll slowly so the `3 / 200` row lands centre frame and rests there. Your strongest
content. Do not rush it.

> I measured why this needs a rollup instead of assuming it.
>
> Same program. Same machine. One endpoint changed.
>
> On Solana devnet, six concurrent bidders land three transactions out of two hundred.
> Concurrency actively hurts, because the burst trips the rate limiter.
>
> Inside the rollup, the identical configuration lands two hundred out of two hundred. I
> pushed the harness to [MEASURED: 758] transactions per second before I stopped looking for
> the ceiling.

---

### ▶ **SWITCH TO WINDOW A** — the commit

### 8 · 2:08-2:16 · committing back

Show the lifecycle panel with `Poll L1` spinning.

> Bidding closes, and the rollup schedules a commit back to Solana. That is asynchronous and
> takes about twenty seconds.

### ✂ **HARD CUT** — caption `⏱ 20s later`, bottom left, 2 seconds

### ▶ **SWITCH TO WINDOW D** — Solana Explorer

### 9 · 2:16-2:28 · settlement

Scroll to the balance changes so escrow going to zero and the winner going up are both
visible.

> And here it lands. One transaction on Solana. The escrow pays the winning agent directly:
> [LIVE: payout] SOL to the agent that bid [LIVE: bid] percent.

---

### ▶ **SWITCH TO WINDOW B** · `?s=7`

### 10 · 2:28-2:44 · **SLIDE 7 then 8** — where it scales, and the model

Advance to slide 8 on "sell the mechanism".

> Strip the agent framing away and this is competitive price discovery with trustless
> settlement. That primitive already has buyers.
>
> Solver auctions in DeFi run off-chain today because onchain was too slow. It no longer is.
> Compute markets price GPUs off-chain and settle on it.
>
> **→ arrow**
>
> So we sell the mechanism before the marketplace. A per-auction fee to protocols that
> already have supply and demand, and a take rate on settled volume once it clears at scale.

### → **ARROW** to slide 9

### 11 · 2:44-2:52 · close

Hold three full seconds after the last word before stopping.

> Agents cannot afford to negotiate on a base layer. Inside an Ephemeral Rollup it costs
> nothing, so they can argue for as long as they need to.
>
> Agents are bidden. Then they bid.

---

## Switch summary

```
0:00   A   demo          hook, auction mid-flight
0:12   B   slide 2       the problem
0:32   B   slide 3   →   the product
0:50   B   slide 4   →   why MagicBlock
1:08   B   slide 6  →→   demo divider, hold 2s
1:12   A   demo          press Run job, live auction
1:44   C   benchmark     the 3/200 table
2:08   A   demo          commit, then ✂ CUT
2:16   D   explorer      settlement
2:28   B   slide 7   →   scale, then slide 8 for the model
2:44   B   slide 9   →   close
```

Four windows, six arrow presses. **Rehearse the switches once without recording.**

## The deck

`localhost:3000/slides` · `F` full screen · arrows advance · `?s=N` deep-links a slide

| # | Slide | In video |
|---|---|---|
| 1 | **bidden** — title | skipped, the hook is stronger |
| 2 | Agents can't haggle onchain | 0:12 |
| 3 | A live reverse auction, settled on Solana | 0:32 |
| 4 | Negotiation moves. Money does not. | 0:50 |
| 5 | We measured it instead of assuming it | skipped, `/benchmark` is better on camera |
| 6 | Demo | 1:08 |
| 7 | The auction is the product, not the agents | 2:28 |
| 8 | Sell the mechanism before the marketplace | 2:36 |
| 9 | **bidden** — close | 2:44 |

Slides 1 and 5 exist so the deck stands alone if you ever present it without a screen share.

## Before export

- [ ] `[LIVE: N]` shot 1 — bids in the first ten seconds
- [ ] `[LIVE: N]` shot 6 — final bid count
- [ ] `[LIVE: payout]` and `[LIVE: bid]` shot 9 — from the result panel
- [ ] Badge reads **live · devnet** in every app shot
- [ ] No frame contains the word `mock`
- [ ] Settlement link resolves on Solana Explorer
- [ ] Nothing claims 1ms blocks
- [ ] Nothing claims rollup transactions are on the explorer

## Upload

YouTube, public. Title: `Bidden — agents are bidden, then they bid | Solana Blitz V7`
That URL goes in **Pitch & Demo**. The Vercel URL goes in **Project website**.

## If devnet degrades mid-recording

Stop the take. Do not finish with the mock and reconcile later. If the final video must use
the simulation, shot 1 changes to "this is a local simulation of the deployed program" and
the benchmark section carries the weight.

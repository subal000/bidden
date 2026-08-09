# Bidden — pitch + demo video

One video, both jobs. **Target 2:50.** Hard ceiling 3:00.

The deck carries the pitch, the app carries the demo. Twelve shots, four windows.

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

### 2 · 0:12-0:22 · **SLIDE 2** — "Two agents want to agree on a price"

> Two agents that work together have to agree on a price. And prove to everyone else that
> they did, or you are back to trusting somebody.

### → **ARROW** to slide 3

### 3 · 0:22-0:36 · **SLIDE 3** — the 1,100 number

Let the number sit for a beat before you speak.

> A real negotiation is chatty. Bids, counter-bids, hundreds of them in half a minute.
>
> Try that on Solana and three land. The rest are rejected before they reach consensus.

### → **ARROW** to slide 4

### 4 · 0:36-0:54 · **SLIDE 4** — the product, as a flow

> Bidden is a reverse auction that lives on chain.
>
> A requester posts a job and funds an escrow. Six agents are summoned to it, each with its
> own keypair and its own cost floor. They undercut each other for thirty seconds, the lowest
> bid wins, and the escrow pays out.
>
> No matching engine in the middle to trust.

### → **ARROW** to slide 5

### 5 · 0:54-1:12 · **SLIDE 5** — why MagicBlock

> Here is the trick.
>
> The job and the six agent registries delegate into a MagicBlock Ephemeral Rollup for the
> thirty seconds bidding takes. Forty three millisecond blocks, and every bid is free.
>
> The escrow never moves. It sits on Solana the whole time. If the rollup vanished
> mid-auction, the money would be exactly where it started.

### → **ARROW** twice, to slide 7 (skip Proof, the real page is better on camera)

### 6 · 1:12-1:16 · **SLIDE 7** — "Demo"

Hold two seconds. No narration. Then cut.

---

### ▶ **SWITCH TO WINDOW A** — the live run

### 7 · 1:16-1:48 · the auction

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

### 8 · 1:48-2:12 · the proof

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

### 9 · 2:12-2:20 · committing back

Show the lifecycle panel with `Poll L1` spinning.

> Bidding closes, and the rollup schedules a commit back to Solana. That is asynchronous and
> takes about twenty seconds.

### ✂ **HARD CUT** — caption `⏱ 20s later`, bottom left, 2 seconds

### ▶ **SWITCH TO WINDOW D** — Solana Explorer

### 10 · 2:20-2:32 · settlement

Scroll to the balance changes so escrow going to zero and the winner going up are both
visible.

> And here it lands. One transaction on Solana. The escrow pays the winning agent directly:
> [LIVE: payout] SOL to the agent that bid [LIVE: bid] percent.

---

### ▶ **SWITCH TO WINDOW B** · `?s=8`

### 11 · 2:32-2:48 · **SLIDES 8 → 9** — where it scales, and the model

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

### → **ARROW** to slide 10

### 12 · 2:48-2:56 · close

Hold three full seconds after the last word before stopping.

> Agents cannot afford to negotiate on a base layer. Inside an Ephemeral Rollup it costs
> nothing, so they can argue for as long as they need to.
>
> Agents are bidden. Then they bid.

---

## Switch summary

```
0:00   A   demo          hook, auction mid-flight
0:12   B   slide 2       two agents want a price
0:22   B   slide 3   →   1,100 messages, three land
0:36   B   slide 4   →   the product, as a flow
0:54   B   slide 5   →   why MagicBlock, the diagram
1:12   B   slide 7  →→   "Demo", hold 2s
1:16   A   demo          press Run job, live auction
1:48   C   benchmark     the 3/200 table
2:12   A   demo          commit, then ✂ CUT
2:20   D   explorer      settlement
2:32   B   slide 8   →   scale, then slide 9 for the model
2:48   B   slide 10  →   close
```

Four windows, six arrow presses. **Rehearse the switches once without recording.**

## The deck

`localhost:3000/slides` · `F` full screen · arrows advance · `?s=N` deep-links a slide

| # | Slide | In video |
|---|---|---|
| 1 | **bidden** — wordmark | skipped, the hook is stronger |
| 2 | Two agents want to agree on a price | 0:12 |
| 3 | **1,100** messages · on Solana, three land | 0:22 |
| 4 | A reverse auction that lives on chain | 0:32 |
| 5 | The talking moves. The money stays. | 0:50 |
| 6 | **3** vs **200** of 200 landed | skipped, `/benchmark` is better on camera |
| 7 | **Demo** | 1:08 |
| 8 | This already happens. Just not on chain. | 2:28 |
| 9 | Sell the auction. Not the marketplace. | 2:36 |
| 10 | **bidden** — close | 2:44 |

Slides 1 and 6 exist so the deck stands alone if you ever present it without a screen share.

The rhythm is deliberate: a sentence, then a number, then a flow, then a diagram, then one
word. Most slides carry almost no text because the narration is doing the talking.

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

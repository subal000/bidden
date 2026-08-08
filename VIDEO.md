# Bidden — pitch + demo video

One video, both jobs. **Target 2:15.** Hard ceiling 2:30.

---

## Before you press record

**Windows to have open, in this order.** Switching is `Cmd+~` (same app) or `Cmd+Tab`.

| # | Window | URL | Notes |
|---|---|---|---|
| A | Browser tab 1 | `localhost:3000/demo` | The live auction. Badge must read **live · devnet** |
| B | Browser tab 2 | `localhost:3000/slides` | Press `F` for full screen, arrows to advance |
| C | Browser tab 3 | `localhost:3000/benchmark` | Scrolled to the ER vs L1 table |
| D | Browser tab 4 | Solana Explorer | Opened on the settle tx **after** the run |
| E | Browser tab 5 | your Vercel URL | One 4-second shot only |

```bash
cd swarm/web && npm run dev      # .env.local already sets NEXT_PUBLIC_LIVE=true
```

**Do one throwaway run first.** It settles the previous job, warms the Go build cache, and
confirms devnet is healthy. You do not want to discover a flaky RPC on take three.

Hide the bookmarks bar. Close every other tab. `Cmd+Shift+5`, record a **selection** so the
dock and menu bar stay out. Hide the cursor except when you are clicking something.

Each run costs ~0.023 SOL. At 3.2 SOL you have roughly 140 takes.

---

## Number discipline

- `[LIVE: ...]` — read off the take you actually record. **Nothing from the mock fills one.**
- `[MEASURED: ...]` — from the benchmark harness, already known, safe to pre-fill. Speak it
  as benchmark measurement, not as this run.

**Never say 1ms.** Measured ER block rate is 22-23 blocks/s, about 43ms.
**Never say rollup transactions are on the explorer.** Only the L1 side is.

---

## The script

### Shot 1 · 0:00-0:12 · WINDOW A (localhost/demo)

**Show:** the auction already mid-flight. Counter climbing, price curve falling, agent cards
pulsing. No cursor, no clicking. Start recording *after* the counter has passed a few
hundred so it is visibly moving from frame one.

> Six autonomous agents are bidding against each other for a job. Right now, on Solana.
>
> Every one of these is a real transaction. [LIVE: N] of them in the last ten seconds.
>
> On Solana L1, that same negotiation lands three transactions out of two hundred.

That last line is the whole pitch and it arrives at twelve seconds. Everything after is
evidence.

---

### ▶ SWITCH TO WINDOW B — slides, full screen, slide 1

### Shot 2 · 0:12-0:34 · SLIDE 1 "Agents have to negotiate"

**Show:** hold the slide still. Do not advance mid-sentence.

> Agents that work together have to negotiate. Price, terms, who does the work.
>
> For that to be trustworthy it has to happen onchain, where anyone can verify it.
>
> But negotiation is chatty. Hundreds of messages in seconds. At four hundred millisecond
> slots with a fee per message, that is economically dead on a base layer.
>
> So today agents either do not negotiate at all, or they do it off-chain and ask you to
> trust the result.

### → PRESS RIGHT ARROW on the last word

### Shot 3 · 0:34-0:52 · SLIDE 2 "Negotiation moves. Money does not."

> This is Bidden.
>
> A job is posted with escrow. Six agents are summoned to it, each with its own keypair and
> its own cost floor.
>
> The job and the agent registries delegate into a MagicBlock Ephemeral Rollup. That is
> where bidding happens, at about forty three millisecond blocks and zero fees.
>
> The escrow is never delegated. It sits on Solana the entire time.

---

### ▶ SWITCH TO WINDOW A (localhost/demo)

### Shot 4 · 0:52-1:24 · LIVE RUN

**Show:** cursor visible, click **Run job**. Then hide the cursor and let it run. Let the
counter climb and the curve fall. Do not narrate over the whole thirty seconds; leave
silence in the middle so the motion carries it.

> Here is one auction, start to finish. I press run.
>
> Seven accounts delegate into the rollup, and six agents start undercutting each other.
>
> *(pause ~6 seconds, let it climb)*
>
> [LIVE: N] bids in thirty seconds. Every one a real transaction. Watch the price fall.

---

### ▶ SWITCH TO WINDOW C (localhost/benchmark)

### Shot 5 · 1:24-1:50 · THE MEASUREMENT

**Show:** the ER vs L1 table. Slowly scroll so the `3 / 200` row lands centre frame and
rest there. This is your strongest content. Do not rush it.

> I measured why this needs a rollup, instead of assuming it.
>
> Same program. Same machine. One endpoint changed.
>
> Against Solana devnet, six concurrent bidders land three transactions out of two hundred.
> Concurrency actively hurts, because the burst trips the rate limiter.
>
> Inside the rollup, the identical configuration lands two hundred out of two hundred. I
> pushed the harness to [MEASURED: 758] transactions per second before I stopped looking for
> the ceiling.

---

### ▶ SWITCH TO WINDOW A (localhost/demo)

### Shot 6 · 1:50-1:58 · THE COMMIT

**Show:** the lifecycle panel with `Poll L1` active and spinning.

> Bidding closes, and the rollup schedules a commit back to Solana. That part is
> asynchronous and takes about twenty seconds.

### ✂ HARD CUT — caption `⏱ 20s later` bottom left, 2 seconds

---

### ▶ SWITCH TO WINDOW D (Solana Explorer)

### Shot 7 · 1:58-2:10 · SETTLEMENT

**Show:** the settle transaction. Scroll to the balance changes so escrow going to zero and
the winner going up are both visible.

> And here it lands. One transaction on Solana. The escrow pays the winning agent directly:
> [LIVE: payout] SOL to the agent that bid [LIVE: bid] percent.

---

### ▶ SWITCH TO WINDOW E (Vercel URL) — 4 seconds only

### Shot 8 · 2:10-2:14 · IT IS LIVE

**Show:** the deployed site with the URL bar visible. Do not interact, just let the URL be
readable.

> It is deployed, and the page reads the auction live from the rollup.

---

### ▶ SWITCH TO WINDOW B — press RIGHT ARROW to slide 3

### Shot 9 · 2:14-2:25 · CLOSE

**Show:** the closing slide. Hold three full seconds after the last word before you stop
recording.

> Agents cannot afford to negotiate on a base layer. Inside an Ephemeral Rollup it costs
> nothing, so they can argue for as long as they need to.
>
> Agents are bidden. Then they bid.

---

## Switch summary

```
0:00  A  localhost/demo       auction mid-flight
0:12  B  slides 1             the problem
0:34  B  slides 2  (→ arrow)  how it works
0:52  A  localhost/demo       press Run job, live auction
1:24  C  localhost/benchmark  the 3/200 table
1:50  A  localhost/demo       commit, then CUT
1:58  D  explorer             settlement
2:10  E  vercel URL           4 seconds
2:14  B  slides 3  (→ arrow)  close
```

Nine shots, five windows, two arrow presses. Rehearse the switches once without recording.

## Slides

Open `localhost:3000/slides`, press `F` for full screen. Arrow keys advance.
`?s=2` jumps straight to a slide, so you can restart a take mid-way.

1. **Agents have to negotiate** — the problem
2. **Negotiation moves. Money does not.** — L1 vs rollup split
3. **bidden** — tagline, headline numbers, repo

## Before export

- [ ] `[LIVE: N]` shot 1 — bids in the first ten seconds
- [ ] `[LIVE: N]` shot 4 — final bid count
- [ ] `[LIVE: payout]` and `[LIVE: bid]` shot 7 — from the result panel
- [ ] Badge reads **live · devnet** in every localhost shot
- [ ] No frame contains the word `mock`
- [ ] Settlement link resolves on Solana Explorer
- [ ] Nothing claims 1ms blocks
- [ ] Nothing claims rollup transactions are on the explorer

## Upload

YouTube, public. Title: `Bidden — agents are bidden, then they bid | Solana Blitz V7`.
Paste that URL into the submission's **Pitch & Demo** field. The Vercel URL goes in
**Project website**.

## If devnet degrades mid-recording

Stop the take. Do not finish with the mock and reconcile later. If the final video has to
use the simulation, shot 1 changes to "this is a local simulation of the deployed program"
and the benchmark section carries the weight.

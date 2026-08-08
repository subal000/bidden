# Bidden — pitch + demo video

One video, both jobs. Target **2:15**. Hard ceiling 2:30.

## Number discipline

- `[LIVE: ...]` — read off the take you actually record. **Nothing from the mock fills one.**
- `[MEASURED: ...]` — from the benchmark harness, already known, safe to pre-fill. Must be
  spoken as benchmark measurement, not as this run.

**Never say 1ms.** Measured ER block rate is 22-23 blocks/s, about 43ms.

**Never say rollup transactions are on the explorer.** They are not. Only the L1 side is.

---

## Shot list

| # | Time | On screen | Caption |
|---|---|---|---|
| 1 | 0:00-0:12 | `/demo` mid-auction. Counter climbing, curve falling, cards pulsing. No cursor. | none |
| 2 | 0:12-0:32 | `/` overview, slow scroll past the hero | none |
| 3 | 0:32-0:50 | Agent cards, one expanded showing floor and reputation | `6 agents · own keypair · own cost floor` |
| 4 | 0:50-1:20 | Press Run job. Full auction, counter 0 to [LIVE: N] | `escrow never leaves Solana L1` |
| 5 | 1:20-1:45 | `/benchmark`, the 3/200 vs 200/200 table | `same program · same machine · one endpoint` |
| 6 | 1:45-2:05 | Commit, hard cut, then Solana Explorer on the settle tx | `⏱ 20s later` |
| 7 | 2:05-2:20 | Final frame: result panel beside the explorer. Hold 3s. | `[LIVE: N] bids · 1 settlement` |

---

## Narration

Read at a normal pace. Roughly 340 words, which is about 2:15.

**Shot 1 — the hook. No setup, no logo, no "hi I'm".**

> Six autonomous agents are bidding against each other for a job. Right now, on Solana.
>
> Every one of these is a real transaction. [LIVE: N] of them in the last ten seconds.
>
> On Solana L1, that same negotiation lands three transactions out of two hundred.

That last line is the whole pitch and it arrives at twelve seconds. Everything after is
evidence.

**Shot 2 — the problem. This is the pitch half.**

> Agents that work together have to negotiate. Price, terms, who does what.
>
> For that to be trustworthy it has to happen onchain, where anyone can verify it.
>
> But negotiation is chatty. Hundreds of messages in seconds. At four hundred millisecond
> slots with a fee per message, that is economically dead on a base layer.
>
> So today agents either do not negotiate at all, or they do it off-chain and ask you to
> trust the result.

**Shot 3 — what it is.**

> This is Bidden.
>
> A job is posted with escrow. Six agents are summoned to it. Each has its own keypair, its
> own cost floor, its own idea of what the work is worth.
>
> They undercut each other for thirty seconds. The lowest bid wins.

**Shot 4 — the demo. Press Run job here.**

> Here is one auction, start to finish.
>
> The job and the six agent registries delegate into a MagicBlock Ephemeral Rollup. That is
> where bidding happens. Blocks land in about forty three milliseconds, and transactions
> cost nothing.
>
> The escrow is never delegated. It sits on Solana the entire time. If the rollup vanished
> mid-auction, the money would be exactly where it started.
>
> [LIVE: N] bids in thirty seconds. Watch the price fall.

**Shot 5 — the measurement. Your strongest technical content. Do not rush it.**

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

Optional, if pacing allows. Worth saying because it is checkable:

> Those numbers come from a counter program I deployed and then closed to fund this one.
> Thirteen thousand four hundred and five landed transactions, zero failures.

**Shot 6 — settlement. Cut here.**

> Bidding closes, and the rollup schedules a commit back to Solana. That part is
> asynchronous and takes about twenty seconds.

`[HARD CUT]` to the explorer, caption `⏱ 20s later` bottom left for two seconds.

> And here it lands. One transaction on Solana. The escrow pays the winning agent directly:
> [LIVE: payout] SOL to the agent that bid [LIVE: bid] percent.

**Shot 7 — close.**

> Agents cannot afford to negotiate on a base layer. Inside an Ephemeral Rollup it costs
> nothing, so they can argue for as long as they need to.
>
> Agents are bidden. Then they bid.

Hold the final frame three full seconds.

---

## Recording

Use **localhost**, not the deployed site. Only localhost can drive a real auction.

```bash
cd swarm/web && npm run dev     # .env.local already sets NEXT_PUBLIC_LIVE=true
```

- Do one throwaway run before recording. It settles the previous job and confirms devnet is
  healthy. You do not want to find a flaky RPC on take three.
- Hide the bookmarks bar, close other tabs, full screen. `Cmd+Shift+5`, record a selection
  so the dock stays out.
- For the first ~8 seconds after pressing Run job the page still shows the previous
  auction while Prepare runs. Start recording once the counter resets to zero, or trim it.
- Record the auction first, while devnet is behaving. The benchmark and explorer shots do
  not depend on a good run and can be captured any time.
- Each run costs about 0.023 SOL. At 3.2 SOL you have roughly 140 takes.

## Before export

- [ ] `[LIVE: N]` shot 1 — bids in the first ten seconds
- [ ] `[LIVE: N]` shots 4, 7 — final bid count
- [ ] `[LIVE: payout]` and `[LIVE: bid]` shot 6 — from the result panel
- [ ] Badge reads **live · devnet** in every frontend shot
- [ ] No frame contains the word `mock`
- [ ] Settlement link opens and resolves on Solana Explorer
- [ ] Nothing claims 1ms blocks
- [ ] Nothing claims rollup transactions are on the explorer

## If devnet degrades mid-recording

Stop the take. Do not finish with the mock and reconcile later. If the final video has to
use the simulation, shot 1 changes to "this is a local simulation of the deployed program"
and the benchmark section carries the weight.

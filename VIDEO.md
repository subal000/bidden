# swarm — 90 second demo video

## Number discipline

Two marker types. Nothing ships with a marker still in it.

- `[LIVE: ...]` — must be filled from the recorded run in Phase 5. **Nothing from the mock
  may ever fill one of these.**
- `[MEASURED: ...]` — from the M1 benchmark harness against the M0 counter program on
  devnet. Already known, safe to pre-fill, but must be described as benchmark measurement
  and not as this run.

Never claim 1ms block time. Measured ER block rate is 22-23 blocks/s, about 43ms.

---

## Shot list

| # | Time | Visual | On-screen text |
|---|---|---|---|
| 1 | 0:00-0:15 | Full frontend, mid-run. Counter climbing, curve descending, cards pulsing. No cursor, no chrome. | none |
| 2 | 0:15-0:30 | Slow zoom to the six agent cards. Bids visibly differ, winner badge moves between them. | `6 agents · own keypair · own cost curve` |
| 3 | 0:30-0:42 | Cut to terminal. `bench --mode suite` output, ER vs L1 table. Highlight the `3/200` row. | `same program · same machine · one endpoint swap` |
| 4 | 0:42-0:55 | Back to frontend. Convergence curve completing, status flips to Awarded. | `escrow never left Solana L1` |
| 5 | 0:55-1:08 | The undelegate gap. **Two variants below.** | `committing to Solana L1` |
| 6 | 1:08-1:20 | Solana explorer, the settle transaction. Escrow drained, winner balance up, Job status Settled. | `one L1 signature` |
| 7 | 1:20-1:30 | Frontend final frame: counter at final value beside the explorer link. Hold 3s. | `[LIVE: final bid count] bids · 1 settlement` |

---

## Narration

**Shot 1 — 0:00 to 0:15. The hook. No setup, no preamble.**

> Six autonomous agents are bidding against each other for a job, right now.
> Every one of these is a real Solana transaction.
> [LIVE: N] of them in the last ten seconds.
>
> On Solana L1, that same negotiation lands three transactions out of two hundred.

That last line is the whole pitch and it arrives at 12 seconds. Everything after is
evidence.

**Shot 2 — 0:15 to 0:30. What you are watching.**

> Each agent has its own keypair, its own cost curve, and its own idea of what this job is
> worth. They undercut each other, hold when they are winning, and drop out when the price
> goes below their floor.
>
> The price converges over thirty seconds because that is what a market looks like. Nobody
> is scripting this.

**Shot 3 — 0:30 to 0:42. The technical core. Give it the screen time.**

> This is the same program, on the same machine, with one endpoint changed.
>
> Against Solana devnet, six concurrent bidders land three transactions out of two hundred.
> Concurrency actively hurts, because the burst trips the RPC rate limiter.
>
> Inside the Ephemeral Rollup, the identical configuration lands two hundred out of two
> hundred, at [MEASURED: 26-38] per second. Zero failures, zero fees.
>
> I pushed the harness to [MEASURED: 758] transactions per second before I stopped looking
> for the ceiling.

Optional, if the pacing allows, and worth saying because a judge will check:

> Those benchmark numbers come from a counter program I deployed to devnet and then closed
> to fund this deploy. Thirteen thousand four hundred and five landed transactions, zero
> failures, across two milestones.

Do **not** append "the signatures are still on the explorer". Rollup transactions are not on
any public explorer, so that would be false. If you want to point at something clickable,
point at the L1 side instead:

> The delegate and commit back to Solana are on the explorer. The rollup throughput is my
> measurement, and the harness is in the repo.

**Shot 4 — 0:42 to 0:55. Why this is not a sidechain toy.**

> The negotiation is fast and free because it happens off L1. The money never does.
>
> The escrow account is never delegated. It sits on Solana for the entire auction. Only the
> job and the agent registries move into the rollup, and only for the thirty seconds the
> bidding takes.

**Shot 5 — 0:55 to 1:08. The commit.**

**MEASURED: the undelegate-to-L1 gap is 20 seconds.** (19.973s on job 5, all seven
accounts flipping at once, zero read failures.) That is 22% of a 90 second video, so
**Variant B is the one you use.** Variant A is kept only in case a future run is far
faster; do not hold 20 seconds of a stalled screen.

*Variant A — ONLY if a future run measures under about 4 seconds. Hold, do not cut.*

> Bidding closes. The rollup schedules a commit back to Solana.
>
> This part is asynchronous. The rollup registers the intent, and the validator executes
> the write on L1 a few seconds later. There is no synchronous path, so the client polls
> until ownership of all seven accounts returns to the program.

On screen: the `poll` output ticking `3/7 → 7/7`, counter frozen at its final value.
The polling output is the visual. It fills the gap honestly.

*Variant B — USE THIS. Measured gap 20s. Cut.*

Hard cut from "schedules a commit back to Solana" straight to the explorer in shot 6, with
a caption `⏱ 20s later` bottom-left for two seconds.

> Bidding closes, and the rollup schedules a commit back to Solana. That settlement is
> asynchronous and takes about twenty seconds, so here is where it lands.

Saying the number out loud is what makes the cut honest. It also happens to be a real
property of the system worth stating: the rollup is fast, committing back to L1 is not,
and pretending otherwise is the kind of thing a MagicBlock engineer would catch instantly.

Do not pretend the gap is not there. The caption is what keeps the cut honest.

**Shot 6 — 1:08 to 1:20. The payoff.**

> One transaction on Solana. The escrow pays the winning agent directly, the job is marked
> settled, and the winner's reputation account records what it earned.
>
> [LIVE: N] bids inside the rollup. One signature on L1.

**Shot 7 — 1:20 to 1:30. Close.**

> Agents cannot afford to negotiate on L1. Four hundred millisecond slots and a fee per
> message make it economically dead.
>
> Inside an Ephemeral Rollup it costs nothing, so they can argue as long as they need to.
> That is the whole project.

Hold the final frame for three full seconds. The counter beside the single signature is the
image the judge should be left with.

---

## Fill-in checklist before export

- [ ] `[LIVE: N]` in shot 1 narration — bids in the first ten seconds of the recorded run
- [ ] `[LIVE: N]` in shot 6 and 7 — final `Job.bid_count` from the recorded run
- [x] shot 5 gap — MEASURED at 20s (19.973s). Variant B, caption reads `⏱ 20s later`
- [ ] `[LIVE: final bid count]` in shot 7 on-screen text
- [ ] `[MEASURED: ...]` values match the numbers in CLAUDE.md
- [ ] Badge reads **live** in every frontend shot
- [ ] Explorer link resolves and the Job account still exists
- [ ] No frame contains the word `mock`

---

## Recording notes

- One desktop resolution, 1680x1050. The layout is fixed and nothing scrolls.
- Hide the cursor during shots 1, 2, 4, 7.
- The convergence curve is the most watchable element. Do not cut away from it during
  its steep phase, roughly the first eight seconds of a run.
- If devnet degrades mid-recording, stop. Do not finish the take with the mock and
  reconcile later. Set `LIVE = false`, and if the final video uses the simulation then the
  narration in shot 1 changes to "this is a local simulation of the deployed program" and
  the benchmark section carries the weight. See the fallback closing in SUBMISSION.md.

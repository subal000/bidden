# Bidden — video script

**~2:50.** Twelve shots. Four browser tabs, nothing else open.

Every shot below has three lines: **SCREEN** is what the viewer sees, **YOU DO** is your
hands, **YOU SAY** is word for word. Read only the bold line you need mid-take.

---

## Setup, once

```bash
cd swarm/web && npm run dev
```

Four tabs, in this order:

| Tab | URL |
|---|---|
| **1** | `localhost:3000/demo` — badge must read **live · devnet** |
| **2** | `localhost:3000/slides?s=2` — press `F` for full screen |
| **3** | `localhost:3000/benchmark` — scrolled so the table fills the frame |
| **4** | Solana Explorer — you'll open the settle tx after the run |

Then: **do one full throwaway run.** It settles the old job, warms the Go cache, and proves
devnet is healthy. Never find a bad RPC on take three.

`Cmd+Shift+5` → record a **selection**, not the screen, so the dock stays out. Hide the
bookmarks bar. Cursor hidden except when you click.

Switching tabs is `Cmd+1` … `Cmd+4`. Practise the sequence once before you record.

## Never say

- **"1ms blocks."** It's 22-23 blocks/s, about 43ms. You measured it.
- **"The signatures are on the explorer"** about rollup transactions. Only the L1 ones are.

---

# SHOT 1 — the hook · 0:00

**SCREEN** Tab 1. An auction already running. Counter climbing, price falling, cards pulsing.

**YOU DO** Nothing. Start recording once the counter is past ~300 so it's moving in the
first frame. No cursor.

**YOU SAY**
> These are six AI agents, and they're bidding against each other for a job. Right now, on
> Solana.
>
> Every one of those is a real transaction. That's [LIVE: N] in the last ten seconds.
>
> Try the same thing on Solana directly, and three of them land.

---

# SHOT 2 — the setup · 0:12

**SCREEN** Tab 2, slide 2. *Two agents need to agree on a price.*

**YOU DO** `Cmd+2`. Wait one beat before speaking.

**YOU SAY**
> Here's the situation. Two agents need to agree on a price. One wants work done, the other
> wants to do it.
>
> And they need to prove they agreed. Otherwise you're just trusting somebody again.

---

# SHOT 3 — what it costs · 0:22

**SCREEN** Slide 3. **1,100** in huge type.

**YOU DO** Press `→` once. **Let the number sit for two full seconds before you speak.**

**YOU SAY**
> But agreeing isn't one message. It's hundreds. Bid, counter-bid, back and forth, for half
> a minute.
>
> On Solana, three of those land. The rest get rejected before they ever reach consensus.

---

# SHOT 4 — the product · 0:36

**SCREEN** Slide 4. The four-step flow.

**YOU DO** Press `→`.

**YOU SAY**
> So we built Bidden.
>
> You post a job and fund an escrow. Six agents show up and start undercutting each other.
> Thirty seconds later the lowest bid wins, and the escrow pays out.
>
> Nobody's running a matching engine. It all happens on chain.

---

# SHOT 5 — why MagicBlock · 0:54

**SCREEN** Slide 5. The two-lane diagram.

**YOU DO** Press `→`.

**YOU SAY**
> Here's the trick.
>
> The job and the six agents move into a MagicBlock rollup for those thirty seconds. Blocks
> land in about forty milliseconds, and every bid is free.
>
> The escrow doesn't move. It stays on Solana the whole time. If the rollup disappeared
> mid-auction, the money would still be sitting right there.

---

# SHOT 6 — the divider · 1:12

**SCREEN** Slide 7. One word: **Demo**.

**YOU DO** Press `→` twice (skipping the numbers slide). Hold two seconds. Say nothing.

---

# SHOT 7 — the live run · 1:16

**SCREEN** Tab 1.

**YOU DO** `Cmd+1`. Cursor visible, click **Run job**, then hide the cursor. Let it run.

**YOU SAY**
> Let me just run one.
>
> Seven accounts delegate into the rollup, and then the agents start.

**PAUSE ~8 SECONDS.** Say nothing. Let the counter climb and the curve fall. This silence is
doing more work than any sentence would.

**YOU SAY**
> [LIVE: N] bids, in thirty seconds. None of them cost anything.

---

# SHOT 8 — the proof · 1:48

**SCREEN** Tab 3. The ER vs L1 table.

**YOU DO** `Cmd+3`. Scroll slowly so the `3 / 200` row lands centre frame, then stop. Leave
it there for the whole shot.

**YOU SAY**
> I wanted to know whether the rollup was actually necessary, so I measured it.
>
> Same code. Same laptop. I changed one endpoint.
>
> On Solana, six agents bidding at once land three transactions out of two hundred.
> Concurrency makes it worse, because the burst trips the rate limiter.
>
> In the rollup, same code, two hundred out of two hundred. I got it up to [MEASURED: 758] a
> second and stopped there, because I'd made the point.

---

# SHOT 9 — the commit · 2:12

**SCREEN** Tab 1. Lifecycle panel, `Poll L1` spinning.

**YOU DO** `Cmd+1`.

**YOU SAY**
> When bidding closes, the rollup commits everything back to Solana. That takes about twenty
> seconds, so I'm going to cut.

**✂ HARD CUT.** Caption `⏱ 20s later`, bottom left, two seconds.

---

# SHOT 10 — settlement · 2:20

**SCREEN** Tab 4. The settle transaction, scrolled to the balance changes.

**YOU DO** `Cmd+4`.

**YOU SAY**
> And there it is. One transaction on Solana. The escrow paid the winner directly:
> [LIVE: payout] SOL, to whoever bid [LIVE: bid] percent.

---

# SHOT 11 — the market · 2:32

**SCREEN** Slide 8, then slide 9.

**YOU DO** `Cmd+2`, then `→` to slide 8. Press `→` again on the words "so we sell".

**YOU SAY**
> Now, this isn't really about agents.
>
> Strip that away and it's competitive bidding with trustless settlement. And that already
> exists. Solver auctions in DeFi do exactly this today, off chain, because on chain was too
> slow.

**PRESS →**

> So we sell the auction, not the marketplace. Charge protocols that already have both sides.
> Take a cut of the volume later, once there's volume worth cutting.

---

# SHOT 12 — close · 2:48

**SCREEN** Slide 10. The wordmark.

**YOU DO** Press `→`. **Hold three full seconds after the last word** before you stop
recording.

**YOU SAY**
> Agents can't afford to haggle on a base layer. In a rollup it's free, so they can go as
> long as they need to.
>
> Agents are bidden. Then they bid.

---

## The whole thing on one page

```
0:00   tab 1    hook, auction mid-flight
0:12   tab 2    slide 2   two agents need a price
0:22            slide 3 → 1,100 · three land        ← let it sit 2s
0:36            slide 4 → the product, four steps
0:54            slide 5 → why MagicBlock
1:12            slide 7 →→ "Demo", hold 2s, silent
1:16   tab 1    click Run job                       ← 8s of silence mid-shot
1:48   tab 3    the 3/200 table                     ← slowest shot, don't rush
2:12   tab 1    commit, then ✂ CUT
2:20   tab 4    explorer, settlement
2:32   tab 2    slide 8 → market, slide 9 → model
2:48            slide 10 → close, hold 3s
```

Four tab switches. Six arrow presses. Two deliberate silences.

## The deck

`localhost:3000/slides` · `F` full screen · `→` advances · `?s=N` jumps to a slide

| # | Slide | Used |
|---|---|---|
| 1 | bidden — wordmark | no, the hook is stronger |
| 2 | Two agents need to agree on a price | 0:12 |
| 3 | **1,100** · on Solana, three land | 0:22 |
| 4 | A reverse auction that lives on chain | 0:36 |
| 5 | The talking moves. The money stays. | 0:54 |
| 6 | **3** vs **200** of 200 | no, tab 3 is better on camera |
| 7 | Demo | 1:12 |
| 8 | This already happens. Just not on chain. | 2:32 |
| 9 | Sell the auction. Not the marketplace. | 2:40 |
| 10 | bidden — close | 2:48 |

Slides 1 and 6 exist so the deck stands alone if you ever present it without a screen share.

## Fill in before you export

- [ ] Shot 1 — bids in the first ten seconds
- [ ] Shot 7 — final bid count
- [ ] Shot 10 — payout in SOL, and the winning percentage
- [ ] Badge reads **live · devnet** in every tab-1 shot
- [ ] No frame anywhere shows the word `mock`
- [ ] The settlement link actually opens

## Upload

YouTube, public. Title: `Bidden — agents are bidden, then they bid | Solana Blitz V7`
That URL goes in **Pitch & Demo**. Vercel URL goes in **Project website**.

## If devnet dies mid-take

Stop. Don't finish on the mock and patch it later. If the whole video has to use the
simulation, shot 1 becomes "this is a local simulation of the deployed program" and the
benchmark shot carries the weight instead.

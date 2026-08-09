# Bidden — video script

**~3:00.** Thirteen shots. Four browser tabs, nothing else open.

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
- **"Solana only lands 3 of 200."** That was a public-RPC rate limit, not a Solana limit. A
  paid endpoint does far better and we never measured it. The defensible argument is cost and
  reaction latency, which is what slide 3 shows. If someone asks, say exactly that.

---

# SHOT 1 — the hook · 0:00

**SCREEN** Tab 1. An auction already running. Counter climbing, price falling.

**YOU DO** Start recording once the counter is past ~300. No cursor.

**YOU SAY**
> This is an auction running on Solana. Not the result of one, the auction itself.
>
> Every one of those bids is a real transaction. That's [LIVE: N] of them in the last ten
> seconds, and none of them cost anything.

---

# SHOT 2 — the problem · 0:12

**SCREEN** Tab 2, slide 2. *Bidding doesn't work on a blockchain.*

**YOU DO** `Cmd+2`. One beat, then speak.

**YOU SAY**
> Here's the thing nobody says out loud. Bidding doesn't really work on a blockchain.
>
> Not NFT auctions, not solver auctions, not compute markets. Anywhere buyers compete in real
> time, the competing happens somewhere else.

---

# SHOT 3 — why · 0:24

**SCREEN** Slide 3. **60** against **698**.

**YOU DO** Press `→`. **Let the numbers sit two seconds before you speak.**

**YOU SAY**
> And the reason is simple. A bid is a transaction.
>
> On Solana, confirmation takes about half a second. So in a thirty second auction you get
> maybe sixty rounds of back and forth, and you pay a fee on every single one.
>
> In a rollup, blocks land in forty milliseconds. Same thirty seconds, about seven hundred
> rounds, and the bids are free.

---

# SHOT 4 — how it's solved today · 0:42

**SCREEN** Slide 4. *So everyone moved the auction off chain.*

**YOU DO** Press `→`.

**YOU SAY**
> So nobody runs the auction on chain. They run it off chain and settle the result.
>
> CoW Swap, UniswapX, 1inch, the compute markets. Solvers submit privately, an operator picks
> a winner, and the chain only ever sees who won.
>
> Billions of dollars clear that way, and every time, you're trusting the operator.

---

# SHOT 5 — what we built · 0:58

**SCREEN** Slide 5. The four-step flow.

**YOU DO** Press `→`.

**YOU SAY**
> So we put the auction back on chain.
>
> You post a job and fund an escrow. Bidders compete, and every bid is a real transaction
> anyone can see. Lowest wins, and the escrow pays out.
>
> No operator in the middle.

---

# SHOT 6 — how · 1:14

**SCREEN** Slide 6. The two-lane diagram.

**YOU DO** Press `→`.

**YOU SAY**
> The way that works is a MagicBlock Ephemeral Rollup.
>
> The auction and everyone bidding move into the rollup for thirty seconds. That's where you
> get the forty millisecond blocks and the free transactions.
>
> The escrow never goes with them. It stays on Solana the whole time. If the rollup
> disappeared mid-auction, the money is still sitting right there.

---

# SHOT 7 — the divider · 1:32

**SCREEN** Slide 7. **Demo**.

**YOU DO** Press `→`. Hold two seconds. Silent.

---

# SHOT 8 — the live run · 1:36

**SCREEN** Tab 1.

**YOU DO** `Cmd+1`. Cursor visible, click **Run job**, then hide the cursor.

**YOU SAY**
> Here's one running for real, on devnet.
>
> Six agents, each with its own wallet and its own price floor. They undercut each other for
> thirty seconds.

**PAUSE ~8 SECONDS.** Say nothing. Let the counter climb and the curve fall.

**YOU SAY**
> [LIVE: N] bids. Every one on chain, and the whole auction cost nothing to run.

---

# SHOT 9 — the proof · 2:08

**SCREEN** Tab 3. The benchmark table.

**YOU DO** `Cmd+3`. Scroll so the table fills the frame, then stop.

**YOU SAY**
> I measured all of this rather than assuming it. Same program, same laptop, one endpoint
> swapped.
>
> The rollup held seven hundred and fifty eight transactions a second with nothing dropped,
> and I stopped there because I'd made the point.
>
> The harness is in the repo. You can rerun it.

---

# SHOT 10 — the commit · 2:26

**SCREEN** Tab 1. `Poll L1` spinning.

**YOU DO** `Cmd+1`.

**YOU SAY**
> When bidding closes, the rollup commits everything back to Solana. That takes about twenty
> seconds, so I'm going to cut.

**✂ HARD CUT.** Caption `⏱ 20s later`, bottom left, two seconds.

---

# SHOT 11 — settlement · 2:34

**SCREEN** Tab 4. The settle transaction, scrolled to the balance changes.

**YOU DO** `Cmd+4`.

**YOU SAY**
> And there it is. One transaction on Solana, and the escrow pays the winner directly.
> [LIVE: payout] SOL, to whoever bid [LIVE: bid] percent.

---

# SHOT 12 — the model · 2:44

**SCREEN** Slide 8.

**YOU DO** `Cmd+2`, then `→` to slide 8.

**YOU SAY**
> We're not trying to start a marketplace. Marketplaces need two sides, and solver networks
> already have both.
>
> So we sell them the auction. Charge per auction, take a cut of volume later.

---

# SHOT 13 — close · 2:56

**SCREEN** Slide 9. The wordmark.

**YOU DO** Press `→`. **Hold three seconds after the last word.**

**YOU SAY**
> Auctions moved off chain because chains were too slow. That reason just expired.

---

## The whole thing on one page

```
0:00   tab 1    hook, auction mid-flight
0:12   tab 2    slide 2   bidding doesn't work on chain
0:24            slide 3 → 60 vs 698 rounds          ← let it sit 2s
0:42            slide 4 → so everyone moved off chain
0:58            slide 5 → what we built
1:14            slide 6 → how, the rollup split
1:32            slide 7 → "Demo", hold 2s, silent
1:36   tab 1    click Run job                       ← 8s of silence mid-shot
2:08   tab 3    the benchmark                       ← don't rush
2:26   tab 1    commit, then ✂ CUT
2:34   tab 4    explorer, settlement
2:44   tab 2    slide 8 → the model
2:56            slide 9 → close, hold 3s
```

Four tab switches. Six arrow presses. Two deliberate silences.

## The deck

`localhost:3000/slides` · `F` full screen · `→` advances · `?s=N` jumps to a slide

| # | Slide | Used |
|---|---|---|
| 1 | bidden — wordmark | no, the hook is stronger |
| 2 | Bidding doesn't work on a blockchain | 0:12 |
| 3 | **60** vs **698** rounds of bidding | 0:24 |
| 4 | So everyone moved the auction off chain | 0:42 |
| 5 | Put the auction back on chain | 0:58 |
| 6 | The bidding moves. The money stays. | 1:14 |
| 7 | Demo | 1:32 |
| 8 | Sell the auction. Not the marketplace. | 2:44 |
| 9 | bidden — close | 2:56 |

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

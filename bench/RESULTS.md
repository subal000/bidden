# bench — measured results

Benchmark harness for MagicBlock Ephemeral Rollups, written for `swarm` (Solana Blitz V7).

Every number here was measured on one machine against Solana devnet and the `devnet-as`
Ephemeral Rollup endpoint. Nothing is estimated. Where something was not tested, it says so.

---

## Two kinds of evidence, kept separate

This distinction matters and is easy to blur, so it is stated first.

**L1 transactions are independently verifiable.** They are on Solana devnet and clickable on
any explorer. The delegate / commit / undelegate lifecycle is entirely L1-visible.

**ER transactions are not on any public explorer.** MagicBlock ships RPC endpoints and a
router, not an explorer. ER transaction history *is* queryable through the ER RPC itself
(`getTransaction` and `getSignaturesForAddress` both work, verified), and Solana Explorer
can be pointed at a custom RPC URL, but there is no permanent public record and the
retention window is unknown. An ER transaction from this project still resolved 11.2 hours
after it was sent; beyond that, untested.

So: **the 13,405 ER transaction figure is our measurement, evidenced by this harness and the
captured output in `logs/`. It is not independently verifiable the way an L1 signature is.**
Treat it as a measurement, not as a citation.

---

## Verifiable on the Solana explorer right now

The M0 counter program used the same delegation lifecycle as `swarm`. Its PDA is
`Foo2tpFY3zhbpJSJsRA5eDuZGVh2z7xfANW5VD9HKKdd`, with **125 L1 transactions** between
2026-08-07 12:45:12 and 18:57:55. Machine-captured list in [`logs/l1-evidence.json`](logs/l1-evidence.json).

The three that prove the lifecycle:

| What | Signature |
|---|---|
| `Initialize` on L1 | [`4z4SAu5uQpCJX9KYfonC7cZjHyhKTht6qoGAWV8HTWpBY9tRLQi3RmSCxumtRwtMTjrYxFAbnZeqvzuRNFgFADp8`](https://explorer.solana.com/tx/4z4SAu5uQpCJX9KYfonC7cZjHyhKTht6qoGAWV8HTWpBY9tRLQi3RmSCxumtRwtMTjrYxFAbnZeqvzuRNFgFADp8?cluster=devnet) |
| `Delegate` — ownership moves to `DELeGG…` | [`3ar1DAAjzTUjBokkqBvn15saLphPqcBpvUuYq8d8cqs6ai87sNchcptmTzC6Mow5kXWhfmkMmFpWjcdThvW3pwf4`](https://explorer.solana.com/tx/3ar1DAAjzTUjBokkqBvn15saLphPqcBpvUuYq8d8cqs6ai87sNchcptmTzC6Mow5kXWhfmkMmFpWjcdThvW3pwf4?cluster=devnet) |
| **`ProcessUndelegation`** — the ER validator writing committed state back to L1 | [`3Xqxsz989EFoNmeeU4AVXCaDxNfoq6NL1q8oWrEeqriB3ry4BvvtsaCMhempnkWCZsZ42HdzJP3m4o9kQguC9hSA`](https://explorer.solana.com/tx/3Xqxsz989EFoNmeeU4AVXCaDxNfoq6NL1q8oWrEeqriB3ry4BvvtsaCMhempnkWCZsZ42HdzJP3m4o9kQguC9hSA?cluster=devnet) |

The third is the important one. It is the delegation program executing the commit on L1,
which is the part of the design that could not be assumed and had to be proven.

The [counter account](https://explorer.solana.com/address/Foo2tpFY3zhbpJSJsRA5eDuZGVh2z7xfANW5VD9HKKdd?cluster=devnet)
is still there, owned by the program, holding its final committed value of **13,524**.

**The program itself is closed.** `BseL5WXo2AZY5kqqhLbsmnH7FnXFFiKfMiNDAcojjmEJ` was closed
under loader-v3 to reclaim 2.0586 SOL of rent toward the `swarm` deploy. The program account
still exists and still reads `executable: true`; its programdata account is gone. That is why
every transaction-sending mode in this harness now exits with an explanation.

---

## ER versus L1

Same program, same machine, one endpoint swap. n=200 per configuration. **Landed counts are
read from the counter account afterwards, never from what the client believed it sent.**

| Configuration | L1 sent | L1 landed | L1 tx/s | ER landed | ER tx/s |
|---|---|---|---|---|---|
| Sequential, await confirm | 16/200 | 16 | 0.5 | 200/200 | 4.2 |
| Sequential, fire and forget | 20/200 | 20 | 0.8 | 200/200 | 8.1 - 12.1 |
| **6 concurrent + 50-100ms jitter** | **3/200** | **3** | **0.5** | **200/200** | **26 - 38** |

**Concurrency actively hurts on L1.** The strategy producing 26-38 tx/s in the rollup lands
three transactions out of two hundred on devnet, because the burst trips the public RPC
limiter immediately (197 rejected, versus 180 when sequential).

Being precise, because this is an infrastructure limit and not a claim about Solana
consensus: public devnet caps `sendTransaction` at roughly 40 per 10 seconds. **L1 never
dropped anything it accepted**, in any of the four runs. Paced under the limit it lands 40/40
with confirm latency median 498ms (min 181, max 721) — the 400ms slot time appearing exactly
where it should — plus 5000 lamports per transaction where the rollup charges zero.

## ER throughput ceiling

| Senders | n | Landed | tx/s |
|---|---|---|---|
| 6 | 200 | 200/200 | 70.6 |
| 12 | 400 | 400/400 | 116.9 |
| 24 | 2000 | 2000/2000 | 228.5 |
| 96 | 2000 | 2000/2000 | 770.2 |
| 96 | 4000 | 4000/4000 | 758.3 |

**Ceiling not found.** 100% landed at 758 tx/s. The constraint is client-side concurrency
and round-trip latency, not the rollup.

## Other measurements

- **ER block rate 22-23 blocks/s, ~43ms per block**, identical idle and under 758 tx/s load.
  This contradicts the 1ms figure in MagicBlock's marketing material, so this project does
  not use it. The rollup packs ~33 transactions per block, which is where the throughput
  comes from.
- **Blockhash window 1198 blocks, ~54s.** Probed directly: lands at 1s, 5s, 15s, 30s, 45s;
  rejects at 60s with `rpc -32003: Blockhash not found`.
- **Concurrent writes to the same PDA serialize, they do not fail.** Zero `AccountInUse` and
  zero errors across all 13,405 transactions. The shared-PDA configuration measured *faster*
  (70.6 tx/s) than six separate PDAs (54.7 tx/s).
- **Websocket is lossless at demo rate, coalescing under stress.** 100% delivery of
  intermediate values at 31 tx/s; 93.2% at 724 tx/s with 33 skipped runs, largest jump 25.
  This is why the UI reads `bid_count` from the account payload rather than counting events.
- **Region latency from this machine**: `devnet-as` 284ms, `devnet-eu` 813ms, `devnet-us`
  858ms.
- **13,405 ER transactions sent, zero failures**, across M0 and M1.

## Two client requirements that came out of measurement

**Tune the HTTP transport.** Go's default of 2 idle connections per host silently serializes
concurrent senders behind TLS handshakes. `rpc.go` has the working configuration. Without it
every concurrency number above would be an artifact of the client.

**Cache one blockhash, refresh in the background.** At 54 seconds of validity a long run
outlives a single blockhash, but fetching one per transaction adds a full round trip and puts
throughput back at ~4 tx/s. `main.go` has a 20-second background refresher.

---

## Running it

```bash
go run . --mode read          # works: reads the counter on L1 and in the ER
go run . --mode seq-fire      # exits 3, explains that the program is closed
```

Every transaction-sending mode (`prepare`, `blockhash`, `seq-confirm`, `seq-fire`, `conc`,
`ws`, `suite`, `sweep`, `undelegate`) checks whether the program's programdata account still
exists and refuses to run if not. The check is live rather than hardcoded, so the harness
starts working again the moment a program is deployed to that address.

## Provenance of the numbers above

The tables are the recorded results of runs performed on 2026-08-07. `logs/` holds the
console output captured from those runs. `logs/l1-evidence.json` was generated by querying
devnet directly and can be regenerated at any time.

The ER-side numbers cannot currently be reproduced, because the program they ran against is
closed. That is a real limitation of this artifact and is the reason the distinction at the
top of this file exists.

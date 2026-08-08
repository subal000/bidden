import type { AgentView, BidEvent, Curve, JobSource, JobState } from "./types";

/**
 * Reads the Job account and every AgentRegistry straight off the Ephemeral
 * Rollup, in one getMultipleAccounts call per poll.
 *
 * Structurally identical to MockSource: submit_bid writes each agent's own
 * delegated registry, so every agent's live bid is readable natively. No
 * transaction log parsing, and no agent freezes at a stale value.
 *
 * Job.bid_count is the authoritative counter. Per agent counts are display
 * only; if they ever disagree, the job total is what goes on screen.
 */

const STATUS = ["Open", "Bidding", "Awarded", "Settled"] as const;

/** Pubkey::default() serialises to 32 zero bytes, which is base58 "111…1".
 *  It means "no bidder yet", not an address. */
const DEFAULT_PUBKEY = "1".repeat(32);

/** Mirrors parseJobState in agents/chain.go. Borsh order after the discriminator. */
export function decodeJob(data: Uint8Array): JobState {
  if (data.length < 130) throw new Error(`job account too short: ${data.length} bytes`);
  const dv = new DataView(data.buffer, data.byteOffset, data.byteLength);
  // Job is 130 bytes: 8 discriminator, then job_id at 8, requester at 16.
  return {
    bestBidBps: dv.getUint16(82, true),
    bidCount: dv.getUint32(116, true),
    bestBidder: (() => {
      const raw = data.slice(84, 116);
      // Decide from the bytes. An all-zero pubkey means "no bidder yet".
      return raw.every((b) => b === 0) ? "" : toBase58(raw);
    })(),
    status: (STATUS[data[128]] ?? "Open") as JobState["status"],
  };
}

export type AgentAccount = {
  authority: string;
  completed: number;
  reputation: number;
  earned: bigint;
  lastBidBps: number;
  bidCount: number;
};

/** Mirrors parseAgent in driver/state.go. AgentRegistry is 62 bytes. */
export function decodeAgent(data: Uint8Array): AgentAccount {
  if (data.length < 62) throw new Error(`agent account too short: ${data.length} bytes`);
  const dv = new DataView(data.buffer, data.byteOffset, data.byteLength);
  return {
    authority: toBase58(data.slice(8, 40)),
    completed: dv.getUint32(41, true),
    reputation: dv.getUint16(45, true),
    earned: dv.getBigUint64(47, true),
    lastBidBps: dv.getUint16(55, true),
    bidCount: dv.getUint32(57, true),
  };
}

const B58 = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

/**
 * Seeding the digit array with [0] made an all-zero input encode to 33 "1"s
 * instead of 32, so Pubkey::default() never matched DEFAULT_PUBKEY and rendered
 * as an address. Same off-by-one hits any key with a leading zero byte, which
 * is roughly 1 in 256 of them.
 */
function toBase58(bytes: Uint8Array): string {
  const digits: number[] = [];
  for (const byte of bytes) {
    let carry = byte;
    for (let j = 0; j < digits.length; j++) {
      carry += digits[j] << 8;
      digits[j] = carry % 58;
      carry = (carry / 58) | 0;
    }
    while (carry > 0) {
      digits.push(carry % 58);
      carry = (carry / 58) | 0;
    }
  }
  let zeros = 0;
  while (zeros < bytes.length && bytes[zeros] === 0) zeros++;
  let out = "1".repeat(zeros);
  for (let i = digits.length - 1; i >= 0; i--) out += B58[digits[i]];
  return out;
}

export type Deployment = {
  erRpc: string;
  /** Public endpoint: read from the visitor's browser, so it carries no key. */
  l1Rpc: string;
  budgetLamports: number;
  jobPda: string;
  escrowPda: string;
  requester: string;
  jobId: number;
  agents: { name: string; authority: string; pda: string }[];
};

export class ChainSource implements JobSource {
  private job: JobState;
  private agents: AgentView[];
  private log: BidEvent[] = [];
  private listeners = new Set<() => void>();
  private timer: ReturnType<typeof setInterval> | null = null;
  private started = 0;
  private settled: { signature: string; explorerUrl: string } | null = null;
  private history: { t: number; bps: number }[] = [];
  private keys: string[];
  private uid = 0;
  private baseline: number | null = null;
  private recovering = false;
  // AgentRegistry.bid_count is lifetime across every job the agent has ever bid
  // on. The cards must show this job's share, or they sum to more than the job
  // counter and the screen contradicts itself.
  private agentBaseline: (number | null)[] = [];

  constructor(
    private dep: Deployment,
    curve: Curve,
    private pollMs = 150,
  ) {
    this.job = { bidCount: 0, bestBidBps: curve.startBps, bestBidder: "", status: "Open" };
    // Agent card order follows curve.json; the deployment file is emitted in
    // the same order by `driver --mode addresses`.
    this.agents = curve.agents.map((spec) => ({
      spec,
      lastBidBps: null,
      sent: 0,
      improved: 0,
      pulse: 0,
    }));
    this.keys = [dep.jobPda, ...dep.agents.map((a) => a.pda)];
  }

  start() {
    this.started = Date.now();
    const tick = async () => {
      try {
        const res = await fetch(this.dep.erRpc, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            jsonrpc: "2.0",
            id: 1,
            method: "getMultipleAccounts",
            params: [this.keys, { encoding: "base64", commitment: "processed" }],
          }),
        });
        const json = await res.json();
        const vals = json?.result?.value;
        if (!Array.isArray(vals) || !vals[0]) return;

        const raw = (v: { data: string[] }) =>
          Uint8Array.from(atob(v.data[0]), (ch) => ch.charCodeAt(0));

        const next = decodeJob(raw(vals[0]));

        for (let i = 0; i < this.agents.length; i++) {
          const v = vals[i + 1];
          if (!v) continue;
          const acct = decodeAgent(raw(v));
          const view = this.agents[i];
          if (this.agentBaseline[i] == null) this.agentBaseline[i] = acct.bidCount;
          const sessionSent = acct.bidCount - (this.agentBaseline[i] as number);
          if (sessionSent !== view.sent) {
            view.pulse += 1;
            // One log row per observed change. At 38 bids/s against a 150ms
            // poll several bids collapse into one row; the counter is
            // unaffected because it comes from the job payload.
            this.log.unshift({
              uid: ++this.uid,
              seq: next.bidCount,
              agent: view.spec.name,
              bidBps: acct.lastBidBps,
              improved: next.bestBidder === this.dep.agents[i]?.authority,
              at: Date.now(),
            });
            if (this.log.length > 400) this.log.length = 400;
          }
          // Lifetime value, carried over from previous jobs. Only meaningful
          // once this agent has actually bid on the current one.
          view.lastBidBps = sessionSent > 0 ? acct.lastBidBps || null : null;
          view.sent = sessionSent;
          if (next.bestBidder === this.dep.agents[i]?.authority && next.bestBidBps !== this.job.bestBidBps) {
            view.improved += 1;
          }
        }

        if (this.baseline === null) this.baseline = next.bidCount;
        if (next.status === "Settled") void this.recoverSettlement();
        this.job = next;
        this.history.push({ t: this.elapsedMs() / 1000, bps: next.bestBidBps });
        this.emit();
      } catch {
        // Transient RPC failure. Keep the last good payload on screen rather
        // than flashing a zero: the counter must stay monotonic.
      }
    };
    this.timer = setInterval(tick, this.pollMs);
    void tick();
  }

  stop() {
    if (this.timer) clearInterval(this.timer);
    this.timer = null;
  }

  /**
   * A visitor arriving after a run has no orchestrator state, so the settlement
   * signature is recovered from the chain: the most recent L1 transaction
   * touching the Job is the settle. One call, only once, never polled.
   */
  private async recoverSettlement() {
    if (this.settled || this.recovering) return;
    this.recovering = true;
    try {
      const res = await fetch(this.dep.l1Rpc, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          id: 1,
          method: "getSignaturesForAddress",
          params: [this.dep.jobPda, { limit: 1 }],
        }),
      });
      const json = await res.json();
      const sig = json?.result?.[0]?.signature;
      if (sig && !json.result[0].err) this.setSettlement(sig);
    } catch {
      // Best effort. The rest of the result panel still renders.
    }
  }

  /** Called once the L1 settle transaction lands, to close the demo. */
  setSettlement(signature: string) {
    this.settled = {
      signature,
      explorerUrl: `https://explorer.solana.com/tx/${signature}?cluster=devnet`,
    };
    this.emit();
  }

  /** Maps the on chain best_bidder pubkey to the agent name the cards use. */
  bestBidderName(): string {
    const i = this.dep.agents.findIndex((a) => a.authority === this.job.bestBidder);
    return i >= 0 ? this.dep.agents[i].name : "";
  }

  getJob() {
    return { ...this.job, bestBidder: this.bestBidderName() || this.job.bestBidder };
  }
  getAgents() {
    return this.agents.map((a) => ({ ...a }));
  }
  getLog() {
    return this.log;
  }
  getHistory() {
    return this.history;
  }
  baselineBidCount() {
    return this.baseline ?? 0;
  }
  elapsedMs() {
    return this.started ? Date.now() - this.started : 0;
  }
  settlement() {
    return this.settled;
  }
  subscribe(fn: () => void) {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }
  private emit() {
    this.listeners.forEach((f) => f());
  }
}

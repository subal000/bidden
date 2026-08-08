import type { AgentSpec, AgentView, BidEvent, Curve, JobSource, JobState } from "./types";

/**
 * In browser market simulation. Zero network calls, so the whole demo runs with
 * devnet unreachable.
 *
 * The bidding rules below mirror agents/agent.go and the submit_bid handler in
 * programs/swarm/src/lib.rs. That duplication is deliberate: this file is the
 * standalone fallback for the video, and the Go copy is what tunes the real
 * agents. Both read the same curve.json, so the tuning cannot drift even though
 * the code exists twice.
 */

const LOG_CAP = 400;

export function effectiveFloor(s: AgentSpec, discount: number): number {
  return Math.max(1, Math.floor(s.floorBps * (1 - (discount * s.reputation) / 10000)));
}

/** Mirrors AgentSpec.NextBid in agents/agent.go. */
function nextBid(s: AgentSpec, job: JobState, own: number, c: Curve, rnd: () => number): number {
  const floor = effectiveFloor(s, c.reputationDiscount);
  const best = job.bestBidBps;

  // Holding restates this agent's own standing offer, not the market best.
  // Echoing the best would print the same number on every agent card.
  const hold = own || c.startBps;

  if (job.bestBidder === s.name) return hold; // already winning, hold
  if (best <= floor) return floor; // cannot beat it and stay profitable
  if (rnd() > s.aggression) return hold; // patience, restate rather than cut

  const gap = best - floor;
  let tick = gap * s.decay;
  tick *= 1 - c.tickNoise + rnd() * 2 * c.tickNoise;
  if (tick < 1) tick = 1;
  return Math.max(floor, Math.round(best - tick));
}

// Deterministic PRNG so a given seed always produces the same curve.
function mulberry32(seed: number) {
  let a = seed >>> 0;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export class MockSource implements JobSource {
  private curve: Curve;
  private job: JobState;
  private agents: AgentView[];
  private log: BidEvent[] = [];
  private listeners = new Set<() => void>();
  private timers: ReturnType<typeof setTimeout>[] = [];
  private raf: ReturnType<typeof setInterval> | null = null;
  private started = 0;
  private running = false;
  private settled: { signature: string; explorerUrl: string } | null = null;
  private history: { t: number; bps: number }[] = [];
  private uid = 0;

  constructor(curve: Curve, private seed = 1) {
    this.curve = curve;
    this.job = {
      bidCount: 0,
      bestBidBps: curve.startBps,
      bestBidder: "",
      status: "Open",
    };
    this.agents = curve.agents.map((spec) => ({
      spec,
      lastBidBps: null,
      sent: 0,
      improved: 0,
      pulse: 0,
    }));
  }

  start() {
    if (this.running) return;
    this.running = true;
    this.started = Date.now();
    this.curve.agents.forEach((spec, i) => this.loop(spec, i, mulberry32(this.seed + i * 7919)));
    // Repaint on a fixed cadence rather than per bid: at 38 bids/s a render per
    // event would thrash React for no visual gain.
    this.raf = setInterval(() => {
      this.history.push({ t: this.elapsedMs() / 1000, bps: this.job.bestBidBps });
      this.emit();
    }, 100);
  }

  stop() {
    this.running = false;
    this.timers.forEach(clearTimeout);
    this.timers = [];
    if (this.raf) clearInterval(this.raf);
    this.raf = null;
  }

  private loop(spec: AgentSpec, idx: number, rnd: () => number) {
    const step = () => {
      if (!this.running) return;
      if (Date.now() - this.started > this.curve.durationSeconds * 1000) {
        this.finish();
        return;
      }
      const own = this.agents[idx].lastBidBps ?? 0;
      const bid = nextBid(spec, this.job, own, this.curve, rnd);
      this.submit(spec.name, idx, bid);

      const jitter =
        this.curve.jitterMinMs +
        rnd() * (this.curve.jitterMaxMs - this.curve.jitterMinMs) +
        this.curve.sendLatencyMs;
      const t = setTimeout(step, jitter);
      this.timers.push(t);
    };
    step();
  }

  /** Mirrors submit_bid: always counts, only improves on a strict undercut. */
  private submit(agent: string, idx: number, bidBps: number) {
    this.job.bidCount += 1;
    if (this.job.status === "Open") this.job.status = "Bidding";

    let improved = false;
    if (this.job.status === "Bidding" && bidBps < this.job.bestBidBps) {
      this.job.bestBidBps = bidBps;
      this.job.bestBidder = agent;
      improved = true;
    }

    const a = this.agents[idx];
    a.lastBidBps = bidBps;
    a.sent += 1;
    a.pulse += 1;
    if (improved) a.improved += 1;

    this.log.unshift({ uid: ++this.uid, seq: this.job.bidCount, agent, bidBps, improved, at: Date.now() });
    if (this.log.length > LOG_CAP) this.log.length = LOG_CAP;
  }

  private finish() {
    if (this.job.status === "Settled") return;
    this.running = false;
    this.timers.forEach(clearTimeout);
    this.job.status = "Awarded";
    this.emit();
    // Settlement is asynchronous on chain: the ER schedules, the validator
    // executes on L1 seconds later. The mock waits too so the pacing of the
    // closing beat matches the real thing.
    setTimeout(() => {
      this.job.status = "Settled";
      this.settled = {
        signature: "MockedLocallyNoTransactionWasSent" + "1".repeat(55),
        explorerUrl: "",
      };
      this.emit();
    }, 2600);
  }

  getJob() {
    return { ...this.job };
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
    return 0;
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

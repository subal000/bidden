/**
 * JobState is the exact shape of the on chain Job account payload.
 *
 * The mock and the chain source both produce this, so switching between them is
 * a data source swap rather than a rewrite.
 */
export type JobState = {
  bidCount: number;
  bestBidBps: number;
  bestBidder: string;
  status: "Open" | "Bidding" | "Awarded" | "Settled";
};

export type BidEvent = {
  /** Unique per row. Several agents can share a seq in one poll, so seq is not
   *  a usable React key. */
  uid: number;
  seq: number;
  agent: string;
  bidBps: number;
  improved: boolean;
  at: number;
};

export type AgentSpec = {
  name: string;
  specialization: string;
  floorBps: number;
  reputation: number;
  decay: number;
  aggression: number;
};

export type Curve = {
  startBps: number;
  durationSeconds: number;
  jitterMinMs: number;
  jitterMaxMs: number;
  tickNoise: number;
  sendLatencyMs: number;
  reputationDiscount: number;
  agents: AgentSpec[];
};

/** Per agent view state the cards render. Not chain data. */
export type AgentView = {
  spec: AgentSpec;
  lastBidBps: number | null;
  sent: number;
  improved: number;
  pulse: number; // incremented on every send, drives the pulse animation
};

/**
 * A JobSource feeds the UI. Two implementations exist: the in browser mock and
 * the ER account reader. The contract is deliberately narrow.
 */
export interface JobSource {
  start(): void;
  stop(): void;
  /** Latest job payload. bidCount comes from here, never from counting events. */
  getJob(): JobState;
  getAgents(): AgentView[];
  /** Most recent bids, newest first. Presentation only. */
  getLog(): BidEvent[];
  /** Sampled best-bid over time. Drives the convergence chart. */
  getHistory(): { t: number; bps: number }[];
  /** bid_count already on the job when this source attached. Lets the rate
   *  readout describe this session rather than the account's whole history. */
  baselineBidCount(): number;
  subscribe(fn: () => void): () => void;
  elapsedMs(): number;
  settlement(): { signature: string; explorerUrl: string } | null;
}

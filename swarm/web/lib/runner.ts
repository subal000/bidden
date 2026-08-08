import { spawn } from "child_process";
import path from "path";

export type Phase =
  | "idle"
  | "preparing"
  | "bidding"
  | "awarding"
  | "committing"
  | "polling"
  | "settling"
  | "done"
  | "error";

export type RunState = {
  phase: Phase;
  startedAt: number | null;
  jobId: number;
  /** Signatures collected as the lifecycle advances, newest last. */
  steps: { name: string; status: "pending" | "active" | "done" | "failed"; detail?: string; sig?: string }[];
  undelegateGapMs: number | null;
  settleSig: string | null;
  /** Set once prepare picks a job, so the UI can follow it. */
  jobPda: string | null;
  escrowPda: string | null;
  error: string | null;
  log: string[];
};

const DRIVER = path.resolve(process.cwd(), "..", "driver");
const AGENTS = path.resolve(process.cwd(), "..", "agents");
const REQUESTER = "2DzDL6VrF1uzjcjd3z68u3GgSxrGzor4RPfJqNgSjucd";

const STEPS = [
  { name: "Prepare job", key: "preparing" },
  { name: "Bidding", key: "bidding" },
  { name: "Award", key: "awarding" },
  { name: "Commit + undelegate", key: "committing" },
  { name: "Poll L1", key: "polling" },
  { name: "Settle on L1", key: "settling" },
] as const;

function freshState(jobId: number): RunState {
  return {
    phase: "idle",
    startedAt: null,
    jobId,
    steps: STEPS.map((s) => ({ name: s.name, status: "pending" as const })),
    undelegateGapMs: null,
    settleSig: null,
    jobPda: null,
    escrowPda: null,
    error: null,
    log: [],
  };
}

/**
 * Next.js dev re-evaluates route modules between requests, which resets
 * module-level state and made a run appear to fall back to idle mid-flight.
 * Pinning to globalThis survives those reloads. Single in-flight run: this
 * drives a demo, not a multi-tenant service.
 */
const g = globalThis as unknown as { __swarmRun?: { state: RunState; running: boolean } };
g.__swarmRun ??= { state: freshState(0), running: false };
const store = g.__swarmRun;

export function getState(): RunState {
  return store.state;
}

function mark(key: Phase, status: "active" | "done" | "failed", detail?: string, sig?: string) {
  const idx = STEPS.findIndex((s) => s.key === key);
  if (idx < 0) return;
  const state = store.state;
  state.steps[idx] = { ...state.steps[idx], status, detail, sig: sig ?? state.steps[idx].sig };
}

/**
 * Devnet routinely returns "not confirmed within 90s" for a transaction that
 * actually landed. Every driver mode skips completed work, so retrying is safe
 * and is what a human does at the CLI.
 */
async function runRetry(cmd: string, args: string[], cwd: string, attempts = 4): Promise<string> {
  let last: unknown;
  for (let i = 0; i < attempts; i++) {
    try {
      return await run(cmd, args, cwd);
    } catch (e) {
      last = e;
      const msg = e instanceof Error ? e.message : String(e);
      if (!/not confirmed within/.test(msg)) throw e;
      store.state.log = [
        ...store.state.log.slice(-200),
        `  retrying after devnet confirmation timeout (attempt ${i + 2}/${attempts})`,
      ];
    }
  }
  throw last;
}

function run(cmd: string, args: string[], cwd: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const p = spawn(cmd, args, { cwd, env: { ...process.env, GOFLAGS: "-mod=mod" } });
    let out = "";
    const push = (b: Buffer) => {
      const t = b.toString();
      out += t;
      for (const line of t.split("\n")) {
        if (line.trim()) store.state.log = [...store.state.log.slice(-200), line.trimEnd()];
      }
    };
    p.stdout.on("data", push);
    p.stderr.on("data", push);
    p.on("close", (code) => (code === 0 ? resolve(out) : reject(new Error(out.slice(-400)))));
    p.on("error", reject);
  });
}

/**
 * Signatures come from the explicit `SIG name=<full>` lines. The human readable
 * output prints sig[:20], which is not a valid signature and produces explorer
 * links that 404.
 */
const sigFrom = (out: string, name: string) =>
  out.match(new RegExp(`SIG ${name}=([1-9A-HJ-NP-Za-km-z]{80,90})`))?.[1] ?? undefined;

/** Fire and forget: the client polls /api/run for progress. */
export function startRun(startFrom: number) {
  let jobId = startFrom;
  if (store.running) return store.state;
  store.running = true;
  store.state = freshState(jobId);
  const state = store.state;
  state.startedAt = Date.now();

  (async () => {
    try {
      // A settled job is spent: award on one fails with BadState (0x1770).
      // Prepare finds the first unused id, creates and delegates it.
      state.phase = "preparing";
      mark("preparing", "active", "creating and delegating 7 accounts");
      const prep = await runRetry("go", ["run", ".", "--mode", "prepare", "--job-id", String(jobId)], DRIVER);
      const m = prep.match(/PREPARED job-id=(\d+) pda=(\w+) escrow=(\w+)/);
      if (!m) throw new Error("prepare did not report a job id");
      jobId = Number(m[1]);
      state.jobId = jobId;
      state.jobPda = m[2];
      state.escrowPda = m[3];
      mark("preparing", "done", `job #${jobId}`);

      state.phase = "bidding";
      mark("bidding", "active");
      const bid = await run("go", ["run", ".", "--mode", "live", "--requester", REQUESTER,
        "--job-id", String(jobId), "--keydir", "../driver/keys"], AGENTS);
      const landed = Number(bid.match(/bids landed\s+(\d+)/)?.[1] ?? 0);
      if (landed === 0) {
        throw new Error("bidding landed 0 bids; the job is probably not delegated");
      }
      mark("bidding", "done", `${landed} bids`);

      state.phase = "awarding";
      mark("awarding", "active");
      const award = await run("go", ["run", ".", "--mode", "award", "--job-id", String(jobId)], DRIVER);
      mark("awarding", "done", undefined, sigFrom(award, "award"));

      state.phase = "committing";
      mark("committing", "active");
      const und = await run("go", ["run", ".", "--mode", "undelegate", "--job-id", String(jobId)], DRIVER);
      mark("committing", "done", "scheduled", sigFrom(und, "undelegate"));

      state.phase = "polling";
      mark("polling", "active", "waiting for Solana to accept the commit");
      const poll = await run("go", ["run", ".", "--mode", "poll", "--job-id", String(jobId)], DRIVER);
      const gap = poll.match(/UNDELEGATE TO L1 GAP: ([\d.]+)s/);
      state.undelegateGapMs = gap ? Math.round(parseFloat(gap[1]) * 1000) : null;
      mark("polling", "done", state.undelegateGapMs ? `${(state.undelegateGapMs / 1000).toFixed(1)}s` : undefined);

      state.phase = "settling";
      mark("settling", "active");
      const settle = await run("go", ["run", ".", "--mode", "settle", "--job-id", String(jobId)], DRIVER);
      const sig = sigFrom(settle, "settle");
      state.settleSig = sig ?? null;
      mark("settling", "done", undefined, sig);

      state.phase = "done";
    } catch (e) {
      state.error = e instanceof Error ? e.message : String(e);
      state.phase = "error";
      const active = state.steps.findIndex((s) => s.status === "active");
      if (active >= 0) state.steps[active].status = "failed";
    } finally {
      store.running = false;
    }
  })();

  return store.state;
}

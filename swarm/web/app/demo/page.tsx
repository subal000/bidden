"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Play, RotateCcw, AlertCircle } from "lucide-react";
import curveJson from "@/lib/curve.json";
import deployment from "@/lib/deployment.json";
import { MockSource, effectiveFloor } from "@/lib/mockSource";
import { ChainSource, type Deployment } from "@/lib/chainSource";
import { AgentCard } from "@/components/AgentCard";
import { BidLog } from "@/components/BidLog";
import { ConvergenceChart } from "@/components/ConvergenceChart";
import { Timeline } from "@/components/Timeline";
import { Button } from "@/components/ui/Button";
import { Address } from "@/components/ui/Address";
import { bps, counter } from "@/lib/format";
import type { Curve, JobSource } from "@/lib/types";
import type { RunState } from "@/lib/runner";

const curve = curveJson as unknown as Curve;
const dep = deployment as unknown as Deployment & {
  jobId: number;
  escrowPda: string;
  requester: string;
};

const LIVE = true;

function sessionBidsSafe(total: number, baseline: number) {
  return Math.max(0, total - baseline);
}

export default function Demo() {
  const [runId, setRunId] = useState(0);
  const [, force] = useState(0);
  const [open, setOpen] = useState<string | null>(null);
  const [run, setRun] = useState<RunState | null>(null);
  const [starting, setStarting] = useState(false);
  const [runError, setRunError] = useState<string | null>(null);

  // The orchestrator picks a fresh job each run, so the page follows its PDA
  // rather than the address file baked in at build time.
  const started = run != null && run.phase !== "idle";
  const activeJobPda = (started && run.jobPda) || dep.jobPda;
  const source = useMemo<JobSource>(
    () =>
      LIVE
        ? new ChainSource({ ...dep, jobPda: activeJobPda }, curve)
        : new MockSource(curve, 1 + runId),
    [runId, activeJobPda],
  );

  useEffect(() => {
    const unsub = source.subscribe(() => force((n) => n + 1));
    source.start();
    return () => {
      unsub();
      source.stop();
    };
  }, [source]);

  // Poll the orchestrator while a run is in flight.
  useEffect(() => {
    if (!LIVE) return;
    let alive = true;
    const tick = async () => {
      try {
        const r = await fetch("/api/run", { cache: "no-store" });
        if (!alive) return;
        setRun(await r.json());
      } catch {
        // Orchestrator only exists locally. On a static deploy the page still
        // renders chain state; it just cannot start a run.
      }
    };
    void tick();
    const iv = setInterval(tick, 1000);
    return () => {
      alive = false;
      clearInterval(iv);
    };
  }, []);

  const start = useCallback(async () => {
    setStarting(true);
    setRunError(null);
    try {
      const r = await fetch("/api/run", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ jobId: dep.jobId }),
      });
      if (!r.ok) throw new Error((await r.json()).error ?? `HTTP ${r.status}`);
      setRun(await r.json());
    } catch (e) {
      setRunError(
        e instanceof Error
          ? `${e.message}. The orchestrator runs locally only — start the dev server from swarm/web.`
          : "Could not start the run.",
      );
    } finally {
      setStarting(false);
    }
  }, []);

  const job = source.getJob();
  const agents = source.getAgents();
  const floors = curve.agents.map((a) => effectiveFloor(a, curve.reputationDiscount));
  const floor = Math.min(...floors);

  const busy = run?.phase != null && run.phase !== "idle" && run.phase !== "done" && run.phase !== "error";
  // A page opened after a run has no session data, so live widgets would render
  // empty against a large counter and read as broken rather than complete.
  const finished = job.status === "Settled" && !busy && sessionBidsSafe(job.bidCount, source.baselineBidCount()) === 0;
  const secs = source.elapsedMs() / 1000;
  const sessionBids = Math.max(0, job.bidCount - source.baselineBidCount());

  return (
    <main className="mx-auto w-full max-w-7xl overflow-x-hidden px-4 py-8 sm:px-6">
      {/* Header row */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-white">Live auction</h1>
          <p className="mt-1 text-[12px] text-dim">
            Job #{started && run.jobId ? run.jobId : dep.jobId} · escrow 0.02 SOL held on Solana L1, never delegated
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span
            className={[
              "rounded px-2 py-1 text-[10px] font-semibold uppercase tracking-wider",
              LIVE ? "bg-accent/15 text-accent" : "bg-edge text-warn",
            ].join(" ")}
          >
            {LIVE ? "live · devnet" : "mock"}
          </span>
          <Button
            onClick={start}
            loading={starting || busy}
            disabled={busy}
            icon={run?.phase === "done" ? <RotateCcw className="h-3.5 w-3.5" aria-hidden /> : <Play className="h-3.5 w-3.5" aria-hidden />}
          >
            {busy ? "Auction running" : run?.phase === "done" ? "Run again" : "Run job"}
          </Button>
        </div>
      </div>

      {run?.phase === "error" && run.error && (
        <div
          role="alert"
          className="mt-4 flex items-start gap-2.5 rounded-lg border border-hot/40 bg-hot/[0.06] px-4 py-3 text-[12px] text-pale"
        >
          <AlertCircle className="mt-px h-4 w-4 shrink-0 text-hot" aria-hidden />
          <span>
            <strong className="text-hot">Run failed.</strong>{" "}
            {run.error.slice(-220)}
          </span>
        </div>
      )}
      {runError && (
        <div
          role="alert"
          className="mt-4 flex items-start gap-2.5 rounded-lg border border-hot/40 bg-hot/[0.06] px-4 py-3 text-[12px] text-pale"
        >
          <AlertCircle className="mt-px h-4 w-4 shrink-0 text-hot" aria-hidden />
          <span>{runError}</span>
        </div>
      )}

      <div className="mt-6 grid gap-4 lg:grid-cols-[320px_1fr_300px]">
        {/* Agents */}
        <section aria-label="Agents" className="flex min-w-0 flex-col gap-2">
          {finished && (
            <div className="rounded-lg border border-edge bg-panel px-3.5 py-3 text-[11px] leading-relaxed text-dim">
              This auction has finished. Agent cards show live bids during a run; press{" "}
              <span className="text-pale">Run job</span> to start another.
            </div>
          )}
          {agents.map((a, i) => (
            <AgentCard
              key={a.spec.name}
              a={a}
              winning={job.bestBidder === a.spec.name}
              pda={dep.agents[i]?.pda}
              authority={dep.agents[i]?.authority}
              floorBps={floors[i]}
              expanded={open === a.spec.name}
              onToggle={() => setOpen(open === a.spec.name ? null : a.spec.name)}
            />
          ))}
        </section>

        {/* Centre: counter, price, chart */}
        <section aria-label="Auction state" className="flex min-w-0 flex-col gap-4">
          <div className="rounded-lg border border-edge bg-panel px-6 py-5">
            <div className="text-[10px] uppercase tracking-[0.16em] text-dim">
              Bids submitted onchain
            </div>
            <div className="mt-1 flex items-baseline gap-3">
              <span className="text-6xl font-bold leading-none tracking-tight tabular-nums text-white">
                {counter(job.bidCount)}
              </span>
              {secs > 1 && sessionBids > 0 && (
                <span className="pb-1 text-sm tabular-nums text-dim">
                  {(sessionBids / secs).toFixed(0)}/s
                </span>
              )}
            </div>
            <p className="mt-2 text-[11px] text-dim">
              Every one a real transaction inside the Ephemeral Rollup, zero fees
            </p>
          </div>

          <div className="rounded-lg border border-edge bg-panel px-6 py-4">
            <div className="flex items-baseline justify-between">
              <span className="text-[10px] uppercase tracking-[0.16em] text-dim">Best bid</span>
              <span className="text-[11px] text-dim">
                {job.bestBidder || "no bids yet"}
              </span>
            </div>
            <div className="mt-1 text-4xl font-semibold leading-none tabular-nums text-accent">
              {bps(job.bestBidBps)}
            </div>
            <div className="mt-4 h-1.5 w-full overflow-hidden rounded-full bg-edge">
              <div
                className="h-full rounded-full bg-gradient-to-r from-hot to-accent transition-[width] duration-200 ease-out"
                style={{
                  width: `${Math.min(100, Math.max(0, ((curve.startBps - job.bestBidBps) / (curve.startBps - floor)) * 100))}%`,
                }}
              />
            </div>
            <div className="mt-1.5 flex justify-between text-[10px] tabular-nums text-dim">
              <span>open {bps(curve.startBps)}</span>
              <span>floor {bps(floor)}</span>
            </div>
          </div>

          <ConvergenceChart
            history={source.getHistory()}
            startBps={curve.startBps}
            floorBps={floor}
            durationSeconds={curve.durationSeconds}
          />
        </section>

        {/* Right: lifecycle + accounts */}
        <aside aria-label="Settlement" className="flex min-w-0 flex-col gap-4">
          <div className="rounded-lg border border-edge bg-panel p-4">
            <h2 className="text-[10px] uppercase tracking-[0.16em] text-dim">Lifecycle</h2>
            <div className="mt-3">
              {run ? (
                <Timeline state={run} />
              ) : (
                <p className="px-2.5 py-2 text-[12px] leading-relaxed text-dim">
                  Press <span className="text-pale">Run job</span> to start bidding. The
                  auction runs for 30 seconds, then commits back to Solana.
                </p>
              )}
            </div>
            {run?.undelegateGapMs != null && (
              <p className="mt-3 border-t border-edge pt-3 text-[11px] leading-relaxed text-dim">
                Committing to L1 took{" "}
                <span className="tabular-nums text-pale">
                  {(run.undelegateGapMs / 1000).toFixed(1)}s
                </span>
                . The rollup schedules the write; the validator executes it on Solana.
              </p>
            )}
          </div>

          {run?.settleSig && (
            <div className="fade-up rounded-lg border border-accent/50 bg-accent/[0.06] p-4">
              <h2 className="text-[13px] font-semibold text-accent">Settled on Solana</h2>
              <p className="mt-1.5 text-[11px] leading-relaxed text-dim">
                Escrow paid the winning agent directly on L1.
              </p>
              <div className="mt-3 text-[11px]">
                <Address value={run.settleSig} kind="tx" label="tx" head={6} tail={6} />
              </div>
            </div>
          )}

          <div className="rounded-lg border border-edge bg-panel p-4">
            <h2 className="text-[10px] uppercase tracking-[0.16em] text-dim">Accounts</h2>
            <dl className="mt-3 space-y-2 text-[11px]">
              {[
                ["Job", activeJobPda],
                ["Escrow (L1)", (started && run.escrowPda) || dep.escrowPda],
                ["Requester", dep.requester],
              ].map(([label, value]) => (
                <div key={label} className="flex items-baseline justify-between gap-2">
                  <dt className="text-dim">{label}</dt>
                  <dd>
                    <Address value={value as string} />
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </aside>
      </div>

      <section aria-label="Bid log" className="mt-4 h-[168px] overflow-hidden rounded-lg border border-edge bg-panel/60">
        <BidLog events={source.getLog()} />
      </section>
    </main>
  );
}

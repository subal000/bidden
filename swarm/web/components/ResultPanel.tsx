"use client";

import { Trophy } from "lucide-react";
import { Address } from "./ui/Address";
import { bps, counter } from "@/lib/format";
import type { JobState } from "@/lib/types";

/**
 * What a visitor sees when they arrive after an auction has finished.
 *
 * The live widgets only carry per-agent data during a run, so a cold visitor
 * used to land on empty cards next to a large counter, which reads as broken
 * rather than complete. This states the outcome instead, entirely from the Job
 * payload plus the recovered settlement signature.
 */
export function ResultPanel({
  job,
  budgetLamports,
  settlement,
}: {
  job: JobState;
  budgetLamports: number;
  settlement: { signature: string; explorerUrl: string } | null;
}) {
  const payoutSol = (budgetLamports * job.bestBidBps) / 10_000 / 1e9;
  const savedSol = (budgetLamports * (10_000 - job.bestBidBps)) / 10_000 / 1e9;

  return (
    <section
      aria-label="Auction result"
      className="fade-up rounded-lg border border-accent/40 bg-accent/[0.05] p-5"
    >
      <div className="flex items-center gap-2">
        <Trophy className="h-4 w-4 text-accent" aria-hidden />
        <h2 className="text-[13px] font-semibold text-accent">Auction settled on Solana</h2>
      </div>

      <dl className="mt-4 grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-4">
        <Cell label="Winner" value={job.bestBidder || "—"} accent />
        <Cell label="Winning bid" value={bps(job.bestBidBps)} accent />
        <Cell label="Bids onchain" value={counter(job.bidCount)} />
        <Cell label="Paid from escrow" value={`${payoutSol.toFixed(6)} SOL`} />
      </dl>

      <p className="mt-4 text-[11px] leading-relaxed text-dim">
        Agents drove the price from 100% down to {bps(job.bestBidBps)}, saving the requester{" "}
        <span className="tabular-nums text-pale">{savedSol.toFixed(6)} SOL</span>. Every bid was
        a transaction inside the Ephemeral Rollup; the payout is one transaction on Solana L1.
      </p>

      {settlement && (
        <div className="mt-4 border-t border-accent/20 pt-3 text-[11px]">
          <Address value={settlement.signature} kind="tx" label="settlement" head={6} tail={6} />
        </div>
      )}
    </section>
  );
}

function Cell({
  label,
  value,
  accent = false,
}: {
  label: string;
  value: string;
  accent?: boolean;
}) {
  return (
    <div>
      <dt className="text-[10px] uppercase tracking-[0.14em] text-dim">{label}</dt>
      <dd
        className={[
          "mt-1 truncate text-[15px] font-semibold tabular-nums",
          accent ? "text-accent" : "text-pale",
        ].join(" ")}
      >
        {value}
      </dd>
    </div>
  );
}

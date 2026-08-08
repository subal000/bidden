"use client";

import { useEffect, useRef, useState } from "react";
import { ChevronRight } from "lucide-react";
import type { AgentView } from "@/lib/types";
import { bps } from "@/lib/format";
import { Address } from "./ui/Address";

/**
 * A real button, because it opens a detail panel. The pulse fires on the agent's
 * send counter rather than on every render, so it tracks activity not React.
 */
export function AgentCard({
  a,
  winning,
  pda,
  authority,
  floorBps,
  expanded,
  onToggle,
}: {
  a: AgentView;
  winning: boolean;
  pda?: string;
  authority?: string;
  floorBps: number;
  expanded: boolean;
  onToggle: () => void;
}) {
  const [flash, setFlash] = useState(false);
  const seen = useRef(a.pulse);

  useEffect(() => {
    if (a.pulse === seen.current) return;
    seen.current = a.pulse;
    setFlash(true);
    const t = setTimeout(() => setFlash(false), 420);
    return () => clearTimeout(t);
  }, [a.pulse]);

  return (
    <div
      className={[
        "rounded-lg border transition-colors duration-150",
        winning ? "border-accent/60 bg-accent/[0.05]" : "border-edge bg-panel",
        flash ? "pulse" : "",
      ].join(" ")}
    >
      <button
        type="button"
        onClick={onToggle}
        aria-expanded={expanded}
        className="flex w-full items-center gap-3 rounded-lg px-3.5 py-2.5 text-left transition-colors duration-100 hover:bg-raise/40 active:translate-y-px"
      >
        <span className="min-w-0 flex-1">
          <span className="flex items-baseline gap-2">
            <span className="truncate text-[14px] font-semibold text-pale">{a.spec.name}</span>
            {winning && (
              <span className="shrink-0 rounded bg-accent/15 px-1.5 py-px text-[9px] font-semibold uppercase tracking-wider text-accent">
                winning
              </span>
            )}
          </span>
          <span className="mt-0.5 block text-[10px] uppercase tracking-wider text-dim">
            {a.spec.specialization}
          </span>
        </span>

        <span className="text-right">
          <span
            className={[
              "block text-lg font-semibold leading-tight tabular-nums",
              winning ? "text-accent" : "text-pale",
            ].join(" ")}
          >
            {a.lastBidBps === null ? "—" : bps(a.lastBidBps)}
          </span>
          <span className="block text-[10px] tabular-nums text-dim">{a.sent} bids</span>
        </span>

        <ChevronRight
          className={[
            "h-4 w-4 shrink-0 text-dim transition-transform duration-150",
            expanded ? "rotate-90" : "",
          ].join(" ")}
          aria-hidden
        />
      </button>

      {expanded && (
        <dl className="row-in space-y-1.5 border-t border-edge px-3.5 py-3 text-[11px]">
          <Row label="Cost floor" value={bps(floorBps)} />
          <Row label="Reputation" value={`${(a.spec.reputation / 100).toFixed(0)} / 100`} />
          <Row label="Improved price" value={`${a.improved}×`} />
          {authority && (
            <div className="flex items-baseline justify-between gap-2">
              <dt className="text-dim">Wallet</dt>
              <dd>
                <Address value={authority} />
              </dd>
            </div>
          )}
          {pda && (
            <div className="flex items-baseline justify-between gap-2">
              <dt className="text-dim">Registry</dt>
              <dd>
                <Address value={pda} />
              </dd>
            </div>
          )}
        </dl>
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-2">
      <dt className="text-dim">{label}</dt>
      <dd className="tabular-nums text-pale">{value}</dd>
    </div>
  );
}

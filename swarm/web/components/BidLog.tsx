"use client";

import type { BidEvent } from "@/lib/types";
import { bps } from "@/lib/format";

/**
 * Scrolls faster than a person can read, which is the point: it should feel
 * like watching a market tape, not a list you audit.
 */
export function BidLog({ events }: { events: BidEvent[] }) {
  const rows = events.slice(0, 24);
  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-edge px-4 py-1.5">
        <span className="text-[10px] uppercase tracking-[0.18em] text-dim">live bid log</span>
        <span className="text-[10px] tabular-nums text-dim">{events.length} shown</span>
      </div>
      {rows.length === 0 ? (
        <div className="flex flex-1 items-center justify-center px-4 text-[12px] text-dim">
          No bids yet. Press <span className="mx-1 text-pale">Run job</span> to start the
          auction.
        </div>
      ) : (
      <div className="grid flex-1 grid-cols-1 gap-x-6 sm:grid-cols-3 xl:grid-cols-6 overflow-hidden px-4 py-2 text-[11px] leading-[1.45]">
        {rows.map((e) => (
          <div
            key={e.uid}
            className="row-in flex items-baseline justify-between gap-2 whitespace-nowrap"
          >
            <span className="tabular-nums text-dim">#{e.seq}</span>
            <span className={e.improved ? "text-accent" : "text-dim"}>{e.agent}</span>
            <span
              className={["tabular-nums", e.improved ? "font-semibold text-accent" : "text-pale/60"].join(" ")}
            >
              {bps(e.bidBps)}
            </span>
          </div>
        ))}
      </div>
      )}
    </div>
  );
}

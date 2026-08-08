"use client";

import { Check, Loader2, X, Circle } from "lucide-react";
import { Address } from "./ui/Address";
import type { RunState } from "@/lib/runner";

/** Award and the commit intent execute inside the rollup, not on Solana. */
const erStep = (name: string) => name === "Award" || name === "Commit + undelegate";

/**
 * Short metrics ("20.4s", "1048 bids") sit inline on the right. Anything longer
 * gets its own line: inline, a long detail with whitespace-nowrap forced the row
 * wider than the panel and squeezed the step name onto two lines.
 */
const INLINE_DETAIL_MAX = 14;

export function Timeline({ state }: { state: RunState }) {
  return (
    <ol className="space-y-0.5">
      {state.steps.map((s) => {
        const isActive = s.status === "active";
        const inlineDetail = !!s.detail && s.detail.length <= INLINE_DETAIL_MAX;
        return (
          <li
            key={s.name}
            className={[
              "overflow-hidden rounded-md px-2.5 py-2 transition-colors duration-150",
              isActive ? "bg-accent/[0.07]" : "",
            ].join(" ")}
          >
            <div className="flex items-center gap-2.5">
              <span className="flex h-4 w-4 shrink-0 items-center justify-center" aria-hidden>
                {s.status === "done" && <Check className="h-3.5 w-3.5 text-accent" />}
                {s.status === "active" && (
                  <Loader2 className="h-3.5 w-3.5 animate-spin text-accent" />
                )}
                {s.status === "failed" && <X className="h-3.5 w-3.5 text-hot" />}
                {s.status === "pending" && <Circle className="h-2 w-2 text-edge" />}
              </span>

              <span
                className={[
                  "flex-1 truncate whitespace-nowrap text-[12px]",
                  s.status === "pending" ? "text-dim" : "text-pale",
                  isActive ? "font-semibold text-white" : "",
                ].join(" ")}
              >
                {s.name}
                <span className="sr-only">: {s.status}</span>
              </span>

              {inlineDetail && (
                <span className="shrink-0 whitespace-nowrap text-[11px] tabular-nums text-dim">
                  {s.detail}
                </span>
              )}
            </div>

            {s.detail && !inlineDetail && (
              <p className="mt-1 pl-[26px] text-[11px] leading-snug text-dim">{s.detail}</p>
            )}

            {s.sig && (
              <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 pl-[26px] text-[11px]">
                <span
                  className={[
                    "shrink-0 rounded px-1 py-px text-[9px] uppercase tracking-wider",
                    erStep(s.name) ? "bg-accent/15 text-accent" : "bg-edge text-pale",
                  ].join(" ")}
                >
                  {erStep(s.name) ? "rollup" : "L1"}
                </span>
                <Address
                  value={s.sig}
                  kind="tx"
                  head={5}
                  tail={5}
                  layer={erStep(s.name) ? "er" : "l1"}
                />
              </div>
            )}
          </li>
        );
      })}
    </ol>
  );
}

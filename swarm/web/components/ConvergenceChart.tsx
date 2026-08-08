"use client";

import { bps } from "@/lib/format";

/**
 * The price convergence curve, which CLAUDE.md calls for explicitly: the
 * negotiation should read as a market with a visible descent over 20-30
 * seconds, not a step function. Plain inline SVG, no chart library.
 */
export function ConvergenceChart({
  history,
  startBps,
  floorBps,
  durationSeconds,
}: {
  history: { t: number; bps: number }[];
  startBps: number;
  floorBps: number;
  durationSeconds: number;
}) {
  const W = 1000;
  const H = 190;
  const padY = 10;

  const span = Math.max(1, startBps - floorBps);
  const x = (t: number) => Math.min(W, (t / durationSeconds) * W);
  const y = (v: number) => padY + (1 - (v - floorBps) / span) * (H - padY * 2);

  // Until a bid lands, every sample sits at the opening ask and the area fill
  // covers the entire panel, which reads as a rendering bug rather than an
  // auction that has not started.
  const moved = history.some((p) => p.bps < startBps);
  const pts = history.length ? history : [{ t: 0, bps: startBps }];
  const line = pts.map((p) => `${x(p.t).toFixed(1)},${y(p.bps).toFixed(1)}`).join(" ");
  const last = pts[pts.length - 1];
  const area = `0,${H} ${line} ${x(last.t).toFixed(1)},${H}`;

  return (
    <div className="flex min-h-[220px] flex-1 flex-col rounded-lg border border-edge bg-panel px-5 py-4">
      <div className="flex items-baseline justify-between">
        <span className="text-[10px] uppercase tracking-[0.18em] text-dim">
          price convergence
        </span>
        <span className="text-[10px] tabular-nums text-dim">
          {last.t.toFixed(0)}s / {durationSeconds}s
        </span>
      </div>

      <div className="relative min-h-0 flex-1">
        <svg
          viewBox={`0 0 ${W} ${H}`}
          preserveAspectRatio="none"
          className="h-full w-full"
          aria-label="best bid over time"
        >
          <defs>
            <linearGradient id="fade" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#4ade80" stopOpacity="0.28" />
              <stop offset="100%" stopColor="#4ade80" stopOpacity="0" />
            </linearGradient>
          </defs>

          {[0, 0.5, 1].map((f) => (
            <line
              key={f}
              x1="0"
              x2={W}
              y1={padY + f * (H - padY * 2)}
              y2={padY + f * (H - padY * 2)}
              stroke="#1c2431"
              strokeWidth="1"
              vectorEffect="non-scaling-stroke"
            />
          ))}

          {moved && (
            <>
              <polygon points={area} fill="url(#fade)" />
              <polyline
                points={line}
                fill="none"
                stroke="#4ade80"
                strokeWidth="2"
                vectorEffect="non-scaling-stroke"
                strokeLinejoin="round"
              />
              <circle
                cx={x(last.t)}
                cy={y(last.bps)}
                r="3.5"
                fill="#4ade80"
                vectorEffect="non-scaling-stroke"
              />
            </>
          )}
        </svg>

        {!moved && (
          <span className="pointer-events-none absolute inset-0 flex items-center justify-center text-[11px] text-dim">
            waiting for the first bid
          </span>
        )}
        <span className="pointer-events-none absolute left-0 top-0 text-[10px] tabular-nums text-dim">
          {bps(startBps)}
        </span>
        <span className="pointer-events-none absolute bottom-0 left-0 text-[10px] tabular-nums text-dim">
          {bps(floorBps)}
        </span>
      </div>
    </div>
  );
}

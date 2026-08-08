"use client";

import { useCallback, useEffect, useState } from "react";

/**
 * Pitch deck for the demo video, in the product's own visual language so cutting
 * between the deck and the app does not look like two different projects.
 *
 * Arrow keys or space advance, F for full screen, ?s=N deep-links a slide so a
 * take can restart mid-way. Deliberately absent from the nav: a recording aid,
 * not a page of the product.
 */
const SLIDES = [
  Title,
  Problem,
  Product,
  Architecture,
  Proof,
  DemoBreak,
  Scale,
  Business,
  Close,
];

export default function Slides() {
  const [i, setI] = useState(0);

  useEffect(() => {
    const s = Number(new URLSearchParams(window.location.search).get("s"));
    if (Number.isInteger(s) && s >= 1 && s <= SLIDES.length) setI(s - 1);
  }, []);

  const go = useCallback((d: number) => {
    setI((n) => Math.min(SLIDES.length - 1, Math.max(0, n + d)));
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "ArrowRight" || e.key === " ") {
        e.preventDefault();
        go(1);
      }
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        go(-1);
      }
      if (e.key.toLowerCase() === "f")
        document.documentElement.requestFullscreen?.();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [go]);

  const Slide = SLIDES[i];

  return (
    <main className="fixed inset-0 flex flex-col justify-center bg-ink px-16">
      <div key={i} className="fade-up mx-auto w-full max-w-5xl">
        <Slide />
      </div>
      <div
        className="pointer-events-none absolute bottom-6 right-8 flex gap-1"
        aria-hidden
      >
        {SLIDES.map((_, n) => (
          <span
            key={n}
            className={`h-1 w-5 rounded-full ${n === i ? "bg-accent" : "bg-edge"}`}
          />
        ))}
      </div>
    </main>
  );
}

function Eyebrow({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-[13px] uppercase tracking-[0.24em] text-accent">
      {children}
    </p>
  );
}

function H({ children }: { children: React.ReactNode }) {
  return (
    <h1 className="mt-5 text-5xl font-bold leading-[1.1] tracking-tight text-white">
      {children}
    </h1>
  );
}

function Card({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-xl border border-edge bg-panel p-6">
      <div className="text-[11px] uppercase tracking-[0.18em] text-dim">
        {label}
      </div>
      <div className="mt-4 space-y-2.5 text-[16px] leading-relaxed text-pale/85">
        {children}
      </div>
    </div>
  );
}

function Title() {
  return (
    <div className="text-center">
      <h1 className="text-8xl font-bold tracking-tight text-white">bidden</h1>
      <p className="mt-6 text-2xl text-accent">
        Agents are bidden. Then they bid.
      </p>
      <p className="mt-10 text-[16px] text-dim">
        An onchain reverse auction for autonomous agents
      </p>
    </div>
  );
}

function Problem() {
  return (
    <>
      <Eyebrow>The problem</Eyebrow>
      <H>Agents can&apos;t haggle onchain.</H>
      <div className="mt-8 grid gap-5 md:grid-cols-3">
        <Card label="What they need">
          <p>To agree a price with each other.</p>
          <p>Verifiably, so nobody has to be trusted.</p>
        </Card>
        <Card label="What that costs">
          <p>Hundreds of messages in seconds.</p>
          <p>Each one a state change others react to.</p>
        </Card>
        <Card label="What happens today">
          <p>400ms slots. A fee per message.</p>
          <p>So it happens off-chain, or not at all.</p>
        </Card>
      </div>
      <p className="mt-8 border-l-2 border-hot pl-5 text-xl font-semibold leading-snug text-white">
        Every agent marketplace today asks you to trust its matching engine.
      </p>
    </>
  );
}

function Product() {
  return (
    <>
      <Eyebrow>What we built</Eyebrow>
      <H>A live reverse auction, settled on Solana.</H>
      <div className="mt-9 space-y-4 text-[18px] leading-relaxed text-pale/85">
        <p>
          A requester posts a job and funds an escrow. Six autonomous agents are
          summoned to it, each with its own keypair, cost floor and reputation.
        </p>
        <p>
          They undercut each other for thirty seconds.{" "}
          <span className="text-white">Roughly 1,100 real onchain bids.</span>{" "}
          Lowest bid wins.
        </p>
        <p>
          The winner is paid from escrow by a single transaction on Solana.
          Every bid, and the award, is public and verifiable.
        </p>
      </div>
      <p className="mt-9 text-[15px] text-dim">
        No matching engine to trust. The auction{" "}
        <span className="text-pale">is</span> the chain.
      </p>
    </>
  );
}

function Architecture() {
  return (
    <>
      <Eyebrow>Why MagicBlock</Eyebrow>
      <H>Negotiation moves. Money does not.</H>
      <div className="mt-8 grid gap-5 md:grid-cols-2">
        <div className="rounded-xl border border-edge bg-panel p-6">
          <div className="text-[11px] uppercase tracking-[0.18em] text-dim">
            Stays on Solana L1
          </div>
          <ul className="mt-4 space-y-2.5 text-[16px] text-pale/85">
            <li>Escrow, never delegated</li>
            <li>Holds the funds start to finish</li>
            <li>One settlement transaction</li>
          </ul>
        </div>
        <div className="rounded-xl border border-accent/40 bg-accent/[0.05] p-6">
          <div className="text-[11px] uppercase tracking-[0.18em] text-accent">
            Moves to the Ephemeral Rollup
          </div>
          <ul className="mt-4 space-y-2.5 text-[16px] text-pale/85">
            <li>Job + 6 agent registries</li>
            <li>~43ms blocks · zero fees</li>
            <li>For 30 seconds, then it commits back</li>
          </ul>
        </div>
      </div>
      <p className="mt-7 text-center text-[15px] text-dim">
        delegate <span className="mx-2 text-accent">→</span> bid{" "}
        <span className="mx-2 text-accent">→</span> award{" "}
        <span className="mx-2 text-accent">→</span> commit back{" "}
        <span className="mx-2 text-accent">→</span> settle on L1
      </p>
      <p className="mt-6 text-[15px] leading-relaxed text-pale/70">
        If the rollup vanished mid-auction, the money would be exactly where it
        started. That is the difference between using a rollup and trusting one.
      </p>
    </>
  );
}

function Proof() {
  return (
    <>
      <Eyebrow>The proof</Eyebrow>
      <H>We measured it instead of assuming it.</H>
      <p className="mt-6 text-[16px] text-dim">
        Same program. Same machine. One endpoint changed. 200 transactions per
        run.
      </p>
      <div className="mt-8 grid gap-5 md:grid-cols-2">
        <div className="rounded-xl border border-edge bg-panel p-7">
          <div className="text-[11px] uppercase tracking-[0.18em] text-dim">
            On Solana L1
          </div>
          <div className="mt-3 flex items-baseline gap-2">
            <span className="text-6xl font-bold tabular-nums text-white">
              3
            </span>
            <span className="text-xl text-dim">/ 200 landed</span>
          </div>
          <p className="mt-3 text-[14px] leading-relaxed text-dim">
            Concurrency actively hurts. The burst trips the rate limiter.
          </p>
        </div>
        <div className="rounded-xl border border-accent/40 bg-accent/[0.05] p-7">
          <div className="text-[11px] uppercase tracking-[0.18em] text-accent">
            In the rollup
          </div>
          <div className="mt-3 flex items-baseline gap-2">
            <span className="text-6xl font-bold tabular-nums text-white">
              200
            </span>
            <span className="text-xl text-dim">/ 200 landed</span>
          </div>
          <p className="mt-3 text-[14px] leading-relaxed text-pale/70">
            Identical config. Zero failures. Zero fees.
          </p>
        </div>
      </div>
      <p className="mt-7 text-[15px] text-dim">
        Pushed to <span className="text-pale">758 tx/s</span> before we stopped
        looking for the ceiling. Harness and raw output are in the repo.
      </p>
    </>
  );
}

function DemoBreak() {
  return (
    <div className="text-center">
      <Eyebrow>Live on devnet</Eyebrow>
      <h1 className="mt-8 text-7xl font-bold tracking-tight text-white">
        Demo
      </h1>
      <p className="mt-8 text-xl text-pale/70">
        One job. Six agents. Thirty seconds. Settled on Solana.
      </p>
    </div>
  );
}

function Scale() {
  return (
    <>
      <Eyebrow>Where this goes</Eyebrow>
      <H>The auction is the product, not the agents.</H>
      <p className="mt-6 text-[16px] leading-relaxed text-pale/80">
        Strip the agent framing away and this is a competitive auction with
        real-time price discovery and trustless settlement. That primitive
        already has buyers.
      </p>
      <div className="mt-8 grid gap-5 md:grid-cols-3">
        <Card label="Solver auctions">
          <p>
            CoW Swap, UniswapX and intent networks run solver competitions
            off-chain.
          </p>
          <p className="text-dim">
            Because onchain was too slow. It no longer is.
          </p>
        </Card>
        <Card label="Compute markets">
          <p>
            Akash, io.net and GPU spot pricing settle onchain but price off it.
          </p>
          <p className="text-dim">Many suppliers, prices that move fast.</p>
        </Card>
        <Card label="Agent economies">
          <p>
            Agent payment rails are arriving. Price discovery between them is
            not.
          </p>
          <p className="text-dim">The bet on 2026, not the business today.</p>
        </Card>
      </div>
    </>
  );
}

function Business() {
  return (
    <>
      <Eyebrow>Business model</Eyebrow>
      <H>Sell the mechanism before the marketplace.</H>
      <div className="mt-8 grid gap-5 md:grid-cols-3">
        <div className="rounded-xl border border-accent/40 bg-accent/[0.05] p-6">
          <div className="text-[11px] uppercase tracking-[0.18em] text-accent">
            Now
          </div>
          <p className="mt-3 text-[17px] font-semibold text-white">
            License the engine
          </p>
          <p className="mt-2 text-[14px] leading-relaxed text-pale/75">
            Per-auction fee to protocols that already have supply and demand.
            Revenue before network effects, and we are a vendor rather than a
            market maker.
          </p>
        </div>
        <Card label="Next">
          <p className="text-[17px] font-semibold text-white">
            Take rate on settled volume
          </p>
          <p className="text-[14px] text-pale/75">
            Basis points on what clears. Scales with the market, worth nothing
            at zero.
          </p>
        </Card>
        <Card label="Honest risk">
          <p className="text-[17px] font-semibold text-white">
            Two-sided cold start
          </p>
          <p className="text-[14px] text-pale/75">
            No supply, no demand yet. Which is exactly why we sell to people who
            already have both.
          </p>
        </Card>
      </div>
      <p className="mt-8 text-[15px] leading-relaxed text-dim">
        The moat is not the program. It is being first to prove high-frequency
        onchain price discovery works, with a benchmark anyone can rerun.
      </p>
    </>
  );
}

function Close() {
  return (
    <div className="text-center">
      <h1 className="text-7xl font-bold tracking-tight text-white">bidden</h1>
      <p className="mt-5 text-2xl text-accent">
        Agents are bidden. Then they bid.
      </p>
      <dl className="mx-auto mt-14 grid max-w-3xl grid-cols-3 gap-8">
        {[
          ["Bids per auction", "~1,100"],
          ["On Solana L1", "3 of 200"],
          ["In the rollup", "200 of 200"],
        ].map(([k, v]) => (
          <div key={k}>
            <dt className="text-[11px] uppercase tracking-[0.16em] text-dim">
              {k}
            </dt>
            <dd className="mt-1.5 text-3xl font-bold tabular-nums text-white">
              {v}
            </dd>
          </div>
        ))}
      </dl>
      <p className="mt-14 text-[15px] text-dim">
        github.com/subal000/bidden
        <span className="mx-3 text-edge">·</span>
        Solana Blitz V7
      </p>
    </div>
  );
}

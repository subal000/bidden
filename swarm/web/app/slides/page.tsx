"use client";

import { useCallback, useEffect, useState } from "react";

/**
 * Deck for the demo video.
 *
 * Rhythm is deliberate: a sparse statement, then a stark number, then a flow,
 * then a diagram, then two enormous numbers, then a single word. Slides that all
 * share one skeleton read as filler, and the narration is doing the talking, so
 * most of these carry almost no text.
 *
 * Arrows or space advance, F for full screen, ?s=N deep-links a slide.
 */
const SLIDES = [
  Title,
  Setup,
  Cost,
  Product,
  Architecture,
  Proof,
  DemoBreak,
  Scale,
  Model,
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
    <main className="fixed inset-0 flex flex-col justify-center bg-ink px-24">
      <div key={i} className="fade-up mx-auto w-full max-w-5xl">
        <Slide />
      </div>
      <div
        className="pointer-events-none absolute bottom-7 right-9 flex gap-1"
        aria-hidden
      >
        {SLIDES.map((_, n) => (
          <span
            key={n}
            className={`h-1 w-4 rounded-full ${n === i ? "bg-accent" : "bg-edge"}`}
          />
        ))}
      </div>
    </main>
  );
}

/* 1 — wordmark, nothing else */
function Title() {
  return (
    <div>
      <h1 className="text-[110px] font-bold leading-none tracking-tight text-white">
        bidden
      </h1>
      <p className="mt-8 text-3xl text-accent">
        Agents are bidden. Then they bid.
      </p>
    </div>
  );
}

/* 2 — one sentence. Let it breathe. */
function Setup() {
  return (
    <div>
      <p className="text-6xl font-bold leading-[1.15] tracking-tight text-white">
        Two agents need to
        <br />
        agree on a price.
      </p>
      <p className="mt-10 text-2xl text-dim">
        And prove it, so nobody has to be trusted.
      </p>
    </div>
  );
}

/* 3 — the cost, as a stark contrast rather than prose */
function Cost() {
  return (
    <div>
      <p className="text-2xl text-pale/80">Agreeing takes about</p>
      <p className="mt-4 text-[130px] font-bold leading-none tracking-tight text-white">
        1,100
      </p>
      <p className="mt-4 text-2xl text-pale/80">
        bids, back and forth, in half a minute.
      </p>
      <p className="mt-14 border-l-2 border-hot pl-6 text-3xl font-semibold leading-snug text-white">
        On Solana, three of them land.
      </p>
    </div>
  );
}

/* 4 — the product as a flow strip, not paragraphs */
function Product() {
  const steps = [
    ["Post a job", "escrow funded on Solana"],
    ["Agents undercut", "six of them, 30 seconds"],
    ["Lowest wins", "nobody in the middle"],
    ["Escrow pays", "one transaction on Solana"],
  ];
  return (
    <div>
      <p className="text-[13px] uppercase tracking-[0.24em] text-accent">
        What we built
      </p>
      <h2 className="mt-5 text-5xl font-bold tracking-tight text-white">
        A reverse auction that lives on chain.
      </h2>
      <ol className="mt-14 grid grid-cols-4 gap-3">
        {steps.map(([t, s], n) => (
          <li key={t} className="border-t-2 border-edge pt-4">
            <span className="text-[11px] tabular-nums text-accent">
              0{n + 1}
            </span>
            <p className="mt-2 text-[19px] font-semibold text-white">{t}</p>
            <p className="mt-1 text-[14px] leading-snug text-dim">{s}</p>
          </li>
        ))}
      </ol>
    </div>
  );
}

/* 5 — an actual diagram */
function Architecture() {
  return (
    <div>
      <p className="text-[13px] uppercase tracking-[0.24em] text-accent">
        Why MagicBlock
      </p>
      <h2 className="mt-5 text-5xl font-bold tracking-tight text-white">
        The talking moves. The money stays.
      </h2>

      <div className="mt-14 space-y-3">
        <div className="flex items-center gap-5">
          <span className="w-28 shrink-0 text-right text-[12px] uppercase tracking-wider text-accent">
            Rollup
          </span>
          <div className="flex-1 rounded-lg border border-accent/40 bg-accent/[0.06] px-6 py-5">
            <p className="text-[19px] text-white">Job + 6 agent registries</p>
            <p className="mt-1 text-[14px] text-dim">
              30 seconds · 43ms blocks · every bid free
            </p>
          </div>
        </div>

        <div className="flex items-center gap-5">
          <span className="w-28 shrink-0" />
          <p className="flex-1 pl-6 text-[13px] text-dim">
            <span className="text-accent">↓</span> delegate, then commit back
          </p>
        </div>

        <div className="flex items-center gap-5">
          <span className="w-28 shrink-0 text-right text-[12px] uppercase tracking-wider text-dim">
            Solana L1
          </span>
          <div className="flex-1 rounded-lg border border-edge bg-panel px-6 py-5">
            <p className="text-[19px] text-white">Escrow · never delegated</p>
            <p className="mt-1 text-[14px] text-dim">Never moves. Not once.</p>
          </div>
        </div>
      </div>
    </div>
  );
}

/* 6 — two numbers, nothing to read */
function Proof() {
  return (
    <div>
      <p className="text-2xl text-pale/80">
        Same code. Same laptop. I changed one endpoint.
      </p>
      <div className="mt-12 grid grid-cols-2 gap-16">
        <div>
          <p className="text-[11px] uppercase tracking-[0.2em] text-dim">
            Solana L1
          </p>
          <p className="mt-2 text-[110px] font-bold leading-none tabular-nums text-white">
            3
          </p>
          <p className="mt-2 text-xl text-dim">of 200 landed</p>
        </div>
        <div>
          <p className="text-[11px] uppercase tracking-[0.2em] text-accent">
            Ephemeral Rollup
          </p>
          <p className="mt-2 text-[110px] font-bold leading-none tabular-nums text-accent">
            200
          </p>
          <p className="mt-2 text-xl text-dim">of 200 landed</p>
        </div>
      </div>
      <p className="mt-12 text-[15px] text-dim">
        I measured this. The harness is in the repo, you can rerun it.
      </p>
    </div>
  );
}

/* 7 — one word */
function DemoBreak() {
  return (
    <div className="text-center">
      <h2 className="text-[120px] font-bold leading-none tracking-tight text-white">
        Demo
      </h2>
    </div>
  );
}

/* 8 — a claim, then the evidence, small */
function Scale() {
  return (
    <div>
      <h2 className="text-5xl font-bold leading-[1.15] tracking-tight text-white">
        This already happens.
        <br />
        Just not on chain.
      </h2>
      <p className="mt-10 text-xl leading-relaxed text-pale/80">
        Solver auctions run competitive bidding off-chain, then settle on it.
        Because on-chain was too slow.
      </p>
      <div className="mt-12 flex gap-10 text-[15px] text-dim">
        <span>CoW Swap</span>
        <span>UniswapX</span>
        <span>Across</span>
        <span className="text-edge">·</span>
        <span>Akash</span>
        <span>io.net</span>
      </div>
      <p className="mt-12 text-2xl font-semibold text-white">
        It is not too slow any more.
      </p>
    </div>
  );
}

/* 9 — two lines and an admission */
function Model() {
  return (
    <div>
      <p className="text-[13px] uppercase tracking-[0.24em] text-accent">
        Business model
      </p>
      <h2 className="mt-5 text-5xl font-bold leading-[1.15] tracking-tight text-white">
        Sell the auction.
        <br />
        Not the marketplace.
      </h2>
      <p className="mt-10 text-xl leading-relaxed text-pale/80">
        A per-auction fee to protocols that already have buyers and sellers.
        Then a take rate on what clears, once it clears at scale.
      </p>
      <p className="mt-14 text-[15px] leading-relaxed text-dim">
        The obvious risk is a two-sided cold start. Which is exactly why the
        first customers are people who already have both sides.
      </p>
    </div>
  );
}

/* 10 — close */
function Close() {
  return (
    <div>
      <h1 className="text-[110px] font-bold leading-none tracking-tight text-white">
        bidden
      </h1>
      <p className="mt-8 text-3xl text-accent">
        Agents are bidden. Then they bid.
      </p>
      <p className="mt-16 text-[15px] text-dim">
        github.com/subal000/bidden
        <span className="mx-4 text-edge">·</span>
        Solana Blitz V7
      </p>
    </div>
  );
}

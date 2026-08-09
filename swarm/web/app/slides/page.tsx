"use client";

import { useCallback, useEffect, useState } from "react";

/**
 * Deck for the demo video.
 *
 * Structured as a business pitch: problem, why it exists, how it is solved
 * today, what we built, how, then market and model. Agents are the demo, not
 * the thesis. The thesis is that competitive bidding cannot live on chain.
 *
 * Numbers here are ones we can defend. The "3 of 200" figure from the harness
 * is a public-RPC rate limit, not a Solana capability limit, so it is not the
 * headline anywhere. Cost and reaction latency are.
 *
 * Arrows or space advance, F for full screen, ?s=N deep-links a slide.
 */
const SLIDES = [
  Title,
  Problem,
  Why,
  Today,
  Product,
  How,
  DemoBreak,
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

/* 1 */
function Title() {
  return (
    <div>
      <h1 className="text-[110px] font-bold leading-none tracking-tight text-white">
        bidden
      </h1>
      <p className="mt-8 text-3xl text-accent">
        Auctions that actually run on chain.
      </p>
    </div>
  );
}

/* 2 — the problem, stated plainly */
function Problem() {
  return (
    <div>
      <p className="text-6xl font-bold leading-[1.15] tracking-tight text-white">
        Bidding doesn&apos;t work
        <br />
        on a blockchain.
      </p>
      <p className="mt-10 max-w-3xl text-2xl leading-relaxed text-dim">
        Not NFT auctions. Not solver auctions. Not compute markets. Anywhere
        buyers compete in real time, the competition happens somewhere else.
      </p>
    </div>
  );
}

/* 3 — why, in numbers we can defend */
function Why() {
  return (
    <div>
      <p className="text-[13px] uppercase tracking-[0.24em] text-accent">Why</p>
      <h2 className="mt-5 text-5xl font-bold tracking-tight text-white">
        A bid is a transaction.
      </h2>
      <p className="mt-6 text-xl text-pale/80">
        So a thirty second auction looks like this.
      </p>

      <div className="mt-12 grid grid-cols-2 gap-16">
        <div>
          <p className="text-[11px] uppercase tracking-[0.2em] text-dim">
            On Solana
          </p>
          <p className="mt-3 text-[92px] font-bold leading-none tabular-nums text-white">
            60
          </p>
          <p className="mt-2 text-xl text-dim">rounds of bidding</p>
          <p className="mt-5 text-[15px] text-pale/70">
            $1.10 in fees, every auction
          </p>
        </div>
        <div>
          <p className="text-[11px] uppercase tracking-[0.2em] text-accent">
            In a rollup
          </p>
          <p className="mt-3 text-[92px] font-bold leading-none tabular-nums text-accent">
            698
          </p>
          <p className="mt-2 text-xl text-dim">rounds of bidding</p>
          <p className="mt-5 text-[15px] text-pale/70">
            Nothing. Bids are free.
          </p>
        </div>
      </div>

      <p className="mt-10 text-[14px] text-dim">
        Measured: 498ms median confirmation on Solana devnet, 43ms blocks in the
        rollup.
      </p>
    </div>
  );
}

/* 4 — how it is solved today, which is also the market */
function Today() {
  return (
    <div>
      <h2 className="text-5xl font-bold leading-[1.15] tracking-tight text-white">
        So everyone moved
        <br />
        the auction off chain.
      </h2>
      <p className="mt-10 max-w-3xl text-xl leading-relaxed text-pale/80">
        Solvers submit privately. An operator picks a winner. The chain only
        sees the result.
      </p>
      <div className="mt-10 flex flex-wrap gap-x-10 gap-y-3 text-[15px] text-dim">
        <span>CoW Swap</span>
        <span>UniswapX</span>
        <span>Across</span>
        <span>1inch Fusion</span>
        <span>Akash</span>
        <span>io.net</span>
      </div>
      <p className="mt-12 border-l-2 border-hot pl-6 text-2xl font-semibold leading-snug text-white">
        Billions clear this way. You trust the operator.
      </p>
    </div>
  );
}

/* 5 — what we built */
function Product() {
  const steps = [
    ["Post a job", "escrow funded on Solana"],
    ["Bidders compete", "every bid a transaction"],
    ["Lowest wins", "nobody in the middle"],
    ["Escrow pays", "one transaction on Solana"],
  ];
  return (
    <div>
      <p className="text-[13px] uppercase tracking-[0.24em] text-accent">
        What we built
      </p>
      <h2 className="mt-5 text-5xl font-bold tracking-tight text-white">
        Put the auction back on chain.
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
      <p className="mt-10 text-[15px] text-dim">
        Every bid public. No operator to trust. Settlement still on Solana.
      </p>
    </div>
  );
}

/* 6 — how */
function How() {
  return (
    <div>
      <p className="text-[13px] uppercase tracking-[0.24em] text-accent">How</p>
      <h2 className="mt-5 text-5xl font-bold tracking-tight text-white">
        The bidding moves. The money stays.
      </h2>

      <div className="mt-14 space-y-3">
        <div className="flex items-center gap-5">
          <span className="w-28 shrink-0 text-right text-[12px] uppercase tracking-wider text-accent">
            Rollup
          </span>
          <div className="flex-1 rounded-lg border border-accent/40 bg-accent/[0.06] px-6 py-5">
            <p className="text-[19px] text-white">
              The auction and every bidder
            </p>
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
            Solana
          </span>
          <div className="flex-1 rounded-lg border border-edge bg-panel px-6 py-5">
            <p className="text-[19px] text-white">Escrow · never delegated</p>
            <p className="mt-1 text-[14px] text-dim">Never moves. Not once.</p>
          </div>
        </div>
      </div>

      <p className="mt-9 text-[15px] text-dim">
        MagicBlock Ephemeral Rollups. If the rollup vanished mid-auction, the
        money is still on Solana.
      </p>
    </div>
  );
}

/* 7 */
function DemoBreak() {
  return (
    <div className="text-center">
      <h2 className="text-[120px] font-bold leading-none tracking-tight text-white">
        Demo
      </h2>
      <p className="mt-8 text-xl text-dim">
        Six agents bidding for one job, live on devnet.
      </p>
    </div>
  );
}

/* 8 — model */
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
      <p className="mt-10 max-w-3xl text-xl leading-relaxed text-pale/80">
        Charge per auction, to protocols that already have bidders. They get
        verifiable price discovery without building a rollup. Take a cut of
        volume later, once there is volume worth cutting.
      </p>
      <p className="mt-12 text-[15px] leading-relaxed text-dim">
        We are not starting a marketplace. Marketplaces need two sides. Solver
        networks already have both.
      </p>
    </div>
  );
}

/* 9 */
function Close() {
  return (
    <div>
      <h1 className="text-[110px] font-bold leading-none tracking-tight text-white">
        bidden
      </h1>
      <p className="mt-8 text-3xl text-accent">
        Auctions that actually run on chain.
      </p>
      <p className="mt-16 text-[15px] text-dim">
        github.com/subal000/bidden
        <span className="mx-4 text-edge">·</span>
        Solana Blitz V7
      </p>
    </div>
  );
}

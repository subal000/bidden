"use client";

import { useCallback, useEffect, useState } from "react";

/**
 * Presentation slides for the demo video, in the product's own visual language
 * so cutting between them and the app does not feel like two different things.
 *
 * Arrow keys or space to advance. Press F for full screen. Deliberately not in
 * the nav: this is a recording aid, not part of the product.
 */
const SLIDES = [Problem, Architecture, Close];

export default function Slides() {
  const [i, setI] = useState(0);

  // ?s=2 jumps straight to a slide, so a take can start anywhere.
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
    <main className="fixed inset-0 flex flex-col justify-center bg-ink px-20">
      <div key={i} className="fade-up mx-auto w-full max-w-5xl">
        <Slide />
      </div>
      <div
        className="pointer-events-none absolute bottom-6 right-8 flex gap-1.5"
        aria-hidden
      >
        {SLIDES.map((_, n) => (
          <span
            key={n}
            className={`h-1 w-6 rounded-full ${n === i ? "bg-accent" : "bg-edge"}`}
          />
        ))}
      </div>
    </main>
  );
}

function Problem() {
  return (
    <>
      <p className="text-[13px] uppercase tracking-[0.24em] text-accent">
        The problem
      </p>
      <h1 className="mt-6 text-6xl font-bold leading-[1.05] tracking-tight text-white">
        Agents have to negotiate.
      </h1>
      <div className="mt-10 space-y-4 text-[19px] leading-relaxed text-pale/80">
        <p>Price, terms, who does the work.</p>
        <p>
          For that to be trustworthy it has to happen onchain, where anyone can
          verify it.
        </p>
        <p>But negotiation is chatty. Hundreds of messages in seconds.</p>
      </div>
      <p className="mt-10 border-l-2 border-hot pl-5 text-2xl font-semibold leading-snug text-white">
        400ms slots and a fee per message.
        <br />
        Economically dead on a base layer.
      </p>
    </>
  );
}

function Architecture() {
  return (
    <>
      <p className="text-[13px] uppercase tracking-[0.24em] text-accent">
        How it works
      </p>
      <h1 className="mt-5 text-5xl font-bold tracking-tight text-white">
        Negotiation moves. Money does not.
      </h1>

      <div className="mt-10 grid gap-5 md:grid-cols-2">
        <div className="rounded-xl border border-edge bg-panel p-7">
          <div className="text-[12px] uppercase tracking-[0.18em] text-dim">
            Solana L1
          </div>
          <ul className="mt-5 space-y-3 text-[17px] text-pale/85">
            <li>Escrow, never delegated</li>
            <li>Holds the funds start to finish</li>
            <li>One settlement transaction</li>
          </ul>
        </div>
        <div className="rounded-xl border border-accent/40 bg-accent/[0.05] p-7">
          <div className="text-[12px] uppercase tracking-[0.18em] text-accent">
            Ephemeral Rollup
          </div>
          <ul className="mt-5 space-y-3 text-[17px] text-pale/85">
            <li>Job + 6 agent registries</li>
            <li>~1,100 bids in 30 seconds</li>
            <li>~43ms blocks · zero fees</li>
          </ul>
        </div>
      </div>

      <p className="mt-8 text-center text-[15px] text-dim">
        delegate <span className="mx-2 text-accent">→</span> 30 seconds of
        bidding
        <span className="mx-2 text-accent">→</span> commit back to Solana
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
      <dl className="mx-auto mt-14 grid max-w-2xl grid-cols-3 gap-8 text-left">
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

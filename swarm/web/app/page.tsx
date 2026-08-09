import Link from "next/link";
import { ArrowRight, Lock, Zap, Layers } from "lucide-react";

export default function Home() {
  return (
    <main className="mx-auto w-full max-w-6xl overflow-x-hidden px-4 sm:px-6">
      <section className="fade-up py-20 sm:py-28">
        <h1 className="max-w-3xl text-4xl font-bold leading-[1.1] tracking-tight text-white sm:text-5xl">
          Agents are bidden. Then they bid.
        </h1>
        <p className="mt-5 max-w-2xl text-[15px] leading-relaxed text-pale/80">
          A job is posted and six autonomous agents are summoned to it. They undercut each
          other for thirty seconds, and every bid is a real Solana transaction. The auction
          runs inside a MagicBlock Ephemeral Rollup; the escrow never leaves Solana L1.
        </p>

        <div className="mt-8 flex flex-wrap items-center gap-3">
          <Link
            href="/demo"
            className="inline-flex min-h-[44px] items-center gap-2 rounded-md bg-accent px-5 text-[13px] font-semibold text-ink transition-[background-color,transform] duration-100 ease-out hover:bg-accent/85 active:translate-y-px"
          >
            Run a live auction
            <ArrowRight className="h-4 w-4" aria-hidden />
          </Link>
          <Link
            href="/benchmark"
            className="inline-flex min-h-[44px] items-center rounded-md border border-edge px-5 text-[13px] font-semibold text-pale transition-[border-color,background-color] duration-100 hover:border-dim hover:bg-raise"
          >
            See the measurements
          </Link>
        </div>
      </section>

      <section className="fade-up border-t border-edge py-16" style={{ animationDelay: "60ms" }}>
        <h2 className="text-[11px] uppercase tracking-[0.16em] text-dim">
          The same program, the same machine, one endpoint changed
        </h2>
        <div className="mt-6 grid gap-4 sm:grid-cols-2">
          <div className="rounded-lg border border-edge bg-panel p-6">
            <div className="text-[11px] uppercase tracking-wider text-dim">On Solana L1</div>
            <div className="mt-2 flex items-baseline gap-2">
              <span className="text-5xl font-bold tabular-nums text-white">3</span>
              <span className="text-lg text-dim">/ 200 landed</span>
            </div>
            <p className="mt-3 text-[13px] leading-relaxed text-dim">
              Six concurrent bidders. Concurrency actively hurts: the burst trips the public
              RPC limiter and 197 transactions are rejected before reaching consensus.
            </p>
          </div>
          <div className="rounded-lg border border-accent/40 bg-accent/[0.05] p-6">
            <div className="text-[11px] uppercase tracking-wider text-accent">
              In the Ephemeral Rollup
            </div>
            <div className="mt-2 flex items-baseline gap-2">
              <span className="text-5xl font-bold tabular-nums text-white">200</span>
              <span className="text-lg text-dim">/ 200 landed</span>
            </div>
            <p className="mt-3 text-[13px] leading-relaxed text-pale/70">
              Identical configuration, 26 to 38 per second, zero failures, zero fees. We
              pushed the harness to 758 tx/s before we stopped looking for the ceiling.
            </p>
          </div>
        </div>
        <p className="mt-4 text-[12px] text-dim">
          Measured on devnet, not estimated. Full method and raw output on the{" "}
          <Link
            href="/benchmark"
            className="text-pale underline decoration-edge underline-offset-2 transition-colors duration-100 hover:decoration-dim"
          >
            benchmark page
          </Link>
          .
        </p>
      </section>

      <section className="fade-up border-t border-edge py-16" style={{ animationDelay: "120ms" }}>
        <h2 className="text-2xl font-semibold tracking-tight text-white">
          Why agents cannot negotiate on L1
        </h2>
        <p className="mt-3 max-w-2xl text-[14px] leading-relaxed text-pale/70">
          Negotiation is a high-frequency workload. It needs hundreds of messages in seconds,
          and each one has to be a state change the other agents can see and react to. At
          400ms slots with a fee per message, that is economically dead.
        </p>

        <div className="mt-8 grid gap-4 md:grid-cols-3">
          {[
            {
              icon: Zap,
              title: "Bidding moves to the rollup",
              body: "The Job and each agent's registry are delegated into an Ephemeral Rollup for the thirty seconds bidding takes. Blocks land in about 43ms and transactions are free.",
            },
            {
              icon: Lock,
              title: "The money never moves",
              body: "The escrow account is never delegated. It sits on Solana L1 for the entire auction. If the rollup vanished mid-auction the funds are still exactly where they were.",
            },
            {
              icon: Layers,
              title: "Settlement returns to L1",
              body: "When bidding closes, state commits back to Solana and a single settlement transaction pays the winning agent directly from escrow.",
            },
          ].map(({ icon: Icon, title, body }) => (
            <div key={title} className="rounded-lg border border-edge bg-panel p-6">
              <Icon className="h-5 w-5 text-accent" aria-hidden />
              <h3 className="mt-4 text-[15px] font-semibold text-white">{title}</h3>
              <p className="mt-2 text-[13px] leading-relaxed text-dim">{body}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="fade-up border-t border-edge py-16" style={{ animationDelay: "180ms" }}>
        <h2 className="text-2xl font-semibold tracking-tight text-white">One job, end to end</h2>
        <ol className="mt-8 space-y-px">
          {(
            [
              ["post_job", "L1", "Requester funds an escrow on Solana."],
              ["delegate", "L1", "Job and six agent registries move into the rollup, one transaction each."],
              ["submit_bid", "Rollup", "Agents undercut each other. Roughly 1,000 bids in thirty seconds."],
              ["award_job", "Rollup", "Lowest bid wins. The winner's registry records the job."],
              ["commit_and_undelegate", "Rollup", "Schedules the commit back to L1. Asynchronous, about twenty seconds."],
              ["settle", "L1", "Escrow pays the winner. One signature, permanently on Solana."],
            ] as const
          ).map(([name, layer, body], i) => (
            <li
              key={name}
              className="flex flex-col gap-1 border-b border-edge py-4 sm:flex-row sm:items-baseline sm:gap-6"
            >
              <span className="w-6 shrink-0 text-[11px] tabular-nums text-dim">{i + 1}</span>
              <code className="w-52 shrink-0 text-[13px] text-accent">{name}</code>
              <span
                className={[
                  "w-16 shrink-0 rounded px-1.5 py-0.5 text-center text-[10px] uppercase tracking-wider",
                  layer === "L1" ? "bg-edge text-pale" : "bg-accent/15 text-accent",
                ].join(" ")}
              >
                {layer}
              </span>
              <span className="text-[13px] leading-relaxed text-dim">{body}</span>
            </li>
          ))}
        </ol>
      </section>

      <section className="fade-up border-t border-edge py-16" style={{ animationDelay: "240ms" }}>
        <div className="rounded-lg border border-edge bg-panel p-8">
          <h2 className="text-xl font-semibold tracking-tight text-white">
            Watch one run start to finish
          </h2>
          <p className="mt-2 max-w-xl text-[13px] leading-relaxed text-dim">
            Post a job, watch six agents drive the price down live, then follow the
            settlement onto Solana and open it on the explorer.
          </p>
          <Link
            href="/demo"
            className="mt-6 inline-flex min-h-[44px] items-center gap-2 rounded-md bg-accent px-5 text-[13px] font-semibold text-ink transition-[background-color,transform] duration-100 ease-out hover:bg-accent/85 active:translate-y-px"
          >
            Open the live demo
            <ArrowRight className="h-4 w-4" aria-hidden />
          </Link>
        </div>
      </section>

      <footer className="border-t border-edge py-8 text-[11px] text-dim">
        Bidden · built for Solana Blitz V7. Every number shown is measured on devnet.
      </footer>
    </main>
  );
}

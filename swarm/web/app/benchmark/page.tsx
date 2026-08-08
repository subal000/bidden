import { Address } from "@/components/ui/Address";

const COMPARISON = [
  ["Sequential, await confirm", "16 / 200", "0.5", "200 / 200", "4.2"],
  ["Sequential, fire and forget", "20 / 200", "0.8", "200 / 200", "8.1 – 12.1"],
  ["6 concurrent + 50-100ms jitter", "3 / 200", "0.5", "200 / 200", "26 – 38"],
];

const CEILING = [
  ["6", "200", "200 / 200", "70.6"],
  ["12", "400", "400 / 400", "116.9"],
  ["24", "2,000", "2,000 / 2,000", "228.5"],
  ["96", "2,000", "2,000 / 2,000", "770.2"],
  ["96", "4,000", "4,000 / 4,000", "758.3"],
];

const LIFECYCLE = [
  ["Initialize on L1", "4z4SAu5uQpCJX9KYfonC7cZjHyhKTht6qoGAWV8HTWpBY9tRLQi3RmSCxumtRwtMTjrYxFAbnZeqvzuRNFgFADp8"],
  ["Delegate — ownership moves to the delegation program", "3ar1DAAjzTUjBokkqBvn15saLphPqcBpvUuYq8d8cqs6ai87sNchcptmTzC6Mow5kXWhfmkMmFpWjcdThvW3pwf4"],
  ["ProcessUndelegation — validator writes committed state back to L1", "3Xqxsz989EFoNmeeU4AVXCaDxNfoq6NL1q8oWrEeqriB3ry4BvvtsaCMhempnkWCZsZ42HdzJP3m4o9kQguC9hSA"],
];

export default function Benchmark() {
  return (
    <main className="mx-auto w-full max-w-5xl overflow-x-hidden px-4 py-12 sm:px-6">
      <h1 className="text-3xl font-bold tracking-tight text-white">Measurements</h1>
      <p className="mt-3 max-w-2xl text-[14px] leading-relaxed text-pale/75">
        Every number here was measured on one machine against Solana devnet and the
        <code className="mx-1 text-accent">devnet-as</code> Ephemeral Rollup endpoint.
        Nothing is estimated. Where something was not tested, it says so.
      </p>

      <Section title="Ephemeral Rollup versus Solana L1">
        <p className="mb-5 text-[13px] leading-relaxed text-dim">
          Same program, same machine, one endpoint swap. 200 transactions per configuration.
          Landed counts are read from the counter account afterwards, never from what the
          client believed it sent.
        </p>
        <Table
          head={["Configuration", "L1 landed", "L1 tx/s", "ER landed", "ER tx/s"]}
          rows={COMPARISON}
          emphasiseLast
        />
        <p className="mt-4 text-[13px] leading-relaxed text-pale/75">
          <strong className="text-white">Concurrency actively hurts on L1.</strong> The
          strategy producing 26-38 tx/s in the rollup lands three transactions out of two
          hundred on devnet, because the burst trips the public RPC limiter immediately.
        </p>
        <p className="mt-3 text-[12px] leading-relaxed text-dim">
          Being precise, since this is an infrastructure limit and not a claim about Solana
          consensus: public devnet caps sendTransaction near 40 per 10 seconds. L1 never
          dropped anything it accepted, in any of the four runs. Paced under the limit it
          lands 40/40 with confirm latency median 498ms, which is the 400ms slot time
          appearing where it should, plus 5,000 lamports per transaction where the rollup
          charges zero.
        </p>
      </Section>

      <Section title="Throughput ceiling">
        <Table head={["Senders", "Sent", "Landed", "tx/s"]} rows={CEILING} />
        <p className="mt-4 text-[13px] text-dim">
          Ceiling not found. 100% landed at 758 tx/s. The constraint is client-side
          concurrency and round-trip latency, not the rollup.
        </p>
      </Section>

      <Section title="Other measured values">
        <dl className="grid gap-x-8 gap-y-3 sm:grid-cols-2">
          {[
            ["ER block rate", "22-23 blocks/s (~43ms), idle and under load"],
            ["Transactions per block", "~33"],
            ["Blockhash window", "1,198 blocks, ~54s"],
            ["Undelegate to L1 gap", "19.97s for seven accounts"],
            ["Failed transactions", "0 across 13,405 sent"],
            ["Websocket delivery", "100% at 31 tx/s, 93.2% at 724 tx/s"],
          ].map(([k, v]) => (
            <div key={k} className="border-b border-edge pb-3">
              <dt className="text-[11px] uppercase tracking-wider text-dim">{k}</dt>
              <dd className="mt-1 text-[13px] tabular-nums text-pale">{v}</dd>
            </div>
          ))}
        </dl>
        <p className="mt-5 rounded-lg border border-warn/30 bg-warn/[0.05] p-4 text-[12px] leading-relaxed text-pale/80">
          <strong className="text-warn">On the 1ms figure.</strong> MagicBlock&apos;s
          marketing cites 1ms block time. This project measured 22-23 blocks per second on
          devnet-as, idle and under 758 tx/s of load, so it does not use that number. The
          throughput comes from packing ~33 transactions per block, not from 1ms blocks.
        </p>
      </Section>

      <Section title="Independently verifiable on the explorer">
        <p className="mb-5 text-[13px] leading-relaxed text-dim">
          Rollup transactions are not on any public explorer, so the throughput figures above
          are our measurement, evidenced by the harness. The delegation lifecycle is entirely
          L1-visible and clickable.
        </p>
        <ul className="space-y-3">
          {LIFECYCLE.map(([label, sig]) => (
            <li key={sig} className="border-b border-edge pb-3">
              <div className="text-[12px] text-pale">{label}</div>
              <div className="mt-1.5 text-[11px]">
                <Address value={sig} kind="tx" head={8} tail={8} />
              </div>
            </li>
          ))}
        </ul>
      </Section>
    </main>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="mt-12 border-t border-edge pt-8">
      <h2 className="mb-5 text-lg font-semibold tracking-tight text-white">{title}</h2>
      {children}
    </section>
  );
}

function Table({
  head,
  rows,
  emphasiseLast = false,
}: {
  head: string[];
  rows: string[][];
  emphasiseLast?: boolean;
}) {
  return (
    <div className="overflow-x-auto rounded-lg border border-edge">
      <table className="w-full text-[12px]">
        <thead>
          <tr className="border-b border-edge bg-panel">
            {head.map((h, i) => (
              <th
                key={h}
                scope="col"
                className={`px-4 py-2.5 font-medium uppercase tracking-wider text-dim ${i === 0 ? "text-left" : "text-right"}`}
              >
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, ri) => {
            const hot = emphasiseLast && ri === rows.length - 1;
            return (
              <tr
                key={`${ri}-${r[0]}`}
                className={`border-b border-edge/60 last:border-0 transition-colors duration-100 hover:bg-raise/40 ${hot ? "bg-accent/[0.04]" : ""}`}
              >
                {r.map((c, ci) => (
                  <td
                    key={ci}
                    className={[
                      "px-4 py-2.5",
                      ci === 0 ? "text-left text-pale" : "text-right tabular-nums",
                      ci > 0 && hot ? "font-semibold text-white" : ci > 0 ? "text-pale/80" : "",
                    ].join(" ")}
                  >
                    {c}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

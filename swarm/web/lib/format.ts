/** Bids are basis points of the escrow budget. 10000 bps is the full budget. */
export function bps(v: number): string {
  return `${(v / 100).toFixed(2)}%`;
}

export function counter(n: number): string {
  return n.toLocaleString("en-US");
}

export function shortKey(k: string): string {
  if (!k || k.length < 12) return k || "—";
  return `${k.slice(0, 4)}…${k.slice(-4)}`;
}

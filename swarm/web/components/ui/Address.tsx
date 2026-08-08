"use client";

import { useState } from "react";
import { Check, Copy, ExternalLink } from "lucide-react";

/** Which chain a signature or account lives on. */
export type Layer = "l1" | "er";

const ER_RPC = "https://devnet-as.magicblock.app";

/**
 * Rollup transactions are not on Solana, so a plain devnet explorer link 404s.
 * Solana Explorer accepts a custom RPC, and the ER endpoint both retains
 * transaction history and sends `access-control-allow-origin: *`, so pointing
 * the explorer at it resolves them properly.
 */
export function explorerUrl(
  value: string,
  kind: "tx" | "address" = "address",
  layer: Layer = "l1",
) {
  if (layer === "er") {
    return `https://explorer.solana.com/${kind}/${value}?cluster=custom&customUrl=${encodeURIComponent(ER_RPC)}`;
  }
  return `https://explorer.solana.com/${kind}/${value}?cluster=devnet`;
}

export function truncate(k: string, head = 4, tail = 4) {
  if (!k) return "—";
  if (k.length <= head + tail + 1) return k;
  return `${k.slice(0, head)}…${k.slice(-tail)}`;
}

/**
 * A pubkey or signature: truncated for scanning, copyable, and openable.
 *
 * Truncation is a reading aid, so the full value stays recoverable via copy and
 * via the title attribute. Explorer links carry the cluster, because a devnet
 * build pointing at mainnet is an immediate tell.
 */
export function Address({
  value,
  kind = "address",
  label,
  head = 4,
  tail = 4,
  layer = "l1",
}: {
  value: string;
  kind?: "tx" | "address";
  label?: string;
  head?: number;
  tail?: number;
  layer?: Layer;
}) {
  const [copied, setCopied] = useState(false);

  if (!value) return <span className="text-dim">—</span>;

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      // Clipboard can be blocked; the title attribute still exposes the value.
    }
  };

  return (
    <span className="inline-flex items-center gap-1.5">
      {label && <span className="text-dim">{label}</span>}
      <code className="text-pale/90" title={value}>
        {truncate(value, head, tail)}
      </code>
      <button
        type="button"
        onClick={copy}
        aria-label={copied ? "Copied to clipboard" : `Copy ${label ?? "address"}`}
        className="inline-flex h-6 w-6 items-center justify-center rounded text-dim transition-colors duration-100 hover:bg-edge hover:text-pale active:translate-y-px"
      >
        {copied ? (
          <Check className="h-3 w-3 text-accent" aria-hidden />
        ) : (
          <Copy className="h-3 w-3" aria-hidden />
        )}
      </button>
      <a
        href={explorerUrl(value, kind, layer)}
        target="_blank"
        rel="noreferrer"
        aria-label={
          layer === "er"
            ? "Open in Solana Explorer via the rollup RPC"
            : "Open in Solana Explorer"
        }
        className="inline-flex h-6 w-6 items-center justify-center rounded text-dim transition-colors duration-100 hover:bg-edge hover:text-pale"
      >
        <ExternalLink className="h-3 w-3" aria-hidden />
      </a>
    </span>
  );
}

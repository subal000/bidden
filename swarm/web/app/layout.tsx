import type { Metadata } from "next";
import { Nav } from "@/components/Nav";
import "./globals.css";

export const metadata: Metadata = {
  title: "Bidden — agents are bidden, then they bid",
  description:
    "An onchain reverse auction where autonomous agents undercut each other in real time. Bidding runs on a MagicBlock Ephemeral Rollup; escrow never leaves Solana L1.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body className="min-h-screen bg-ink font-mono antialiased">
        <a
          href="#content"
          className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded focus:bg-accent focus:px-3 focus:py-2 focus:text-[13px] focus:font-semibold focus:text-ink"
        >
          Skip to content
        </a>
        <Nav />
        <div id="content">{children}</div>
      </body>
    </html>
  );
}

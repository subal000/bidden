"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Code2 } from "lucide-react";

const LINKS = [
  { href: "/", label: "Overview" },
  { href: "/demo", label: "Live demo" },
  { href: "/benchmark", label: "Benchmark" },
];

export function Nav() {
  const pathname = usePathname();
  return (
    <header className="sticky top-0 z-50 border-b border-edge/80 bg-ink/85 backdrop-blur-md">
      <nav
        aria-label="Main"
        className="mx-auto flex h-14 max-w-6xl items-center justify-between gap-2 px-4 sm:px-6"
      >
        <Link
          href="/"
          className="flex items-baseline gap-2.5 rounded transition-opacity duration-100 hover:opacity-80"
        >
          <span className="shrink-0 text-base font-bold tracking-tight text-white">bidden</span>
          <span className="hidden text-[11px] text-dim sm:inline">
            agents are bidden, then they bid
          </span>
        </Link>

        <div className="flex min-w-0 shrink items-center gap-0.5 overflow-x-auto sm:gap-1">
          {LINKS.map((l) => {
            const active = pathname === l.href;
            return (
              <Link
                key={l.href}
                href={l.href}
                aria-current={active ? "page" : undefined}
                className={[
                  "shrink-0 whitespace-nowrap rounded-md px-2 py-2 text-[12px] transition-colors duration-100 sm:px-3 sm:text-[13px]",
                  active ? "bg-raise text-white" : "text-dim hover:bg-raise/60 hover:text-pale",
                ].join(" ")}
              >
                {l.label}
              </Link>
            );
          })}
          <a
            href="https://github.com"
            target="_blank"
            rel="noreferrer"
            aria-label="View source"
            className="ml-1 inline-flex h-10 w-10 items-center justify-center rounded-md text-dim transition-colors duration-100 hover:bg-raise/60 hover:text-pale"
          >
            <Code2 className="h-4 w-4" aria-hidden />
          </a>
        </div>
      </nav>
    </header>
  );
}

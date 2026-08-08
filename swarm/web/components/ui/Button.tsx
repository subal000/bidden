"use client";

import { Loader2 } from "lucide-react";
import type { ButtonHTMLAttributes, ReactNode } from "react";

type Variant = "primary" | "ghost";

/**
 * All six states are treated as one system: resting, hover, focus, pressed,
 * disabled, loading. Loading keeps the label visible so the layout never shifts.
 */
export function Button({
  children,
  variant = "primary",
  loading = false,
  icon,
  className = "",
  disabled,
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant;
  loading?: boolean;
  icon?: ReactNode;
}) {
  const base =
    "inline-flex min-h-[40px] items-center justify-center gap-2 rounded-md px-4 text-[13px] font-semibold " +
    "transition-[background-color,border-color,color,transform] duration-100 ease-out " +
    "active:translate-y-px disabled:pointer-events-none disabled:opacity-50";

  const variants: Record<Variant, string> = {
    primary: "bg-accent text-ink hover:bg-accent/85",
    ghost: "border border-edge text-pale hover:border-dim hover:bg-raise",
  };

  return (
    <button
      {...rest}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={`${base} ${variants[variant]} ${className}`}
    >
      {loading ? (
        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden />
      ) : (
        icon
      )}
      {children}
    </button>
  );
}

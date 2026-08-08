/** Label above value, baseline aligned, tabular so digits never jitter. */
export function Stat({
  label,
  value,
  hint,
  tone = "default",
}: {
  label: string;
  value: string;
  hint?: string;
  tone?: "default" | "accent" | "dim";
}) {
  const toneClass =
    tone === "accent" ? "text-accent" : tone === "dim" ? "text-dim" : "text-pale";
  return (
    <div>
      <div className="text-[10px] uppercase tracking-[0.16em] text-dim">{label}</div>
      <div className={`mt-1 text-xl font-semibold tabular-nums ${toneClass}`}>{value}</div>
      {hint && <div className="mt-0.5 text-[11px] text-dim">{hint}</div>}
    </div>
  );
}

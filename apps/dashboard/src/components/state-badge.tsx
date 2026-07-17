import clsx from "clsx";

// State is never color-alone: each badge pairs a symbol + label with the
// status color (dataviz rule).
const styles: Record<string, { cls: string; dot: string; label: string }> = {
  up: { cls: "border-node/40 text-node", dot: "●", label: "up" },
  down: { cls: "border-alert/50 text-alert", dot: "▲", label: "down" },
  unknown: { cls: "border-ink-line text-mist-faint", dot: "○", label: "pending" },
};

export function StateBadge({ state }: { state: string }) {
  const s = styles[state] ?? styles.unknown;
  return (
    <span
      className={clsx(
        "inline-flex items-center gap-1.5 rounded-full border bg-ink-soft px-2.5 py-0.5 font-mono text-xs",
        s.cls,
      )}
    >
      <span aria-hidden>{s.dot}</span>
      {s.label}
    </span>
  );
}

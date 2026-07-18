"use client";

import type { CheckResult } from "@/lib/types";
import { fmtTime } from "@/lib/format";

// Sparkline of recent checks, oldest → newest: one thin bar per check, teal
// ok / red fail (status colors; the row's StateBadge carries the textual
// state, and each bar has a native title tooltip, so state is never
// color-alone).
export function Sparkline({ results, max = 40 }: { results: CheckResult[]; max?: number }) {
  // API returns newest-first; show the most recent `max`, oldest on the left.
  const recent = results.slice(0, max).reverse();
  if (recent.length === 0) {
    return <span className="font-mono text-xs text-mist-faint">no checks yet</span>;
  }

  const w = 3;
  const gap = 2; // ≥2px surface gap between adjacent marks
  const h = 20;
  const width = recent.length * (w + gap) - gap;

  return (
    <svg
      width={width}
      height={h}
      viewBox={`0 0 ${width} ${h}`}
      role="img"
      aria-label={`Last ${recent.length} checks`}
      className="shrink-0"
    >
      {recent.map((r, i) => (
        <rect
          key={r.id}
          x={i * (w + gap)}
          y={r.ok ? 4 : 0}
          width={w}
          height={r.ok ? h - 4 : h}
          rx={1.5}
          className={r.ok ? "fill-node/80" : "fill-alert"}
        >
          <title>
            {`${fmtTime(r.checked_at)} — ${r.ok ? "ok" : "failed"}${
              r.latency_ms !== null ? `, ${r.latency_ms}ms` : ""
            }${r.error ? ` (${r.error})` : ""}`}
          </title>
        </rect>
      ))}
    </svg>
  );
}

"use client";

import type { SeriesPoint } from "@/lib/types";
import { fmtTime } from "@/lib/format";

// Sparkline of recent history, oldest → newest: one thin bar per time bucket,
// teal when every check in the bucket passed, red when any failed. Reads
// aggregated buckets rather than raw rows — a 24h window used to mean fetching
// ~1,440 results per monitor to draw 40 bars.
//
// Each bar carries a title tooltip and the row's StateBadge states the current
// condition in words, so nothing here is conveyed by colour alone.
export function Sparkline({ buckets }: { buckets: SeriesPoint[] }) {
  if (buckets.length === 0) {
    return <span className="font-mono text-xs text-mist-faint">no checks yet</span>;
  }

  const w = 3;
  const gap = 2; // ≥2px surface gap between adjacent marks
  const h = 20;
  const width = buckets.length * (w + gap) - gap;

  return (
    <svg
      width={width}
      height={h}
      viewBox={`0 0 ${width} ${h}`}
      role="img"
      aria-label={`Recent checks across ${buckets.length} intervals`}
      className="shrink-0"
    >
      {buckets.map((b, i) => {
        const failed = b.fail > 0;
        return (
          <rect
            key={b.t}
            x={i * (w + gap)}
            y={failed ? 0 : 4}
            width={w}
            height={failed ? h : h - 4}
            rx={1.5}
            className={failed ? "fill-alert" : "fill-node/80"}
          >
            <title>
              {`${fmtTime(b.t)} — ${b.ok} ok${failed ? `, ${b.fail} failed` : ""}${
                b.avg_ms !== null ? `, ~${b.avg_ms}ms` : ""
              }`}
            </title>
          </rect>
        );
      })}
    </svg>
  );
}

"use client";

import { useMemo, useRef, useState } from "react";
import type { SeriesPoint } from "@/lib/types";
import { fmtTime } from "@/lib/format";

type Point = { x: number; y: number; b: SeriesPoint };

// Average response time per bucket (teal, 2px), with buckets containing a
// failure marked in red at the top of the plot. Single series → no legend box;
// the card title names it. Hover gives a crosshair + tooltip. Grid is
// recessive.
//
// Consumes aggregated buckets, so the payload is a fixed ~120 points whether
// the range is an hour or a month.
export function LatencyChart({ buckets }: { buckets: SeriesPoint[] }) {
  const [hover, setHover] = useState<Point | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  const W = 720;
  const H = 180;
  const PAD = { top: 12, right: 12, bottom: 24, left: 48 };

  const { points, failures, ticks, yMax } = useMemo(() => {
    const withLatency = buckets.filter((b) => b.avg_ms !== null);
    const rawMax = Math.max(100, ...withLatency.map((b) => b.avg_ms ?? 0));
    const yMax = Math.ceil(rawMax / 100) * 100;
    const innerW = W - PAD.left - PAD.right;
    const innerH = H - PAD.top - PAD.bottom;

    // x is positioned by index across all buckets so gaps in latency (an
    // all-failed bucket) don't distort the time axis.
    const xAt = (i: number) =>
      PAD.left + (buckets.length === 1 ? innerW / 2 : (i / (buckets.length - 1)) * innerW);

    const points: Point[] = [];
    const failures: Point[] = [];
    buckets.forEach((b, i) => {
      const x = xAt(i);
      if (b.avg_ms !== null) {
        points.push({ x, y: PAD.top + innerH - (b.avg_ms / yMax) * innerH, b });
      }
      if (b.fail > 0) {
        failures.push({ x, y: PAD.top + 4, b });
      }
    });

    const ticks = [0, 0.5, 1].map((f) => ({
      y: PAD.top + innerH - f * innerH,
      label: `${Math.round(f * yMax)}ms`,
    }));
    return { points, failures, ticks, yMax };
  }, [buckets]);

  if (buckets.length === 0) {
    return (
      <div className="flex h-44 items-center justify-center font-mono text-sm text-mist-faint">
        no data in this range
      </div>
    );
  }

  const path = points
    .map((p, i) => `${i === 0 ? "M" : "L"}${p.x.toFixed(1)},${p.y.toFixed(1)}`)
    .join(" ");

  function onMove(e: React.MouseEvent<SVGSVGElement>) {
    const rect = svgRef.current?.getBoundingClientRect();
    if (!rect || points.length === 0) return;
    const x = ((e.clientX - rect.left) / rect.width) * W;
    let best = points[0];
    for (const p of points) {
      if (Math.abs(p.x - x) < Math.abs(best.x - x)) best = p;
    }
    setHover(best);
  }

  return (
    <div className="relative">
      <svg
        ref={svgRef}
        viewBox={`0 0 ${W} ${H}`}
        className="w-full"
        role="img"
        aria-label={`Average response time, peak ${yMax}ms`}
        onMouseMove={onMove}
        onMouseLeave={() => setHover(null)}
      >
        {ticks.map((t) => (
          <g key={t.y}>
            <line
              x1={PAD.left}
              x2={W - PAD.right}
              y1={t.y}
              y2={t.y}
              className="stroke-ink-line"
              strokeWidth={1}
            />
            <text
              x={PAD.left - 8}
              y={t.y + 3}
              textAnchor="end"
              className="fill-mist-faint font-mono text-[10px]"
            >
              {t.label}
            </text>
          </g>
        ))}
        <text x={PAD.left} y={H - 6} className="fill-mist-faint font-mono text-[10px]">
          {fmtTime(buckets[0].t)}
        </text>
        <text
          x={W - PAD.right}
          y={H - 6}
          textAnchor="end"
          className="fill-mist-faint font-mono text-[10px]"
        >
          {fmtTime(buckets[buckets.length - 1].t)}
        </text>

        {hover && (
          <line
            x1={hover.x}
            x2={hover.x}
            y1={PAD.top}
            y2={H - PAD.bottom}
            className="stroke-mist-faint"
            strokeWidth={1}
            strokeDasharray="3 3"
          />
        )}

        {points.length > 1 && (
          <path d={path} fill="none" className="stroke-node" strokeWidth={2} />
        )}

        {/* Buckets containing a failure, pinned to the top with a surface ring. */}
        {failures.map((p) => (
          <circle
            key={`f-${p.b.t}`}
            cx={p.x}
            cy={p.y}
            r={4}
            className="fill-alert stroke-ink-card"
            strokeWidth={2}
          />
        ))}

        {hover && (
          <circle cx={hover.x} cy={hover.y} r={4.5} className="fill-node stroke-ink" strokeWidth={2} />
        )}
      </svg>

      {hover && (
        <div
          className="pointer-events-none absolute -top-2 z-10 -translate-x-1/2 -translate-y-full rounded-md border border-ink-line bg-ink-soft px-3 py-2 font-mono text-xs shadow-lg"
          style={{ left: `${(hover.x / W) * 100}%` }}
        >
          <div className="text-mist">{fmtTime(hover.b.t)}</div>
          <div className="text-node">
            {hover.b.avg_ms}ms avg
            {hover.b.max_ms !== null && hover.b.max_ms !== hover.b.avg_ms
              ? ` · ${hover.b.max_ms}ms peak`
              : ""}
          </div>
          <div className={hover.b.fail > 0 ? "text-alert" : "text-mist-dim"}>
            {hover.b.ok} ok{hover.b.fail > 0 ? ` · ${hover.b.fail} failed` : ""}
            {hover.b.phase ? ` (${hover.b.phase})` : ""}
          </div>
        </div>
      )}
    </div>
  );
}

"use client";

import { useMemo, useRef, useState } from "react";
import type { CheckResult } from "@/lib/types";
import { fmtTime } from "@/lib/format";

type Point = { x: number; y: number; r: CheckResult };

// Single-series latency line (teal, 2px) with failed checks overlaid as red
// markers at the failure's observed latency (timeouts sit at the top).
// Single series → no legend box; the card title names it (dataviz rule).
// Hover shows a crosshair + tooltip; grid is recessive.
export function LatencyChart({ results }: { results: CheckResult[] }) {
  const [hover, setHover] = useState<Point | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  const W = 720;
  const H = 180;
  const PAD = { top: 12, right: 12, bottom: 24, left: 48 };

  const { points, yMax, ticks } = useMemo(() => {
    // Oldest → newest, keep only checks that measured a latency.
    const data = [...results].reverse().filter((r) => r.latency_ms !== null);
    const rawMax = Math.max(100, ...data.map((r) => r.latency_ms ?? 0));
    const yMax = Math.ceil(rawMax / 100) * 100;
    const innerW = W - PAD.left - PAD.right;
    const innerH = H - PAD.top - PAD.bottom;
    const points: Point[] = data.map((r, i) => ({
      x: PAD.left + (data.length === 1 ? innerW / 2 : (i / (data.length - 1)) * innerW),
      y: PAD.top + innerH - ((r.latency_ms ?? 0) / yMax) * innerH,
      r,
    }));
    const ticks = [0, 0.5, 1].map((f) => ({
      y: PAD.top + innerH - f * innerH,
      label: `${Math.round(f * yMax)}ms`,
    }));
    return { points, yMax, ticks };
  }, [results]);

  if (points.length === 0) {
    return (
      <div className="flex h-44 items-center justify-center font-mono text-sm text-mist-faint">
        no latency data in this window
      </div>
    );
  }

  const okPath = points
    .map((p, i) => `${i === 0 ? "M" : "L"}${p.x.toFixed(1)},${p.y.toFixed(1)}`)
    .join(" ");
  const fails = points.filter((p) => !p.r.ok);

  function onMove(e: React.MouseEvent<SVGSVGElement>) {
    const rect = svgRef.current?.getBoundingClientRect();
    if (!rect) return;
    const x = ((e.clientX - rect.left) / rect.width) * W;
    let best: Point = points[0];
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
        aria-label={`Response time, max ${yMax}ms`}
        onMouseMove={onMove}
        onMouseLeave={() => setHover(null)}
      >
        {/* recessive grid + axis labels in text tokens */}
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
            <text x={PAD.left - 8} y={t.y + 3} textAnchor="end" className="fill-mist-faint font-mono text-[10px]">
              {t.label}
            </text>
          </g>
        ))}
        <text x={PAD.left} y={H - 6} className="fill-mist-faint font-mono text-[10px]">
          {fmtTime(points[0].r.checked_at)}
        </text>
        <text x={W - PAD.right} y={H - 6} textAnchor="end" className="fill-mist-faint font-mono text-[10px]">
          {fmtTime(points[points.length - 1].r.checked_at)}
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

        <path d={okPath} fill="none" className="stroke-node" strokeWidth={2} />

        {/* failures: red markers with a 2px surface ring so they read over the line */}
        {fails.map((p) => (
          <circle key={p.r.id} cx={p.x} cy={p.y} r={4} className="fill-alert stroke-ink-card" strokeWidth={2} />
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
          <div className="text-mist">{fmtTime(hover.r.checked_at)}</div>
          <div className={hover.r.ok ? "text-node" : "text-alert"}>
            {hover.r.ok ? "ok" : "failed"} · {hover.r.latency_ms}ms
            {hover.r.status_code !== null ? ` · HTTP ${hover.r.status_code}` : ""}
          </div>
          {hover.r.error && <div className="max-w-56 truncate text-mist-dim">{hover.r.error}</div>}
        </div>
      )}
    </div>
  );
}

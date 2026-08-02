"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Monitor, SeriesPoint, Uptime } from "@/lib/types";
import { fmtPct, fmtTime } from "@/lib/format";
import { StateBadge } from "@/components/state-badge";
import { Sparkline } from "@/components/sparkline";
import { SetupGuide } from "@/components/setup-guide";

function MonitorRow({ monitor }: { monitor: Monitor }) {
  // 40 buckets is exactly what the sparkline draws. Previously this pulled
  // every raw result in the window — up to ~1,440 rows per monitor, per
  // refresh — to render the same 40 bars.
  const series = useQuery({
    queryKey: ["series", monitor.id, "24h", 40],
    queryFn: () =>
      api<{ buckets: SeriesPoint[] }>(`/api/monitors/${monitor.id}/series?window=24h&buckets=40`),
  });
  const uptime = useQuery({
    queryKey: ["uptime", monitor.id, "7d"],
    queryFn: () => api<Uptime>(`/api/monitors/${monitor.id}/uptime?window=7d`),
  });

  return (
    <Link
      href={`/monitors/${monitor.id}`}
      className="card flex flex-wrap items-center gap-4 hover:border-node/40"
    >
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-3">
          <span className="truncate font-medium text-white">{monitor.name}</span>
          <StateBadge state={monitor.enabled ? monitor.state : "unknown"} />
          {monitor.kind === "heartbeat" && (
            <span className="rounded-full border border-ink-line bg-ink-soft px-2 py-0.5 font-mono text-xs text-mist-faint">
              heartbeat
            </span>
          )}
          {!monitor.enabled && (
            <span className="font-mono text-xs text-mist-faint">paused</span>
          )}
        </div>
        <div className="mt-1 truncate font-mono text-xs text-mist-dim">
          {monitor.kind === "heartbeat"
            ? monitor.last_ping_at
              ? `last ping ${fmtTime(monitor.last_ping_at)}`
              : "waiting for the first ping"
            : monitor.url}
        </div>
      </div>
      <div className="flex items-center gap-6">
        <div className="text-right">
          <div className="font-mono text-sm text-mist">
            {uptime.data ? fmtPct(uptime.data.uptime_pct) : "…"}
          </div>
          <div className="text-xs text-mist-faint">7d uptime</div>
        </div>
        <Sparkline buckets={series.data?.buckets ?? []} />
      </div>
    </Link>
  );
}

export default function MonitorsPage() {
  const monitors = useQuery({
    queryKey: ["monitors"],
    queryFn: () => api<{ monitors: Monitor[] }>("/api/monitors"),
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Monitors</h1>
        <Link href="/monitors/new" className="btn-primary">
          Add monitor
        </Link>
      </div>

      <SetupGuide />

      {monitors.isPending && <p className="font-mono text-sm text-mist-dim">loading…</p>}
      {monitors.isError && <p className="text-alert">{monitors.error.message}</p>}

      {/* No empty-state card here: with zero monitors SetupGuide always renders
          its checklist, whose first step carries the same guidance and CTA.
          Two cards both saying "add a monitor" is noise. */}

      <div className="space-y-3">
        {monitors.data?.monitors.map((m) => (
          <MonitorRow key={m.id} monitor={m} />
        ))}
      </div>
    </div>
  );
}

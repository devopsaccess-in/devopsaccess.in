"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { CheckResult, Monitor, Uptime } from "@/lib/types";
import { fmtPct, fmtTime } from "@/lib/format";
import { StateBadge } from "@/components/state-badge";
import { Sparkline } from "@/components/sparkline";

function MonitorRow({ monitor }: { monitor: Monitor }) {
  const results = useQuery({
    queryKey: ["results", monitor.id, "24h"],
    queryFn: () =>
      api<{ results: CheckResult[] }>(`/api/monitors/${monitor.id}/results?window=24h`),
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
        <Sparkline results={results.data?.results ?? []} />
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

      {monitors.isPending && <p className="font-mono text-sm text-mist-dim">loading…</p>}
      {monitors.isError && <p className="text-alert">{monitors.error.message}</p>}

      {monitors.data?.monitors.length === 0 && (
        <div className="card text-center">
          <p className="text-mist">No monitors yet.</p>
          <p className="mt-2 text-sm text-mist-dim">
            Add your first endpoint and we start checking it within a minute.
          </p>
          <Link href="/monitors/new" className="btn-primary mt-4">
            Add your first monitor
          </Link>
        </div>
      )}

      <div className="space-y-3">
        {monitors.data?.monitors.map((m) => (
          <MonitorRow key={m.id} monitor={m} />
        ))}
      </div>
    </div>
  );
}

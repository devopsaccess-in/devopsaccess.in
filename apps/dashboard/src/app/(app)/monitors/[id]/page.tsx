"use client";

import Link from "next/link";
import { use, useState } from "react";
import { useRouter } from "next/navigation";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { CheckResult, Incident, Monitor, Uptime } from "@/lib/types";
import { fmtDuration, fmtPct, fmtTime } from "@/lib/format";
import { StateBadge } from "@/components/state-badge";
import { LatencyChart } from "@/components/latency-chart";
import { EmbedBadge } from "@/components/embed-badge";
import { PhaseBreakdown, TLSChip } from "@/components/phase-breakdown";
import { PingURL, fmtSeconds } from "@/components/ping-url";
import { EditMonitor } from "@/components/edit-monitor";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { SetupGuide } from "@/components/setup-guide";

function UptimeTile({ id, window: w }: { id: string; window: string }) {
  const uptime = useQuery({
    queryKey: ["uptime", id, w],
    queryFn: () => api<Uptime>(`/api/monitors/${id}/uptime?window=${w}`),
  });
  return (
    <div className="card py-4 text-center">
      <div className="font-mono text-xl text-white">
        {uptime.data ? fmtPct(uptime.data.uptime_pct) : "…"}
      </div>
      <div className="mt-1 text-xs text-mist-faint">{w} uptime</div>
    </div>
  );
}

// Next 16: params is a Promise — unwrap with use() in client components.
export default function MonitorDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const router = useRouter();
  const qc = useQueryClient();
  const [windowSel, setWindowSel] = useState<"24h" | "7d">("24h");
  const [confirmDelete, setConfirmDelete] = useState(false);

  const monitor = useQuery({
    queryKey: ["monitor", id],
    queryFn: () => api<Monitor>(`/api/monitors/${id}`),
  });
  const results = useQuery({
    queryKey: ["results", id, windowSel],
    queryFn: () =>
      api<{ results: CheckResult[] }>(`/api/monitors/${id}/results?window=${windowSel}`),
  });
  const incidents = useQuery({
    queryKey: ["incidents", id],
    queryFn: () => api<{ incidents: Incident[] }>(`/api/incidents?monitor_id=${id}`),
  });

  const toggle = useMutation({
    mutationFn: (enabled: boolean) =>
      api<Monitor>(`/api/monitors/${id}`, { method: "PATCH", body: JSON.stringify({ enabled }) }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["monitor", id] });
      void qc.invalidateQueries({ queryKey: ["monitors"] });
    },
  });
  const remove = useMutation({
    mutationFn: () => api<void>(`/api/monitors/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["monitors"] });
      router.push("/monitors");
    },
  });

  if (monitor.isPending) {
    return <p className="font-mono text-sm text-mist-dim">loading…</p>;
  }
  if (monitor.isError) {
    return <p className="text-alert">{monitor.error.message}</p>;
  }
  const m = monitor.data;

  return (
    <div className="space-y-6">
      <div>
        <Link href="/monitors" className="text-sm text-mist-dim hover:text-mist">
          ← Monitors
        </Link>
        <div className="mt-2 flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-semibold">{m.name}</h1>
          <StateBadge state={m.enabled ? m.state : "unknown"} />
          {!m.enabled && <span className="font-mono text-xs text-mist-faint">paused</span>}
          <TLSChip expiresAt={m.tls_expires_at} issuer={m.tls_issuer} />
        </div>
        <p className="mt-1 font-mono text-sm text-mist-dim">
          {m.kind === "heartbeat" ? (
            <>
              heartbeat · expect a ping every {fmtSeconds(m.period_seconds)} ·{" "}
              {fmtSeconds(m.grace_seconds)} grace
            </>
          ) : (
            <>
              {m.method} {m.url} · every {m.interval_seconds}s · expect {m.expected_status} ·
              alert after {m.failure_threshold} fails
            </>
          )}
        </p>
      </div>

      <SetupGuide compact />

      <div className="grid grid-cols-3 gap-4">
        <UptimeTile id={id} window="24h" />
        <UptimeTile id={id} window="7d" />
        <UptimeTile id={id} window="30d" />
      </div>

      <PingURL monitor={m} />

      {m.kind !== "heartbeat" && (
      <div className="card">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-base font-medium text-white">Response time</h2>
          <div className="flex gap-1">
            {(["24h", "7d"] as const).map((w) => (
              <button
                key={w}
                onClick={() => setWindowSel(w)}
                className={
                  w === windowSel
                    ? "rounded-md bg-ink-soft px-3 py-1 font-mono text-xs text-white"
                    : "rounded-md px-3 py-1 font-mono text-xs text-mist-faint hover:text-mist"
                }
              >
                {w}
              </button>
            ))}
          </div>
        </div>
        <LatencyChart results={results.data?.results ?? []} />
      </div>
      )}

      {m.kind !== "heartbeat" && (
      <div className="card">
        <div className="mb-4 flex items-baseline justify-between">
          <h2 className="text-base font-medium text-white">Where the time goes</h2>
          <span className="text-xs text-mist-faint">latest check</span>
        </div>
        <PhaseBreakdown results={results.data?.results ?? []} />
      </div>
      )}

      <div className="card">
        <h2 className="mb-4 text-base font-medium text-white">Incidents</h2>
        {incidents.data?.incidents.length === 0 && (
          <p className="font-mono text-sm text-mist-faint">no incidents — nice.</p>
        )}
        <ul className="divide-y divide-ink-line">
          {incidents.data?.incidents.map((i) => (
            <li key={i.id} className="flex flex-wrap items-center gap-3 py-3">
              <StateBadge state={i.resolved_at ? "up" : "down"} />
              <span className="text-sm text-mist">{fmtTime(i.started_at)}</span>
              <span className="font-mono text-xs text-mist-dim">
                {i.resolved_at ? `resolved after ${fmtDuration(i.started_at, i.resolved_at)}` : "ongoing"}
              </span>
              <span className="min-w-0 flex-1 truncate text-right font-mono text-xs text-mist-faint">
                {i.cause}
              </span>
            </li>
          ))}
        </ul>
      </div>

      <EditMonitor key={m.updated_at} monitor={m} />

      <EmbedBadge monitorId={id} />

      <div className="card flex flex-wrap items-center gap-3">
        <button
          className="btn-ghost"
          disabled={toggle.isPending}
          onClick={() => toggle.mutate(!m.enabled)}
        >
          {m.enabled ? "Pause checks" : "Resume checks"}
        </button>
        <button
          className="btn-danger"
          disabled={remove.isPending}
          onClick={() => setConfirmDelete(true)}
        >
          Delete monitor
        </button>
        {(toggle.isError || (remove.isError && !confirmDelete)) && (
          <span className="text-sm text-alert">
            {toggle.error?.message ?? remove.error?.message}
          </span>
        )}
      </div>

      <ConfirmDialog
        open={confirmDelete}
        title="Delete this monitor?"
        confirmLabel="Delete monitor"
        destructive
        pending={remove.isPending}
        error={remove.isError ? remove.error.message : null}
        onCancel={() => setConfirmDelete(false)}
        onConfirm={() => remove.mutate()}
        body={
          <>
            <p>
              <span className="text-white">{m.name}</span> and all of its check history,
              incidents, and uptime figures will be permanently removed.
            </p>
            {m.kind === "heartbeat" && (
              <p className="mt-2">
                Its ping URL stops working immediately — anything still calling it will
                start getting 404s.
              </p>
            )}
            <p className="mt-2">This cannot be undone.</p>
          </>
        }
      />
    </div>
  );
}

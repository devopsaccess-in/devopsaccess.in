"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Incident } from "@/lib/types";
import { fmtDuration, fmtTime } from "@/lib/format";
import { StateBadge } from "@/components/state-badge";

export default function IncidentsPage() {
  const incidents = useQuery({
    queryKey: ["incidents", "all"],
    queryFn: () => api<{ incidents: Incident[] }>("/api/incidents"),
  });

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Incidents</h1>

      {incidents.isPending && <p className="font-mono text-sm text-mist-dim">loading…</p>}
      {incidents.isError && <p className="text-alert">{incidents.error.message}</p>}
      {incidents.data?.incidents.length === 0 && (
        <div className="card text-center text-mist">
          No incidents recorded. Your monitors have never crossed their failure threshold.
        </div>
      )}

      <div className="space-y-3">
        {incidents.data?.incidents.map((i) => (
          <Link
            key={i.id}
            href={`/monitors/${i.monitor_id}`}
            className="card flex flex-wrap items-center gap-4 hover:border-node/40"
          >
            <StateBadge state={i.resolved_at ? "up" : "down"} />
            <div className="min-w-0 flex-1">
              <div className="font-medium text-white">{i.monitor_name}</div>
              <div className="mt-0.5 truncate font-mono text-xs text-mist-dim">{i.cause}</div>
            </div>
            <div className="text-right">
              <div className="text-sm text-mist">{fmtTime(i.started_at)}</div>
              <div className="font-mono text-xs text-mist-faint">
                {i.resolved_at
                  ? `resolved after ${fmtDuration(i.started_at, i.resolved_at)}`
                  : `ongoing — ${fmtDuration(i.started_at, null)}`}
              </div>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}

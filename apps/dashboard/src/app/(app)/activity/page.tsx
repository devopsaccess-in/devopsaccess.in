"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { AuditEntry } from "@/lib/types";
import { fmtTime } from "@/lib/format";

// Icon + tone per action family. Actions the system took (incidents) read
// differently from things a person did.
function actionStyle(action: string): { mark: string; cls: string } {
  if (action.startsWith("incident.")) {
    return action === "incident.open"
      ? { mark: "▲", cls: "text-alert" }
      : { mark: "●", cls: "text-node" };
  }
  if (action.endsWith(".delete")) return { mark: "−", cls: "text-alert" };
  if (action.endsWith(".create")) return { mark: "+", cls: "text-node" };
  if (action === "user.first_login") return { mark: "★", cls: "text-signal" };
  return { mark: "·", cls: "text-mist-dim" };
}

export default function ActivityPage() {
  const audit = useQuery({
    queryKey: ["audit"],
    queryFn: () => api<{ entries: AuditEntry[] }>("/api/audit"),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Activity</h1>
        <p className="mt-1 max-w-prose text-sm text-mist-dim">
          Everything that changed in this workspace, and who changed it. Kept for 90 days.
        </p>
      </div>

      {audit.isPending && <p className="font-mono text-sm text-mist-dim">loading…</p>}
      {audit.isError && <p className="text-alert">{audit.error.message}</p>}
      {audit.data?.entries.length === 0 && (
        <div className="card text-center text-sm text-mist-dim">Nothing recorded yet.</div>
      )}

      {audit.data && audit.data.entries.length > 0 && (
        <div className="card">
          <ul className="divide-y divide-ink-line">
            {audit.data.entries.map((e) => {
              const s = actionStyle(e.action);
              return (
                <li key={e.id} className="flex flex-wrap items-baseline gap-x-3 gap-y-1 py-3">
                  <span className={`font-mono text-sm ${s.cls}`} aria-hidden>
                    {s.mark}
                  </span>
                  <span className="min-w-0 flex-1 text-sm text-mist">{e.summary}</span>
                  <span className="font-mono text-xs text-mist-faint">
                    {e.actor_email || "system"}
                  </span>
                  <span className="font-mono text-xs text-mist-faint">
                    {fmtTime(e.created_at)}
                  </span>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}

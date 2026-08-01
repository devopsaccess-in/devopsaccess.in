"use client";

import { useState } from "react";
import type { Monitor } from "@/lib/types";
import { fmtTime } from "@/lib/format";

// PingURL is the heartbeat's whole interface: the secret URL your job calls
// when it finishes, with ready-to-paste snippets. Shown only for heartbeat
// monitors.
export function PingURL({ monitor }: { monitor: Monitor }) {
  const [copied, setCopied] = useState<string | null>(null);
  if (monitor.kind !== "heartbeat" || !monitor.ping_token) return null;

  const origin =
    typeof window !== "undefined" ? window.location.origin : "https://app.devopsaccess.in";
  const url = `${origin}/api/ping/${monitor.ping_token}`;

  const snippets = [
    { which: "curl", label: "At the end of your script", value: `curl -fsS -m 10 --retry 3 ${url}` },
    {
      which: "cron",
      label: "In crontab (only pings if the job succeeded)",
      value: `0 2 * * * /path/to/backup.sh && curl -fsS -m 10 --retry 3 ${url}`,
    },
  ];

  const copy = (text: string, which: string) => {
    void navigator.clipboard.writeText(text).then(() => {
      setCopied(which);
      setTimeout(() => setCopied(null), 1500);
    });
  };

  return (
    <div className="card space-y-4">
      <div>
        <h2 className="text-base font-medium text-white">Ping URL</h2>
        <p className="mt-1 text-xs text-mist-faint">
          Call this when your job finishes successfully. If we don&apos;t hear from it within{" "}
          <span className="text-mist">
            {fmtSeconds(monitor.period_seconds)} + {fmtSeconds(monitor.grace_seconds)} grace
          </span>
          , we open an incident and alert you. Keep the URL secret — anyone with it can report
          your job as healthy.
        </p>
      </div>

      <div className="flex items-center gap-2">
        <code className="min-w-0 flex-1 overflow-x-auto rounded-md border border-ink-line bg-ink-soft px-3 py-2 font-mono text-xs text-mist">
          {url}
        </code>
        <button type="button" onClick={() => copy(url, "url")} className="btn-ghost shrink-0">
          {copied === "url" ? "copied ✓" : "copy"}
        </button>
      </div>

      {snippets.map((s) => (
        <div key={s.which}>
          <div className="mb-1 flex items-center justify-between">
            <span className="text-xs text-mist-faint">{s.label}</span>
            <button
              type="button"
              onClick={() => copy(s.value, s.which)}
              className="font-mono text-xs text-node hover:underline"
            >
              {copied === s.which ? "copied ✓" : "copy"}
            </button>
          </div>
          <code className="block overflow-x-auto rounded-md border border-ink-line bg-ink-soft px-3 py-2 font-mono text-xs text-mist">
            {s.value}
          </code>
        </div>
      ))}

      <p className="font-mono text-xs text-mist-faint">
        last ping:{" "}
        {monitor.last_ping_at ? (
          <span className="text-mist">{fmtTime(monitor.last_ping_at)}</span>
        ) : (
          "never — waiting for the first one"
        )}
      </p>
    </div>
  );
}

export function fmtSeconds(s: number): string {
  if (s % 86400 === 0) return `${s / 86400}d`;
  if (s % 3600 === 0) return `${s / 3600}h`;
  if (s % 60 === 0) return `${s / 60}m`;
  return `${s}s`;
}

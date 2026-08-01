"use client";

import type { CheckResult } from "@/lib/types";

// Where the time went on the most recent check: DNS -> TCP connect -> TLS
// handshake -> server (time to first byte, minus the connection phases).
// Every segment is directly labelled with its name and value, so the phases
// are never identified by colour alone.
const PHASES = [
  { key: "dns", label: "DNS", cls: "bg-node" },
  { key: "tcp", label: "TCP", cls: "bg-[#a78bfa]" },
  { key: "tls", label: "TLS", cls: "bg-signal" },
  { key: "server", label: "Server", cls: "bg-mist-dim" },
] as const;

type Segment = { key: string; label: string; cls: string; ms: number };

export function PhaseBreakdown({ results }: { results: CheckResult[] }) {
  // Newest check that actually recorded a breakdown.
  const latest = results.find((r) => r.ttfb_ms !== null || r.connect_ms !== null);
  if (!latest) {
    return (
      <p className="font-mono text-sm text-mist-faint">
        no timing breakdown yet — it appears after the next check
      </p>
    );
  }

  const dns = latest.dns_ms ?? 0;
  const tcp = latest.connect_ms ?? 0;
  const tls = latest.tls_ms ?? 0;
  // TTFB is measured from the start of the request, so the server's own think
  // time is what's left after the connection phases.
  const server = Math.max(0, (latest.ttfb_ms ?? 0) - dns - tcp - tls);

  const values: Record<string, number> = { dns, tcp, tls, server };
  const segments: Segment[] = PHASES.map((p) => ({ ...p, ms: values[p.key] })).filter(
    (s) => s.ms > 0,
  );
  const total = segments.reduce((sum, s) => sum + s.ms, 0);

  if (total === 0) {
    return (
      <p className="font-mono text-sm text-mist-faint">
        responded too fast to break down (&lt;1ms per phase)
      </p>
    );
  }

  return (
    <div className="space-y-3">
      {/* Stacked bar. gap-0.5 gives the 2px surface gap between segments. */}
      <div className="flex h-6 w-full gap-0.5 overflow-hidden rounded">
        {segments.map((s) => (
          <div
            key={s.key}
            className={`${s.cls} first:rounded-l last:rounded-r`}
            style={{ width: `${(s.ms / total) * 100}%` }}
            title={`${s.label}: ${s.ms}ms`}
          />
        ))}
      </div>

      {/* Legend + direct values. */}
      <div className="flex flex-wrap gap-x-5 gap-y-1">
        {segments.map((s) => (
          <span key={s.key} className="flex items-center gap-1.5 text-xs">
            <span className={`inline-block h-2 w-2 rounded-sm ${s.cls}`} aria-hidden />
            <span className="text-mist-faint">{s.label}</span>
            <span className="font-mono text-mist">{s.ms}ms</span>
          </span>
        ))}
        <span className="ml-auto flex items-center gap-1.5 text-xs">
          <span className="text-mist-faint">total</span>
          <span className="font-mono text-white">{total}ms</span>
        </span>
      </div>
    </div>
  );
}

// TLSChip shows certificate expiry with urgency colouring — the warning that
// prevents the outage instead of reporting it. Text always states the days, so
// urgency is never colour-alone.
export function TLSChip({
  expiresAt,
  issuer,
}: {
  expiresAt: string | null;
  issuer: string;
}) {
  if (!expiresAt) return null;

  const days = Math.floor((new Date(expiresAt).getTime() - Date.now()) / 86_400_000);
  const tone =
    days < 0
      ? "border-alert/50 text-alert"
      : days <= 3
        ? "border-alert/50 text-alert"
        : days <= 14
          ? "border-signal/50 text-signal"
          : "border-ink-line text-mist-dim";

  const text =
    days < 0
      ? `TLS cert EXPIRED ${Math.abs(days)}d ago`
      : days === 0
        ? "TLS cert expires today"
        : `TLS cert valid ${days}d`;

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border bg-ink-soft px-2.5 py-0.5 font-mono text-xs ${tone}`}
      title={issuer ? `Issued by ${issuer}` : undefined}
    >
      {text}
    </span>
  );
}

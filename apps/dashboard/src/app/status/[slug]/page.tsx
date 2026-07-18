import type { Metadata } from "next";
import { notFound } from "next/navigation";
import type { StatusPage } from "@/lib/types";
import { fmtDuration, fmtPct, fmtTime } from "@/lib/format";
import { StateBadge } from "@/components/state-badge";

// Public status page: server-rendered, auth-exempt in middleware. Talks to
// the API directly over loopback (same VM) — not through nginx.
const API_BASE = process.env.API_INTERNAL_URL ?? "http://127.0.0.1:8081";

async function fetchStatus(slug: string): Promise<StatusPage | null> {
  const res = await fetch(`${API_BASE}/api/status/${encodeURIComponent(slug)}`, {
    next: { revalidate: 30 },
  });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error(`status API returned ${res.status}`);
  return (await res.json()) as StatusPage;
}

// Next 16: params is a Promise — await it.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const page = await fetchStatus(slug);
  return {
    title: page ? `${page.name} status` : "Status",
    robots: { index: false },
  };
}

export default async function PublicStatusPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const page = await fetchStatus(slug);
  if (!page) notFound();

  const allUp =
    page.monitors.length > 0 && page.monitors.every((m) => m.state !== "down");

  return (
    <div className="mx-auto max-w-3xl px-5 py-12">
      <header className="mb-8">
        <p className="eyebrow">status</p>
        <h1 className="mt-2 text-3xl font-semibold">{page.name}</h1>
        <p className={`mt-3 text-sm ${allUp ? "text-node" : "text-alert"}`}>
          {page.monitors.length === 0
            ? "No public monitors yet."
            : allUp
              ? "All systems operational."
              : "Some systems are experiencing issues."}
        </p>
      </header>

      <section className="space-y-3">
        {page.monitors.map((m) => (
          <div key={m.id} className="card flex items-center gap-4">
            <span className="min-w-0 flex-1 truncate font-medium text-white">{m.name}</span>
            <span className="font-mono text-sm text-mist-dim">
              {fmtPct(m.uptime_pct)} <span className="text-mist-faint">30d</span>
            </span>
            <StateBadge state={m.state} />
          </div>
        ))}
      </section>

      {page.incidents.length > 0 && (
        <section className="mt-10">
          <h2 className="mb-4 text-lg font-medium text-white">Incident history</h2>
          <ul className="divide-y divide-ink-line">
            {page.incidents.map((i) => (
              <li key={i.id} className="flex flex-wrap items-center gap-3 py-3 text-sm">
                <StateBadge state={i.resolved_at ? "up" : "down"} />
                <span className="text-mist">{i.monitor_name}</span>
                <span className="text-mist-dim">{fmtTime(i.started_at)}</span>
                <span className="ml-auto font-mono text-xs text-mist-faint">
                  {i.resolved_at
                    ? `resolved after ${fmtDuration(i.started_at, i.resolved_at)}`
                    : "ongoing"}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}

      <footer className="mt-12 border-t border-ink-line pt-6 text-xs text-mist-faint">
        Powered by{" "}
        <a href="https://devopsaccess.in" className="text-node hover:underline">
          DevOps Access
        </a>{" "}
        uptime monitoring.
      </footer>
    </div>
  );
}

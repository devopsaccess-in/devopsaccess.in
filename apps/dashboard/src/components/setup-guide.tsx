"use client";

import Link from "next/link";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Channel, Monitor } from "@/lib/types";

// SetupGuide closes the gap that quietly kills activation: a workspace with
// monitors but no alert channel looks healthy and does nothing. Incidents open
// silently, nobody is told, and the product appears not to work.
//
// Three states, in order of urgency:
//   monitors + no channels -> a warning, always shown, not dismissible
//   nothing set up yet     -> a two-step getting-started card
//   channels but no monitors -> a nudge to add the first monitor
// Fully set up renders nothing at all.
export function SetupGuide({ compact = false }: { compact?: boolean }) {
  const monitors = useQuery({
    queryKey: ["monitors"],
    queryFn: () => api<{ monitors: Monitor[] }>("/api/monitors"),
  });
  const channels = useQuery({
    queryKey: ["channels"],
    queryFn: () => api<{ channels: Channel[] }>("/api/channels"),
  });

  // Say nothing until both are known — a flash of "set up alerts" while the
  // channel list loads is worse than a beat of silence.
  if (!monitors.data || !channels.data) return null;

  const hasMonitors = monitors.data.monitors.length > 0;
  const hasChannels = channels.data.channels.length > 0;

  if (hasMonitors && hasChannels) return null;

  // The dangerous state: being watched, with nowhere to send the news.
  if (hasMonitors && !hasChannels) {
    return (
      <div className="rounded-lg border border-signal/50 bg-signal/5 p-4">
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <span className="font-mono text-sm text-signal" aria-hidden>
            ▲
          </span>
          <div className="min-w-0 flex-1">
            <p className="text-sm font-medium text-white">
              Nobody will be told when something breaks
            </p>
            <p className="mt-0.5 text-xs text-mist-dim">
              You have {monitors.data.monitors.length} monitor
              {monitors.data.monitors.length === 1 ? "" : "s"} but no alert channel, so
              incidents will only ever appear in this dashboard.
            </p>
          </div>
          <Link href="/channels" className="btn-primary shrink-0">
            Add an alert channel
          </Link>
        </div>
      </div>
    );
  }

  if (compact) return null; // the detail page only needs the warning above

  // Nothing yet, or channels-but-no-monitors: show what's left to do.
  const steps = [
    {
      done: hasMonitors,
      title: "Add something to watch",
      // Carries the discovery copy for both monitor kinds: this card is the
      // only thing an empty workspace shows, and heartbeats are otherwise
      // invisible from the page everyone starts on.
      body: (
        <>
          <span className="text-mist-dim">A website or API</span> — we call it every minute
          and alert you when it stops answering or returns the wrong status.{" "}
          <span className="text-mist-dim">A cron job or backup</span> — it pings a secret URL
          when it finishes, and we alert you when a run is missed.
        </>
      ),
      href: "/monitors/new",
      cta: "Add a monitor",
    },
    {
      done: hasChannels,
      title: "Tell us where to send alerts",
      body: <>Email or a Slack webhook. Send a test to prove it works before you rely on it.</>,
      href: "/channels",
      cta: "Add a channel",
    },
  ];

  return (
    <div className="card space-y-4">
      <div>
        <h2 className="text-base font-medium text-white">Get set up</h2>
        <p className="mt-0.5 text-xs text-mist-faint">Two steps, about a minute.</p>
      </div>
      <ol className="space-y-3">
        {steps.map((s, i) => (
          <li key={s.title} className="flex gap-3">
            <span
              className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full border font-mono text-xs ${
                s.done
                  ? "border-node bg-node/10 text-node"
                  : "border-ink-line text-mist-faint"
              }`}
              aria-hidden
            >
              {s.done ? "✓" : i + 1}
            </span>
            <div className="min-w-0 flex-1">
              <p className={s.done ? "text-sm text-mist-dim line-through" : "text-sm text-white"}>
                {s.title}
              </p>
              {!s.done && <p className="mt-0.5 text-xs text-mist-faint">{s.body}</p>}
            </div>
            {!s.done && (
              <Link href={s.href} className="btn-ghost shrink-0 self-start">
                {s.cta}
              </Link>
            )}
          </li>
        ))}
      </ol>
    </div>
  );
}

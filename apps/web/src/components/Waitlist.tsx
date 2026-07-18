"use client";

import { useState } from "react";
import Turnstile from "@/components/Turnstile";
import { track } from "@/lib/track";

// Early-access email capture. Posts to the Go service /api/waitlist.
export default function Waitlist({
  heading = "Early access is open",
  blurb = "Uptime monitoring and alerting that a small team can actually afford — built in the open. Join the list to try it first and tell us what to build next.",
}: {
  heading?: string;
  blurb?: string;
}) {
  const [status, setStatus] = useState<{ msg: string; ok: boolean } | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = Object.fromEntries(new FormData(form).entries());
    if (!data.email) {
      setStatus({ msg: "Please enter your email.", ok: false });
      return;
    }
    setBusy(true);
    setStatus({ msg: "Adding you…", ok: true });
    try {
      const res = await fetch("/api/waitlist", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (res.ok) {
        form.reset();
        setStatus({ msg: "You're on the list — thanks!", ok: true });
        track("waitlist_submitted");
      } else {
        const body = await res.json().catch(() => ({}) as { error?: string });
        setStatus({ msg: body.error || "Something went wrong. Please try again.", ok: false });
      }
    } catch {
      setStatus({ msg: "Network error. Please try again.", ok: false });
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="container-px py-16">
      <div className="rounded-xl border border-ink-line bg-ink-card/50 p-8 sm:p-10">
        <p className="eyebrow">Early access</p>
        <h2 className="mt-2 max-w-2xl text-3xl font-bold">{heading}</h2>
        <p className="prose-body mt-4 max-w-prose">{blurb}</p>

        <form className="mt-6 max-w-md space-y-3" noValidate onSubmit={onSubmit}>
          <div className="flex flex-col gap-3 sm:flex-row">
            <input
              name="email"
              type="email"
              required
              placeholder="you@startup.in"
              autoComplete="email"
              className="w-full rounded-md border border-ink-line bg-ink-soft px-3 py-2 text-mist outline-none focus:border-node"
            />
            <button type="submit" disabled={busy} className="btn-primary shrink-0">
              Join waitlist
            </button>
          </div>
          {/* Honeypot */}
          <input
            name="website"
            type="text"
            tabIndex={-1}
            autoComplete="off"
            className="hidden"
            aria-hidden="true"
          />
          <Turnstile />
        </form>
        {status && (
          <p
            className={`mt-3 font-mono text-xs ${status.ok ? "text-node" : "text-signal"}`}
            role="status"
            aria-live="polite"
          >
            {status.msg}
          </p>
        )}
      </div>
    </section>
  );
}

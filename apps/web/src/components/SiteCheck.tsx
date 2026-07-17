"use client";

import { useState } from "react";
import Turnstile from "@/components/Turnstile";
import Waitlist from "@/components/Waitlist";

type CheckItem = { name: string; detail: string; ok: boolean };
type Section = { grade: string; items: CheckItem[] };
type CheckResult = { host: string; security: Section; tls: Section; seo: Section };

function gradeClass(g: string) {
  if (g === "A" || g === "B") return "text-node border-node/40";
  if (g === "C") return "text-signal border-signal/40";
  return "text-red-400 border-red-400/40";
}

function ResultCard({ title, section }: { title: string; section: Section }) {
  return (
    <div className="card">
      <div className="flex items-center justify-between">
        <h3 className="text-lg">{title}</h3>
        <span
          className={`rounded-md border px-2.5 py-0.5 font-display text-lg ${gradeClass(section.grade)}`}
        >
          {section.grade}
        </span>
      </div>
      <ul className="mt-4 space-y-2">
        {section.items.map((i) => (
          <li key={i.name} className="flex items-start gap-2 text-sm">
            <span className={`${i.ok ? "text-node" : "text-red-400"} mt-0.5 font-mono`}>
              {i.ok ? "✓" : "✗"}
            </span>
            <span className="text-mist">
              {i.name}
              <span className="text-mist-dim"> — {i.detail}</span>
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default function SiteCheck() {
  const [status, setStatus] = useState<{ msg: string; ok: boolean } | null>(null);
  const [owns, setOwns] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<CheckResult | null>(null);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = Object.fromEntries(new FormData(form).entries()) as Record<string, string>;
    if (!data.url) return setStatus({ msg: "Enter a URL.", ok: false });
    if (data.owns !== "on")
      return setStatus({ msg: "Please confirm you own the site.", ok: false });

    setBusy(true);
    setStatus({ msg: "Checking… this takes a few seconds.", ok: true });
    setResult(null);

    try {
      const res = await fetch("/api/sitecheck", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url: data.url,
          owns: data.owns === "on",
          turnstile: data.turnstile,
          website: data.website,
        }),
      });
      const body = await res.json();
      if (!res.ok) {
        setStatus({ msg: body.error || "Something went wrong. Please try again.", ok: false });
        return;
      }
      setResult(body as CheckResult);
      setStatus({ msg: `Checked ${body.host}. Scroll down to keep improving.`, ok: true });
    } catch {
      setStatus({ msg: "Network error. Please try again.", ok: false });
    } finally {
      setBusy(false);
      // Turnstile tokens are single-use; reset so the next scan gets a fresh one.
      try {
        window.turnstile?.reset();
      } catch {
        /* not loaded */
      }
    }
  }

  return (
    <>
      <section className="container-px pt-16 pb-8">
        <p className="eyebrow">Free tool</p>
        <h1 className="mt-2 max-w-3xl text-4xl font-bold sm:text-5xl">
          Is your site healthy &amp; secure?
        </h1>
        <p className="prose-body mt-5 max-w-prose text-lg">
          A free, instant check of your <strong>security headers</strong>,{" "}
          <strong>SSL/TLS</strong>, and <strong>SEO basics</strong>. No signup, no spam — just
          paste your URL.
        </p>

        <form className="mt-8 max-w-xl space-y-4" noValidate onSubmit={onSubmit}>
          <div className="flex flex-col gap-3 sm:flex-row">
            <input
              name="url"
              type="text"
              required
              placeholder="yourstartup.in"
              autoComplete="url"
              className="w-full rounded-md border border-ink-line bg-ink-soft px-3 py-2 text-mist outline-none focus:border-node"
            />
            <button
              type="submit"
              disabled={!owns || busy}
              className="btn-primary shrink-0 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {busy ? (
                <span className="inline-flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent"></span>
                  Checking…
                </span>
              ) : (
                "Run free check"
              )}
            </button>
          </div>
          <label className="flex items-start gap-2 text-sm text-mist-dim">
            <input
              name="owns"
              type="checkbox"
              required
              checked={owns}
              onChange={(e) => setOwns(e.target.checked)}
              className="mt-1 h-4 w-4 accent-node"
            />
            <span>I own this site or have permission to scan it.</span>
          </label>
          <input
            name="website"
            type="text"
            tabIndex={-1}
            autoComplete="off"
            className="hidden"
            aria-hidden="true"
          />
          <Turnstile />
          {status && (
            <p
              className={`font-mono text-xs ${status.ok ? "text-node" : "text-signal"}`}
              role="status"
              aria-live="polite"
            >
              {status.msg}
            </p>
          )}
        </form>
      </section>

      {result && (
        <section className="container-px pb-8">
          <div className="grid gap-5 sm:grid-cols-3">
            <ResultCard title="Security headers" section={result.security} />
            <ResultCard title="SSL / TLS" section={result.tls} />
            <ResultCard title="SEO basics" section={result.seo} />
          </div>
        </section>
      )}

      {/* Early-access CTA: revealed only after a scan delivers value */}
      {result && (
        <Waitlist
          heading="Want this monthly — and the platform?"
          blurb="Join early access to get site checks and affordable uptime monitoring, before everyone else."
        />
      )}
    </>
  );
}

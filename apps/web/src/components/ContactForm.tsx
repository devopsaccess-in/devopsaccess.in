"use client";

import { useState } from "react";
import Turnstile from "@/components/Turnstile";
import { track } from "@/lib/track";

export default function ContactForm() {
  const [status, setStatus] = useState<{ msg: string; ok: boolean } | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const data = Object.fromEntries(new FormData(form).entries());

    if (!data.email || !data.message) {
      setStatus({ msg: "Email and message are required.", ok: false });
      return;
    }

    setBusy(true);
    setStatus({ msg: "Sending…", ok: true });
    try {
      const res = await fetch("/api/contact", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(data),
      });
      if (res.ok) {
        form.reset();
        setStatus({ msg: "Thanks — we'll get back to you within one business day.", ok: true });
        track("contact_submitted");
      } else {
        const body = await res.json().catch(() => ({}) as { error?: string });
        setStatus({
          msg: body.error || "Something went wrong. Please email support@devopsaccess.in.",
          ok: false,
        });
      }
    } catch {
      setStatus({ msg: "Network error. Please email support@devopsaccess.in.", ok: false });
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="mt-6 space-y-4" noValidate onSubmit={onSubmit}>
      <div>
        <label htmlFor="name" className="block font-mono text-xs text-mist-dim">
          First name
        </label>
        <input
          id="name"
          name="name"
          type="text"
          autoComplete="given-name"
          className="mt-1 w-full rounded-md border border-ink-line bg-ink-soft px-3 py-2 text-mist outline-none focus:border-node"
        />
      </div>
      <div>
        <label htmlFor="email" className="block font-mono text-xs text-mist-dim">
          Email <span className="text-signal">*</span>
        </label>
        <input
          id="email"
          name="email"
          type="email"
          required
          autoComplete="email"
          className="mt-1 w-full rounded-md border border-ink-line bg-ink-soft px-3 py-2 text-mist outline-none focus:border-node"
        />
      </div>
      <div>
        <label htmlFor="message" className="block font-mono text-xs text-mist-dim">
          Message <span className="text-signal">*</span>
        </label>
        <textarea
          id="message"
          name="message"
          rows={5}
          required
          className="mt-1 w-full rounded-md border border-ink-line bg-ink-soft px-3 py-2 text-mist outline-none focus:border-node"
        />
      </div>

      {/* Honeypot: hidden from humans, tempting to bots. Must stay empty. */}
      <div className="hidden" aria-hidden="true">
        <label htmlFor="website">Website</label>
        <input id="website" name="website" type="text" tabIndex={-1} autoComplete="off" />
      </div>

      <Turnstile />

      <button type="submit" disabled={busy} className="btn-primary">
        Send message
      </button>
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
  );
}

"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Monitor } from "@/lib/types";
import { track } from "@/lib/analytics";

export default function NewMonitorPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [kind, setKind] = useState<"http" | "heartbeat">("http");
  const [form, setForm] = useState({
    name: "",
    url: "",
    method: "GET",
    interval_seconds: 60,
    expected_status: 200,
    failure_threshold: 2,
    period_seconds: 3600,
    grace_seconds: 300,
  });

  const create = useMutation({
    mutationFn: () => {
      // Send only the fields that apply to the chosen kind.
      const body =
        kind === "heartbeat"
          ? {
              kind,
              name: form.name,
              period_seconds: form.period_seconds,
              grace_seconds: form.grace_seconds,
            }
          : { kind, ...form };
      return api<Monitor>("/api/monitors", { method: "POST", body: JSON.stringify(body) });
    },
    onSuccess: (m) => {
      track("monitor_created", { kind });
      void qc.invalidateQueries({ queryKey: ["monitors"] });
      // A heartbeat is useless until you wire up its ping URL, so go straight
      // to the detail page where the snippets live.
      router.push(kind === "heartbeat" ? `/monitors/${m.id}` : "/monitors");
    },
  });

  return (
    <div className="mx-auto max-w-xl space-y-6">
      <h1 className="text-2xl font-semibold">Add monitor</h1>
      <form
        className="card space-y-4"
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate();
        }}
      >
        {/* What kind of thing are we watching? */}
        <div className="grid grid-cols-2 gap-3">
          {(
            [
              {
                k: "http" as const,
                title: "Website / API",
                blurb: "We call your URL on a schedule",
              },
              {
                k: "heartbeat" as const,
                title: "Cron / job",
                blurb: "Your job calls us when it finishes",
              },
            ]
          ).map((opt) => (
            <button
              key={opt.k}
              type="button"
              onClick={() => setKind(opt.k)}
              aria-pressed={kind === opt.k}
              className={`rounded-lg border p-3 text-left transition ${
                kind === opt.k
                  ? "border-node bg-ink-soft"
                  : "border-ink-line hover:border-mist-faint"
              }`}
            >
              <div className="text-sm font-medium text-white">{opt.title}</div>
              <div className="mt-0.5 text-xs text-mist-faint">{opt.blurb}</div>
            </button>
          ))}
        </div>

        <label className="block">
          <span className="mb-1 block text-sm text-mist">Name</span>
          <input
            className="field"
            required
            maxLength={100}
            placeholder={kind === "heartbeat" ? "Nightly database backup" : "Production API"}
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
        </label>
        {kind === "heartbeat" ? (
          <>
            <div className="grid grid-cols-2 gap-4">
              <label className="block">
                <span className="mb-1 block text-sm text-mist">Expect a ping every</span>
                <select
                  className="field"
                  value={form.period_seconds}
                  onChange={(e) => setForm({ ...form, period_seconds: Number(e.target.value) })}
                >
                  <option value={300}>5 minutes</option>
                  <option value={900}>15 minutes</option>
                  <option value={3600}>hour</option>
                  <option value={21600}>6 hours</option>
                  <option value={86400}>day</option>
                  <option value={604800}>week</option>
                </select>
              </label>
              <label className="block">
                <span className="mb-1 block text-sm text-mist">Grace period</span>
                <select
                  className="field"
                  value={form.grace_seconds}
                  onChange={(e) => setForm({ ...form, grace_seconds: Number(e.target.value) })}
                >
                  <option value={60}>1 minute</option>
                  <option value={300}>5 minutes</option>
                  <option value={1800}>30 minutes</option>
                  <option value={3600}>1 hour</option>
                  <option value={21600}>6 hours</option>
                </select>
              </label>
            </div>
            <p className="text-xs text-mist-faint">
              We alert when no ping arrives within the period plus the grace. You get the ping
              URL and copy-paste snippets right after creating this.
            </p>
          </>
        ) : (
          <>
        <label className="block">
          <span className="mb-1 block text-sm text-mist">URL</span>
          <input
            className="field font-mono"
            required
            type="url"
            placeholder="https://api.example.com/healthz"
            value={form.url}
            onChange={(e) => setForm({ ...form, url: e.target.value })}
          />
          <span className="mt-1 block text-xs text-mist-faint">
            Public http(s) endpoints only, ports 80/443.
          </span>
        </label>
        <div className="grid grid-cols-2 gap-4">
          <label className="block">
            <span className="mb-1 block text-sm text-mist">Method</span>
            <select
              className="field"
              value={form.method}
              onChange={(e) => setForm({ ...form, method: e.target.value })}
            >
              <option value="GET">GET</option>
              <option value="HEAD">HEAD</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-mist">Check every</span>
            <select
              className="field"
              value={form.interval_seconds}
              onChange={(e) => setForm({ ...form, interval_seconds: Number(e.target.value) })}
            >
              <option value={60}>1 minute</option>
              <option value={120}>2 minutes</option>
              <option value={300}>5 minutes</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-mist">Expected status</span>
            <input
              className="field font-mono"
              type="number"
              min={100}
              max={599}
              value={form.expected_status}
              onChange={(e) => setForm({ ...form, expected_status: Number(e.target.value) })}
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-mist">Alert after</span>
            <select
              className="field"
              value={form.failure_threshold}
              onChange={(e) => setForm({ ...form, failure_threshold: Number(e.target.value) })}
            >
              <option value={1}>1 failed check</option>
              <option value={2}>2 failed checks</option>
              <option value={3}>3 failed checks</option>
              <option value={5}>5 failed checks</option>
            </select>
          </label>
        </div>
          </>
        )}

        {create.isError && <p className="text-sm text-alert">{create.error.message}</p>}

        <div className="flex gap-3">
          <button type="submit" className="btn-primary" disabled={create.isPending}>
            {create.isPending ? "Creating…" : "Create monitor"}
          </button>
          <button type="button" className="btn-ghost" onClick={() => router.back()}>
            Cancel
          </button>
        </div>
      </form>
    </div>
  );
}

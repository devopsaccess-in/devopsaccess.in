"use client";

import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Monitor } from "@/lib/types";

// EditMonitor lets you change a monitor after creating it — fix a typo'd URL,
// slow the interval down, widen a heartbeat's window. Sends only the fields
// that actually changed, so a PATCH never rewrites something you didn't touch.
export function EditMonitor({ monitor }: { monitor: Monitor }) {
  const qc = useQueryClient();
  const [open, setOpen] = useState(false);
  const [saved, setSaved] = useState(false);
  const heartbeat = monitor.kind === "heartbeat";

  const [form, setForm] = useState({
    name: monitor.name,
    url: monitor.url,
    method: monitor.method,
    interval_seconds: monitor.interval_seconds,
    expected_status: monitor.expected_status,
    failure_threshold: monitor.failure_threshold,
    period_seconds: monitor.period_seconds,
    grace_seconds: monitor.grace_seconds,
  });

  const save = useMutation({
    mutationFn: () => {
      // Diff against the monitor as loaded; unchanged fields are omitted.
      const body: Record<string, unknown> = {};
      const put = <K extends keyof typeof form>(k: K, current: unknown) => {
        if (form[k] !== current) body[k as string] = form[k];
      };
      put("name", monitor.name);
      put("failure_threshold", monitor.failure_threshold);
      if (heartbeat) {
        put("period_seconds", monitor.period_seconds);
        put("grace_seconds", monitor.grace_seconds);
      } else {
        put("url", monitor.url);
        put("method", monitor.method);
        put("interval_seconds", monitor.interval_seconds);
        put("expected_status", monitor.expected_status);
      }
      return api<Monitor>(`/api/monitors/${monitor.id}`, {
        method: "PATCH",
        body: JSON.stringify(body),
      });
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["monitor", monitor.id] });
      void qc.invalidateQueries({ queryKey: ["monitors"] });
      void qc.invalidateQueries({ queryKey: ["audit"] });
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
      setOpen(false);
    },
  });

  if (!open) {
    return (
      <div className="card flex items-center justify-between">
        <div>
          <h2 className="text-base font-medium text-white">Settings</h2>
          <p className="mt-0.5 text-xs text-mist-faint">
            {heartbeat
              ? "Change the name or how often you promise to ping."
              : "Change the URL, how often we check, or what counts as healthy."}
          </p>
        </div>
        <div className="flex items-center gap-3">
          {saved && <span className="font-mono text-xs text-node">saved ✓</span>}
          <button type="button" className="btn-ghost" onClick={() => setOpen(true)}>
            Edit
          </button>
        </div>
      </div>
    );
  }

  return (
    <form
      className="card space-y-4"
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate();
      }}
    >
      <h2 className="text-base font-medium text-white">Edit monitor</h2>

      <label className="block">
        <span className="mb-1 block text-sm text-mist">Name</span>
        <input
          className="field"
          required
          maxLength={100}
          value={form.name}
          onChange={(e) => setForm({ ...form, name: e.target.value })}
        />
      </label>

      {heartbeat ? (
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
      ) : (
        <>
          <label className="block">
            <span className="mb-1 block text-sm text-mist">URL</span>
            <input
              className="field font-mono"
              required
              type="url"
              value={form.url}
              onChange={(e) => setForm({ ...form, url: e.target.value })}
            />
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
          <p className="text-xs text-mist-faint">
            Changing the URL restarts the monitor from unknown and closes any open incident —
            past results describe a different target.
          </p>
        </>
      )}

      {save.isError && <p className="text-sm text-alert">{save.error.message}</p>}

      <div className="flex gap-3">
        <button type="submit" className="btn-primary" disabled={save.isPending}>
          {save.isPending ? "Saving…" : "Save changes"}
        </button>
        <button type="button" className="btn-ghost" onClick={() => setOpen(false)}>
          Cancel
        </button>
      </div>
    </form>
  );
}

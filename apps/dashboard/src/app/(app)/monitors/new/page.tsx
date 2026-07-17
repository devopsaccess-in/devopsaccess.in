"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Monitor } from "@/lib/types";

export default function NewMonitorPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [form, setForm] = useState({
    name: "",
    url: "",
    method: "GET",
    interval_seconds: 60,
    expected_status: 200,
    failure_threshold: 2,
  });

  const create = useMutation({
    mutationFn: () => api<Monitor>("/api/monitors", { method: "POST", body: JSON.stringify(form) }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["monitors"] });
      router.push("/monitors");
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
        <label className="block">
          <span className="mb-1 block text-sm text-mist">Name</span>
          <input
            className="field"
            required
            maxLength={100}
            placeholder="Production API"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
        </label>
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

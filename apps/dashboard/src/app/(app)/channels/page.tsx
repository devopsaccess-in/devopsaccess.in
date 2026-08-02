"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Channel } from "@/lib/types";
import { ConfirmDialog } from "@/components/confirm-dialog";

export default function ChannelsPage() {
  const qc = useQueryClient();
  const [type, setType] = useState<"email" | "slack_webhook">("email");
  const [value, setValue] = useState("");
  const [testResult, setTestResult] = useState<Record<string, string>>({});
  const [pendingRemove, setPendingRemove] = useState<Channel | null>(null);

  const channels = useQuery({
    queryKey: ["channels"],
    queryFn: () => api<{ channels: Channel[] }>("/api/channels"),
  });

  const create = useMutation({
    mutationFn: () =>
      api<Channel>("/api/channels", {
        method: "POST",
        body: JSON.stringify({
          type,
          config: type === "email" ? { to: value } : { url: value },
        }),
      }),
    onSuccess: () => {
      setValue("");
      void qc.invalidateQueries({ queryKey: ["channels"] });
    },
  });

  const remove = useMutation({
    mutationFn: (id: string) => api<void>(`/api/channels/${id}`, { method: "DELETE" }),
    onSuccess: () => {
      setPendingRemove(null); // otherwise the dialog lingers over a deleted row
      void qc.invalidateQueries({ queryKey: ["channels"] });
    },
  });

  const test = useMutation({
    mutationFn: (id: string) => api<{ ok: boolean }>(`/api/channels/${id}/test`, { method: "POST" }),
    onMutate: (id) => setTestResult((r) => ({ ...r, [id]: "sending…" })),
    onSuccess: (_data, id) => setTestResult((r) => ({ ...r, [id]: "sent ✓" })),
    onError: (err, id) => setTestResult((r) => ({ ...r, [id]: err.message })),
  });

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Alert channels</h1>
      <p className="max-w-prose text-sm text-mist-dim">
        Every enabled channel gets notified when a monitor goes down and again when it
        recovers. Send a test before you rely on one.
      </p>

      <form
        className="card flex flex-wrap items-end gap-3"
        onSubmit={(e) => {
          e.preventDefault();
          create.mutate();
        }}
      >
        <label className="block">
          <span className="mb-1 block text-sm text-mist">Type</span>
          <select
            className="field w-44"
            value={type}
            onChange={(e) => {
              setType(e.target.value as typeof type);
              setValue("");
            }}
          >
            <option value="email">Email</option>
            <option value="slack_webhook">Slack webhook</option>
          </select>
        </label>
        <label className="block min-w-64 flex-1">
          <span className="mb-1 block text-sm text-mist">
            {type === "email" ? "Address" : "Webhook URL"}
          </span>
          <input
            className="field font-mono"
            required
            type={type === "email" ? "email" : "url"}
            placeholder={
              type === "email" ? "oncall@yourteam.in" : "https://hooks.slack.com/services/…"
            }
            value={value}
            onChange={(e) => setValue(e.target.value)}
          />
        </label>
        <button type="submit" className="btn-primary" disabled={create.isPending}>
          Add channel
        </button>
        {create.isError && <p className="w-full text-sm text-alert">{create.error.message}</p>}
      </form>

      {channels.isPending && <p className="font-mono text-sm text-mist-dim">loading…</p>}
      {channels.isError && <p className="text-alert">{channels.error.message}</p>}

      <div className="space-y-3">
        {channels.data?.channels.map((c) => (
          <div key={c.id} className="card flex flex-wrap items-center gap-4">
            <span className="rounded-full border border-ink-line bg-ink-soft px-2.5 py-0.5 font-mono text-xs text-mist-dim">
              {c.type === "email" ? "email" : "slack"}
            </span>
            <span className="min-w-0 flex-1 truncate font-mono text-sm text-mist">
              {c.type === "email" ? c.config.to : c.config.url}
            </span>
            {testResult[c.id] && (
              <span className="font-mono text-xs text-mist-dim">{testResult[c.id]}</span>
            )}
            <button className="btn-ghost" disabled={test.isPending} onClick={() => test.mutate(c.id)}>
              Send test
            </button>
            <button
              className="btn-danger"
              disabled={remove.isPending}
              onClick={() => setPendingRemove(c)}
            >
              Remove
            </button>
          </div>
        ))}
        {channels.data?.channels.length === 0 && (
          <div className="card space-y-2 text-sm">
            <p className="text-white">No channels yet</p>
            <p className="text-mist-dim">
              Until you add one, incidents will open silently — visible here, but nobody
              gets told. Add the address or Slack channel your team actually watches, then
              send a test to prove it arrives.
            </p>
          </div>
        )}
      </div>

      <ConfirmDialog
        open={pendingRemove !== null}
        title="Remove this alert channel?"
        confirmLabel="Remove channel"
        destructive
        pending={remove.isPending}
        error={remove.isError ? remove.error.message : null}
        onCancel={() => setPendingRemove(null)}
        onConfirm={() => pendingRemove && remove.mutate(pendingRemove.id)}
        body={
          <>
            <p>
              <span className="text-white">
                {pendingRemove?.type === "email"
                  ? pendingRemove.config.to
                  : pendingRemove?.config.url}
              </span>{" "}
              will stop receiving alerts.
            </p>
            {channels.data?.channels.length === 1 && (
              <p className="mt-2 text-signal">
                This is your only channel — incidents will open silently, with no email or
                Slack notification at all.
              </p>
            )}
          </>
        }
      />
    </div>
  );
}

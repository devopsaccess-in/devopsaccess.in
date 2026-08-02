"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useMe } from "@/app/(app)/providers";
import { api } from "@/lib/api";
import { track } from "@/lib/analytics";

export default function SettingsPage() {
  const me = useMe();
  const qc = useQueryClient();
  const tenant = me.data?.tenant;
  const origin = typeof window !== "undefined" ? window.location.origin : "https://app.devopsaccess.in";
  const statusURL = tenant ? `${origin}/status/${tenant.slug}` : null;
  const enabled = tenant?.public_status_enabled ?? false;

  const toggle = useMutation({
    mutationFn: (next: boolean) =>
      api("/api/settings", {
        method: "PATCH",
        body: JSON.stringify({ public_status_enabled: next }),
      }),
    onSuccess: (_data, next) => {
      // Only the enabling direction is a funnel step — that is the moment a
      // customer decides the product is worth showing their own users.
      if (next) track("status_page_enabled");
      void qc.invalidateQueries({ queryKey: ["me"] });
    },
  });

  return (
    <div className="max-w-2xl space-y-6">
      <h1 className="text-2xl font-semibold">Settings</h1>

      <div className="card space-y-4">
        <h2 className="text-base font-medium text-white">Workspace</h2>
        <dl className="grid grid-cols-[8rem_1fr] gap-y-3 text-sm">
          <dt className="text-mist-faint">Name</dt>
          <dd className="text-mist">{tenant?.name ?? "…"}</dd>
          <dt className="text-mist-faint">Slug</dt>
          <dd className="font-mono text-mist">{tenant?.slug ?? "…"}</dd>
        </dl>
      </div>

      <div className="card space-y-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-medium text-white">Public status page</h2>
            <p className="mt-1 text-xs text-mist-faint">
              Off by default. When on, anyone with the link sees monitor names, states,
              30-day uptime and incident history — never URLs or error details.
            </p>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={enabled}
            disabled={!tenant || toggle.isPending}
            onClick={() => toggle.mutate(!enabled)}
            className={`mt-1 inline-flex h-6 w-11 shrink-0 items-center rounded-full transition ${
              enabled ? "bg-node" : "bg-ink-line"
            } disabled:opacity-50`}
          >
            <span
              className={`inline-block h-5 w-5 transform rounded-full bg-white transition ${
                enabled ? "translate-x-5" : "translate-x-0.5"
              }`}
            />
          </button>
        </div>
        {enabled && statusURL && (
          <a href={statusURL} className="block font-mono text-sm text-node hover:underline">
            {statusURL}
          </a>
        )}
        {toggle.isError && <p className="text-sm text-alert">{(toggle.error as Error).message}</p>}
      </div>

      <div className="card space-y-4">
        <h2 className="text-base font-medium text-white">Account</h2>
        <dl className="grid grid-cols-[8rem_1fr] gap-y-3 text-sm">
          <dt className="text-mist-faint">Email</dt>
          <dd className="text-mist">{me.data?.user.email || "—"}</dd>
          <dt className="text-mist-faint">Name</dt>
          <dd className="text-mist">{me.data?.user.name || "—"}</dd>
        </dl>
        <a href="/auth/logout" className="btn-ghost">
          Sign out
        </a>
      </div>

      <div className="card space-y-2">
        <h2 className="text-base font-medium text-white">Billing</h2>
        <p className="text-sm text-mist-dim">
          The uptime MVP is free while in early access. Paid plans (₹-priced, starting at
          $29/mo) arrive with the public launch — early-access users keep a discount.
        </p>
      </div>
    </div>
  );
}

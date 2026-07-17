"use client";

import { useMe } from "@/app/(app)/providers";

export default function SettingsPage() {
  const me = useMe();
  const tenant = me.data?.tenant;
  const statusURL = tenant ? `https://app.devopsaccess.in/status/${tenant.slug}` : null;

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
          <dt className="text-mist-faint">Status page</dt>
          <dd>
            {statusURL ? (
              <a href={statusURL} className="font-mono text-node hover:underline">
                {statusURL}
              </a>
            ) : (
              "…"
            )}
          </dd>
        </dl>
        <p className="text-xs text-mist-faint">
          The status page is public — share it with your users. It shows monitor names,
          states, 30-day uptime and incident history, never URLs.
        </p>
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

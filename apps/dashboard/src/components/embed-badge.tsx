"use client";

import { useState } from "react";
import { useMe } from "@/app/(app)/providers";

// EmbedBadge shows the live badge for a monitor plus copy-paste Markdown/HTML
// snippets. Only meaningful once the public status page is enabled (the badge
// endpoint is gated on the same opt-in), so we nudge the user there otherwise.
export function EmbedBadge({ monitorId }: { monitorId: string }) {
  const me = useMe();
  const [copied, setCopied] = useState<string | null>(null);

  const tenant = me.data?.tenant;
  const origin =
    typeof window !== "undefined" ? window.location.origin : "https://app.devopsaccess.in";
  if (!tenant) return null;

  const enabled = tenant.public_status_enabled;
  const badgeURL = `${origin}/api/badge/${tenant.slug}/${monitorId}.svg`;
  const statusURL = `${origin}/status/${tenant.slug}`;
  const markdown = `[![uptime](${badgeURL})](${statusURL})`;
  const html = `<a href="${statusURL}"><img src="${badgeURL}" alt="uptime"></a>`;

  const copy = (text: string, which: string) => {
    void navigator.clipboard.writeText(text).then(() => {
      setCopied(which);
      setTimeout(() => setCopied(null), 1500);
    });
  };

  return (
    <div className="card space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-base font-medium text-white">Embed badge</h2>
        {enabled && (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={`${badgeURL}?metric=uptime`} alt="uptime badge preview" height={20} />
        )}
      </div>

      {!enabled ? (
        <p className="text-sm text-mist-dim">
          Enable your public status page in{" "}
          <a href="/settings" className="text-node hover:underline">
            Settings
          </a>{" "}
          to embed a live uptime badge in your README or site.
        </p>
      ) : (
        <div className="space-y-3">
          <p className="text-xs text-mist-faint">
            A live SVG badge — drop it in a GitHub README or your site. Add{" "}
            <code className="text-mist">?metric=status</code> for up/down, or{" "}
            <code className="text-mist">?days=7</code> to change the window.
          </p>
          {[
            { which: "md", label: "Markdown", value: markdown },
            { which: "html", label: "HTML", value: html },
          ].map((s) => (
            <div key={s.which}>
              <div className="mb-1 flex items-center justify-between">
                <span className="text-xs text-mist-faint">{s.label}</span>
                <button
                  type="button"
                  onClick={() => copy(s.value, s.which)}
                  className="font-mono text-xs text-node hover:underline"
                >
                  {copied === s.which ? "copied ✓" : "copy"}
                </button>
              </div>
              <code className="block overflow-x-auto rounded-md border border-ink-line bg-ink-soft px-3 py-2 font-mono text-xs text-mist">
                {s.value}
              </code>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

"use client";

import { useEffect, useState } from "react";
import { applyPrefs, readPrefs, savePrefs } from "@/lib/analytics";

// Consent for dashboard analytics. Separate from the marketing site's banner
// because localStorage is per-origin (app.devopsaccess.in vs devopsaccess.in),
// and deliberately plainer: this is a signed-in tool, not a landing page, and
// the honest answer to "what do you collect" is short.
export function ConsentBanner() {
  const [show, setShow] = useState(false);

  useEffect(() => {
    const stored = readPrefs();
    if (stored) {
      applyPrefs(stored);
    } else {
      setShow(true);
    }
  }, []);

  if (!show) return null;

  const choose = (analytics: boolean) => {
    savePrefs({ analytics });
    setShow(false);
  };

  return (
    <div className="fixed inset-x-0 bottom-0 z-50 border-t border-ink-line bg-ink-soft/95 backdrop-blur">
      <div className="container-px flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-mist-dim">
          May we record which features you use — six events, nothing more — to work out what
          to improve? We never record your monitor URLs, alert addresses, or screen
          contents.
        </p>
        <div className="flex shrink-0 gap-3">
          <button className="btn-ghost !px-4 !py-1.5 text-sm" onClick={() => choose(false)}>
            No thanks
          </button>
          <button className="btn-primary !px-4 !py-1.5 text-sm" onClick={() => choose(true)}>
            Allow
          </button>
        </div>
      </div>
    </div>
  );
}

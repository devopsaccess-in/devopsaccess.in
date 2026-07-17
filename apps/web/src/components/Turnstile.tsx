"use client";

import { useEffect, useRef } from "react";

// Cloudflare Turnstile widget (explicit rendering). Renders nothing when
// NEXT_PUBLIC_TURNSTILE_SITEKEY is unset — forms then submit without a token,
// matching the backend's optional verification. Explicit rendering (instead of
// the implicit .cf-turnstile scan) survives client-side navigations remounting
// the form. The widget injects a hidden input named "turnstile" into the form.

const SITEKEY = process.env.NEXT_PUBLIC_TURNSTILE_SITEKEY || "";
const SCRIPT_ID = "cf-turnstile-api";

type TurnstileApi = {
  render: (el: HTMLElement, opts: Record<string, unknown>) => string;
  remove: (id: string) => void;
  reset: (id?: string) => void;
};

declare global {
  interface Window {
    turnstile?: TurnstileApi;
  }
}

function loadScript(onReady: () => void) {
  if (window.turnstile) {
    onReady();
    return;
  }
  const existing = document.getElementById(SCRIPT_ID);
  if (existing) {
    existing.addEventListener("load", onReady, { once: true });
    return;
  }
  const s = document.createElement("script");
  s.id = SCRIPT_ID;
  s.src = "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit";
  s.async = true;
  s.defer = true;
  s.addEventListener("load", onReady, { once: true });
  document.head.appendChild(s);
}

export default function Turnstile() {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!SITEKEY) return;
    let widgetId: string | undefined;
    let cancelled = false;

    loadScript(() => {
      if (cancelled || !ref.current || !window.turnstile) return;
      widgetId = window.turnstile.render(ref.current, {
        sitekey: SITEKEY,
        "response-field-name": "turnstile",
      });
    });

    return () => {
      cancelled = true;
      if (widgetId !== undefined) window.turnstile?.remove(widgetId);
    };
  }, []);

  if (!SITEKEY) return null;
  return <div ref={ref} />;
}

"use client";

// Product analytics for the dashboard: a deliberately tiny activation funnel,
// not general-purpose tracking.
//
// Why this is stricter than the marketing site's setup:
// this app renders a customer's own infrastructure — monitor URLs, alert
// addresses, heartbeat ping tokens (which are credentials). So autocapture and
// session replay are OFF, and pageviews are recorded as route PATTERNS rather
// than URLs. The only things sent are the six funnel events below, with
// non-identifying properties.
//
// Nothing loads until the visitor opts in.

const CONSENT_KEY = "da_consent_v2";
const POSTHOG_KEY = process.env.NEXT_PUBLIC_POSTHOG_KEY ?? "";
const POSTHOG_HOST = process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://r.devopsaccess.in";

/** The complete event vocabulary. Anything not here is not tracked. */
export type FunnelEvent =
  | "signup_completed"
  | "monitor_created"
  | "channel_created"
  | "channel_tested"
  | "incident_viewed"
  | "status_page_enabled";

type PostHog = {
  init: (key: string, config: Record<string, unknown>) => void;
  capture?: (event: string, props?: Record<string, unknown>) => void;
  identify?: (id: string, props?: Record<string, unknown>) => void;
  opt_out_capturing?: () => void;
  reset?: () => void;
};

declare global {
  interface Window {
    __daPhLoaded?: boolean;
    posthog?: PostHog;
  }
}

export type Prefs = { analytics: boolean };

export function readPrefs(): Prefs | null {
  if (typeof window === "undefined") return null;
  try {
    return JSON.parse(localStorage.getItem(CONSENT_KEY) || "null");
  } catch {
    return null;
  }
}

export function savePrefs(prefs: Prefs): void {
  localStorage.setItem(CONSENT_KEY, JSON.stringify({ ...prefs, v: 1, ts: Date.now() }));
  applyPrefs(prefs);
}

export function applyPrefs(prefs: Prefs): void {
  if (prefs.analytics) {
    loadPostHog();
  } else {
    try {
      window.posthog?.opt_out_capturing?.();
    } catch {
      /* never loaded */
    }
  }
}

function loadPostHog(): void {
  if (!POSTHOG_KEY || window.__daPhLoaded) return;
  window.__daPhLoaded = true;

  const s = document.createElement("script");
  s.src = POSTHOG_HOST.replace(/\/$/, "") + "/static/array.js";
  s.async = true;
  s.onload = () => {
    window.posthog?.init(POSTHOG_KEY, {
      api_host: POSTHOG_HOST,
      ui_host: "https://us.posthog.com",
      persistence: "localStorage", // cookieless
      // Everything below is deliberately off: this screen shows customer
      // infrastructure, and none of it belongs in an analytics tool.
      autocapture: false,
      capture_pageview: false,
      capture_pageleave: false,
      disable_session_recording: true,
      disable_surveys: true,
    });
  };
  document.head.appendChild(s);
}

/** Record a funnel event. No-ops unless analytics were accepted and loaded. */
export function track(event: FunnelEvent, props?: Record<string, unknown>): void {
  window.posthog?.capture?.(event, props);
}

/**
 * Associate events with a workspace. Identifies by opaque ids only — no email
 * or name — so the analytics tool never becomes a second copy of the user
 * directory.
 */
export function identify(userId: string, tenantId: string, tenantSlug: string): void {
  window.posthog?.identify?.(userId, { tenant_id: tenantId, tenant_slug: tenantSlug });
}

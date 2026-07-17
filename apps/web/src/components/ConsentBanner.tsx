"use client";

import { useEffect, useState } from "react";

// Category-based consent. Two categories:
//   - Strictly Necessary: always on (the consent choice itself; contact form).
//   - Analytics: opt-in. Loads PostHog (product analytics incl. session replay)
//     + Google Analytics 4. Decline => neither loads.
// Cloudflare Web Analytics runs at the edge (cookieless) regardless of choice.
// Re-open from the footer "cookie preferences" button (dispatches da-open-prefs).

const KEY = "da_consent_v2";
const POSTHOG_KEY = process.env.NEXT_PUBLIC_POSTHOG_KEY ?? "";
// PostHog reverse proxy (own domain, dodges ad blockers). Override via NEXT_PUBLIC_POSTHOG_HOST.
const POSTHOG_HOST = process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://r.devopsaccess.in";
const GA4_ID = process.env.NEXT_PUBLIC_GA4_ID || "";

type Prefs = { analytics: boolean };

declare global {
  interface Window {
    __phLoaded?: boolean;
    __ga4Loaded?: boolean;
    posthog?: {
      init: (key: string, config: Record<string, unknown>) => void;
      opt_out_capturing?: () => void;
    };
    dataLayer?: unknown[];
    gtag?: (...args: unknown[]) => void;
  }
}

function loadPostHog() {
  if (!POSTHOG_KEY || window.__phLoaded) return;
  window.__phLoaded = true;
  const s = document.createElement("script");
  s.src = POSTHOG_HOST.replace(/\/$/, "") + "/static/array.js";
  s.async = true;
  s.onload = () => {
    if (!window.posthog) return;
    window.posthog.init(POSTHOG_KEY, {
      api_host: POSTHOG_HOST, // reverse proxy (r.devopsaccess.in)
      ui_host: "https://us.posthog.com", // toolbar links point at the real PostHog UI
      persistence: "localStorage", // cookieless
      capture_pageview: true,
      capture_pageleave: true, // time-on-page / retention
      autocapture: true, // clicks & interactions
      person_profiles: "always", // geo / browser / referrer for anon visitors
      session_recording: { maskAllInputs: true }, // replay, but never record typed input
      disable_surveys: true, // we don't use PostHog surveys; skip loading surveys.js
    });
  };
  document.head.appendChild(s);
}

function loadGA4() {
  if (!GA4_ID || window.__ga4Loaded) return;
  window.__ga4Loaded = true;
  const s = document.createElement("script");
  s.src = "https://www.googletagmanager.com/gtag/js?id=" + GA4_ID;
  s.async = true;
  document.head.appendChild(s);
  window.dataLayer = window.dataLayer || [];
  function gtag(...args: unknown[]) {
    window.dataLayer!.push(args);
  }
  window.gtag = gtag;
  gtag("js", new Date());
  gtag("config", GA4_ID);
}

// Stop capturing if analytics was previously on and is now turned off.
function stopAnalytics() {
  try {
    window.posthog?.opt_out_capturing?.();
  } catch {
    /* posthog not loaded */
  }
  if (GA4_ID) (window as unknown as Record<string, unknown>)["ga-disable-" + GA4_ID] = true;
}

function apply(prefs: Prefs) {
  if (prefs.analytics) {
    if (GA4_ID) (window as unknown as Record<string, unknown>)["ga-disable-" + GA4_ID] = false;
    loadPostHog();
    loadGA4();
  } else {
    stopAnalytics();
  }
}

function readPrefs(): Prefs | null {
  try {
    return JSON.parse(localStorage.getItem(KEY) || "null");
  } catch {
    return null;
  }
}

export default function ConsentBanner() {
  const [showBanner, setShowBanner] = useState(false);
  const [showModal, setShowModal] = useState(false);
  const [analyticsChecked, setAnalyticsChecked] = useState(false);

  useEffect(() => {
    const stored = readPrefs();
    if (stored) {
      apply(stored);
    } else {
      setShowBanner(true);
    }

    const openPrefs = () => {
      const p = readPrefs();
      setAnalyticsChecked(!!p?.analytics);
      setShowBanner(false);
      setShowModal(true);
    };
    window.addEventListener("da-open-prefs", openPrefs);
    return () => window.removeEventListener("da-open-prefs", openPrefs);
  }, []);

  function save(prefs: Prefs) {
    localStorage.setItem(KEY, JSON.stringify({ ...prefs, v: 1, ts: Date.now() }));
    apply(prefs);
    setShowBanner(false);
    setShowModal(false);
  }

  function openModal() {
    const p = readPrefs();
    setAnalyticsChecked(!!p?.analytics);
    setShowBanner(false);
    setShowModal(true);
  }

  return (
    <>
      {showBanner && (
        <div className="fixed inset-x-0 bottom-0 z-50 border-t border-ink-line bg-ink-soft/95 backdrop-blur">
          <div className="container-px flex flex-col gap-3 py-4 sm:flex-row sm:items-center sm:justify-between">
            <p className="prose-body text-sm">
              We use privacy-first analytics to understand and improve the site. You choose what&apos;s
              allowed — see our{" "}
              <a href="/cookie-policy" className="text-node hover:underline">
                Cookie Policy
              </a>
              .
            </p>
            <div className="flex shrink-0 flex-wrap gap-3">
              <button
                className="btn-ghost !px-4 !py-1.5 text-sm"
                onClick={() => save({ analytics: false })}
              >
                Reject all
              </button>
              <button className="btn-ghost !px-4 !py-1.5 text-sm" onClick={openModal}>
                Manage
              </button>
              <button
                className="btn-primary !px-4 !py-1.5 text-sm"
                onClick={() => save({ analytics: true })}
              >
                Accept all
              </button>
            </div>
          </div>
        </div>
      )}

      {showModal && (
        <div
          className="fixed inset-0 z-[60] flex items-center justify-center bg-ink/70 p-4 backdrop-blur-sm"
          onClick={(e) => {
            if (e.target === e.currentTarget) setShowModal(false);
          }}
        >
          <div className="w-full max-w-lg rounded-xl border border-ink-line bg-ink-soft p-6 shadow-xl">
            <h2 className="text-xl font-bold">Cookie preferences</h2>
            <p className="prose-body mt-2 text-sm">
              Choose which cookies to allow. You can change this anytime from the footer.
            </p>

            <div className="mt-5 space-y-4">
              <div className="rounded-lg border border-ink-line bg-ink-card/40 p-4">
                <div className="flex items-center justify-between">
                  <span className="font-medium text-mist">Strictly necessary</span>
                  <span className="font-mono text-xs text-node">Always on</span>
                </div>
                <p className="prose-body mt-1 text-xs">
                  Remembers your consent choice and powers the contact form. No tracking.
                </p>
              </div>

              <div className="rounded-lg border border-ink-line bg-ink-card/40 p-4">
                <div className="flex items-center justify-between">
                  <label htmlFor="cat-analytics" className="font-medium text-mist">
                    Analytics
                  </label>
                  <input
                    id="cat-analytics"
                    type="checkbox"
                    className="h-4 w-4 accent-node"
                    checked={analyticsChecked}
                    onChange={(e) => setAnalyticsChecked(e.target.checked)}
                  />
                </div>
                <p className="prose-body mt-1 text-xs">
                  PostHog + Google Analytics: pages viewed, traffic source, country/browser, time on
                  page, and session recordings (with form inputs masked). Helps us improve the
                  product.
                </p>
              </div>
            </div>

            <div className="mt-6 flex flex-wrap justify-end gap-3">
              <button
                className="btn-ghost !px-4 !py-1.5 text-sm"
                onClick={() => save({ analytics: false })}
              >
                Reject all
              </button>
              <button
                className="btn-ghost !px-4 !py-1.5 text-sm"
                onClick={() => save({ analytics: analyticsChecked })}
              >
                Save preferences
              </button>
              <button
                className="btn-primary !px-4 !py-1.5 text-sm"
                onClick={() => save({ analytics: true })}
              >
                Accept all
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

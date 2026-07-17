// Fire a PostHog event if (and only if) analytics were accepted — i.e. the
// consent banner has loaded posthog. No-ops otherwise, so it's always safe to call.
export function track(event: string, props?: Record<string, unknown>): void {
  const ph = (window as unknown as { posthog?: { capture?: (e: string, p?: unknown) => void } })
    .posthog;
  if (ph && typeof ph.capture === "function") {
    ph.capture(event, props);
  }
}

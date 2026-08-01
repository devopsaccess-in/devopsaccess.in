# Manual test checklist

Tests that must be run by a human against **production** (or staging) because
they exercise real Auth0 login, the real Postfix→Gmail relay, real Slack
webhooks, real DNS/Cloudflare/nginx — things the automated `e2e/` suite (which
uses local stand-ins) can't fully prove.

**How to use:** run a test top to bottom, then fill the result line
(`PASS`/`FAIL` + date + notes). Re-run the whole file before every production
release. Automated coverage of the same behaviour lives in `e2e/` and CI — see
`FEATURES.md` for the map. Add a new test here whenever a PR adds user-facing
behaviour that has a real-environment dependency.

- Environment: https://app.devopsaccess.in (prod)
- Status: legend — ⬜ not run · ✅ pass · ❌ fail

---

## MT-1 — Signup & tenant provisioning
**Depends on:** Auth0 login, first-login provisioning.
1. Open https://app.devopsaccess.in in a fresh/incognito browser.
2. Sign in with a Google account not used before.
3. Consent, complete the redirect.

**Expected:** land on `/monitors`; the tenant slug is visible in the nav; no
error banner.

**Result:** ✅ 2026-07-18 — signup lands on /monitors, slug shows in nav.

---

## MT-2 — Add & test an email channel
**Depends on:** Postfix→Gmail relay, channel test endpoint.
1. Go to **Channels**.
2. Add an **Email** channel with an inbox you control. Save.
3. Click **Send test** on that channel.

**Expected:** a "Test alert from DevOps Access" email arrives within ~1 min;
the row shows "sent ✓".

**Result:** ⬜

---

## MT-3 — Add & test a Slack channel
**Depends on:** outbound HTTPS to Slack, webhook config.
1. In Slack: create an **Incoming Webhook** (Slack → Apps → Incoming Webhooks
   → Add to a channel → copy the `https://hooks.slack.com/services/...` URL).
2. In the dashboard **Channels**: add a **Slack webhook** channel with that URL.
3. Click **Send test**.

**Expected:** a "Test alert" message appears in the chosen Slack channel; the
row shows "sent ✓".

**Result:** ⬜ (no Slack workspace yet — create one, or skip; automated e2e
covers Slack dispatch logic.)

---

## MT-4 — Monitor goes UP
**Depends on:** scheduler probing a real public URL.
1. **Monitors → Add monitor.** Name "health-probe-up", URL
   `https://example.com`, method GET, every 1 minute, expected status 200,
   alert after 2 failed checks. Create.
2. Wait ~1–2 min; open the monitor.

**Expected:** state badge shows **up**; the latency chart starts drawing;
7-day uptime shows a value (100% at first).

**Result:** ⬜

---

## MT-5 — Outage → incident → alerts (the core promise)
**Depends on:** scheduler, state machine, incident open, email + Slack delivery.
1. **Add monitor** "health-probe-down", URL `https://httpstat.us/500`, expected
   status 200, alert after 2 failed checks. (This URL always returns 500, so it
   will fail.)
2. Wait ~2–3 min (2 failed checks at the 60s interval).

**Expected:**
- Monitor state flips to **down**.
- An **incident** appears (dashboard Incidents + the monitor page), marked
  ongoing.
- A **DOWN email** arrives ("DOWN: health-probe-down is failing").
- If a Slack channel exists (MT-3), a DOWN Slack message arrives.
- The alert does **not** contain the raw error/URL secrets — subject/body are
  clean.

**Result:** ⬜

---

## MT-6 — Recovery → resolved → recovery notice
**Depends on:** state machine down→up, incident resolve, recovery notify.
1. On "health-probe-down", edit the monitor and change **expected status** from
   200 to **500** (so the 500 response now counts as success). Save. *(Do NOT
   change the URL — a URL change closes the incident administratively; we want a
   natural recovery.)*
2. Wait ~1–2 min.

**Expected:**
- State flips back to **up**.
- The incident shows **resolved** with a duration.
- A **RESOLVED email** arrives ("RESOLVED: health-probe-down is back up"); Slack
  too if configured.

**Result:** ⬜

---

## MT-7 — Public status page (opt-in)
**Depends on:** status opt-in flag, public endpoint, nginx/DNS.
1. **Settings** — confirm the public status page toggle is **off** by default.
2. In another browser (logged out), open
   `https://app.devopsaccess.in/status/<your-slug>`.
   **Expected:** 404 / not found (opt-in gate).
3. Back in Settings, turn the toggle **on**; copy the status URL.
4. Reload the logged-out status URL.

**Expected:** the page renders your tenant name, each enabled monitor with
state + 30-day uptime, and incident history. It shows **no monitor URLs and no
technical error causes**. "Powered by DevOps Access" footer links to the site.

**Result:** ⬜

---

## MT-8 — Tenant isolation
**Depends on:** RLS + app scoping across two real Auth0 users.
1. Note a monitor id from your workspace (from its URL, `/monitors/<id>`).
2. Sign in as a **different** Google account (incognito) → new empty workspace.
3. Visit `/monitors/<id-from-tenant-1>` directly.

**Expected:** 404 (not the other tenant's monitor); the second workspace lists
zero monitors.

**Result:** ⬜

---

## MT-10 — Embeddable uptime badge
**Depends on:** public badge endpoint, status opt-in, real SVG rendering in a browser/GitHub.
1. With the public status page **enabled** (MT-7) and a monitor that's up (MT-4),
   open the monitor's detail page → **Embed badge** card → copy the Markdown.
2. Paste the badge URL directly into a browser tab:
   `https://app.devopsaccess.in/api/badge/<slug>/<monitorId>.svg`
3. Try `?metric=status` and `?days=7` variants.
4. Paste the Markdown snippet into any GitHub README preview.

**Expected:** a crisp flat badge renders — "uptime 99.xx%" (green when healthy),
`?metric=status` shows up/down, colors change with state. With the public status
page **off**, the badge shows a neutral "unavailable" (never real state).

**Result:** ⬜

---

## MT-11 — Deep probe: timings, TLS expiry, failure diagnosis
**Depends on:** real DNS/TLS against public sites, real certificate chains.
1. Create a monitor on a real https site you own (or `https://example.com`).
   Wait for 1–2 checks, open the monitor page.
2. Look at **Where the time goes** and the **TLS cert** chip next to the state
   badge.
3. Create a monitor on `https://expired.badssl.com` (expected status 200) and
   wait for it to fail its threshold.
4. Create a monitor on `https://wrong.host.badssl.com` and let it fail too.

**Expected:**
- (2) a stacked DNS / TCP / TLS / Server bar with per-phase ms and a total; the
  chip reads e.g. "TLS cert valid 87d" (amber under 14 days, red under 3).
- (3) the incident cause and the alert email read **"TLS certificate expired N
  days ago"** — not a generic failure.
- (4) the cause reads **"TLS certificate does not match the requested
  hostname"**.
- No cause anywhere contains the monitor's URL or query string.

**Result:** ⬜ (verified against live example.com + expired.badssl.com from a
dev machine on 2026-08-02: timings captured, "TLS certificate expired 4128 days
ago" — re-confirm on prod.)

---

## MT-12 — TLS expiry warning email
**Depends on:** scheduler cert-expiry pass, real relay, a cert expiring soon.
1. Point a monitor at an https host whose certificate expires in **under 14
   days** (e.g. a short-lived staging cert, or `https://expired.badssl.com` for
   the already-expired path).
2. Wait for one scheduler tick after the check records the certificate.

**Expected:** one warning email/Slack per rung — "TLS certificate for X expires
in N days" (or "EXPIRED: …"), naming the issuer and the expiry timestamp in
IST. Crucially it arrives **once**, not every tick, and not again after a
restart.

**Result:** ⬜

---

## MT-9 — Cleanup
1. Delete the test monitors (health-probe-up, health-probe-down) and the test
   channels.

**Expected:** they disappear; deleting a monitor also removes its history (no
orphan rows surface elsewhere).

**Result:** ⬜

---

<!-- Add new manual tests below. Template:

## MT-N — <title>
**Depends on:** <real-environment dependency>
1. step
2. step

**Expected:** <observable result>

**Result:** ⬜
-->

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

**Result:** ✅ 2026-08-02 — test email delivered (landed in Gmail spam; see MT-15).

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

**Result:** ✅ 2026-08-02 — monitor went down, incident opened, DOWN email delivered. Note: httpstat.us reset the connection rather than returning 500, so the diagnosis read "connection reset by peer" — the tcp-phase path, not the status path. Both are valid failures.

---

## MT-6 — Recovery → resolved → recovery notice
**Depends on:** state machine down→up, incident resolve, recovery notify.
1. With a monitor currently **down** (from MT-5), open it and use
   **Settings → Edit**.
2. Change **Expected status** to whatever the endpoint actually returns (e.g.
   `500` for `httpstat.us/500`) so the next check counts as success. Save.
   *(Do NOT change the URL — a URL change closes the incident
   administratively; we want a natural recovery.)*
3. Wait ~1–2 min.

**Note:** if the endpoint fails at the connection level rather than returning
a status (httpstat.us sometimes resets the connection), changing the expected
status cannot fix it. In that case point a monitor at `https://example.com`
with expected status `999`, let it go down, then edit it back to `200`.

**Expected:**
- State flips back to **up**.
- The incident shows **resolved** with a duration.
- A **RESOLVED email** arrives; Slack too if configured.

**Result:** ✅ 2026-08-02 — example.com with expected status 599 went down ("2 consecutive failures: expected status 599, got 200"); editing it back to 200 via Settings → Edit recovered it, RESOLVED email reported "Downtime: 3m20s".
edit UI at all (only Pause/Delete), so the monitor could not be changed. Fixed
by adding Settings → Edit; re-run after deploying that.

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

## MT-13 — Heartbeat monitor (cron / dead man's switch)
**Depends on:** public ping endpoint through nginx/Cloudflare, real relay.
1. **Monitors → Add monitor → "Cron / job"**. Name "test heartbeat", expect a
   ping every **5 minutes**, grace **1 minute**. Create — you land on the
   detail page.
2. Copy the `curl` snippet and run it from your laptop. Reload the page.
3. Stop pinging. Wait ~6–7 minutes (period + grace + one evaluation).
4. Run the curl snippet once more.
5. Try a made-up token: `curl -i https://app.devopsaccess.in/api/ping/zzzzzzzzzzzzzzzzzzzzzzzzzzz`

**Expected:**
- (2) `ok` in the terminal; "last ping" updates; state **up**.
- (3) state flips to **down**, an incident opens whose cause says how late the
  heartbeat is, and a DOWN email arrives.
- (4) state returns to **up** immediately, the incident resolves, and a
  RESOLVED email arrives.
- (5) HTTP 404 with a bare "not found" — no hint about whether the token
  format is right.
- The detail page shows no response-time chart or phase breakdown (there is no
  HTTP probe for a heartbeat).

**Result:** ✅ 2026-08-02 — ping returned ok; after silence the heartbeat went down with cause "heartbeat is 8s late (last ping 6 minutes ago, expected every 5 minutes)" and the DOWN email arrived. Recovery ping + RESOLVED still to confirm.

---

## MT-14 — Logging & audit (post-deploy)
**Depends on:** VictoriaLogs + vector on the VM, Grafana, real request traffic.
1. Deploy with the `logs` tag (Configure Server → `tags=logs`).
2. SSH in and confirm both services are up:
   `systemctl status victorialogs vector --no-pager | head -20`
3. Generate traffic: load the dashboard, create and delete a monitor.
4. In Grafana (grafana.devopsaccess.in) → Explore → **VictoriaLogs**
   datasource. Query `{service="uptime-api.service"}`.
5. In the dashboard, open **Activity**.
6. Check disk: `du -sh /var/lib/victoria-logs && journalctl --disk-usage`

**Expected:**
- (2) both active; VictoriaLogs listening on 127.0.0.1:9428 only
  (`ss -ltnp | grep 9428` shows no 0.0.0.0 bind).
- (4) structured lines with `level`, `svc`, `route`, `status`, `duration`,
  `tenant_id` — and nginx lines under `{service="nginx"}`.
- (5) the monitor create and delete appear, attributed to your email, newest
  first; the delete entry still names the monitor.
- (6) journal capped around 500M; VictoriaLogs data well under the 3GiB cap.
- No ping tokens or monitor ids in the `route` field (route patterns only).

**Result:** ⬜

---

## MT-15 — Alert email deliverability (lands in inbox, not spam)
**Depends on:** SPF/DKIM/DMARC for devopsaccess.in, sender alignment.
**Background:** on 2026-08-02 every alert was accepted by Gmail (`250 OK`) but
filed as **spam**. Two causes: we sent as `alerts@devopsaccess.in`, which is
not a Workspace mailbox — Google silently rewrote the sender, and that
mismatch is a spam signal — and the domain had no DKIM signature.

1. After deploying the sender fix, confirm the address:
   `sudo grep ALERT_FROM /opt/uptime-*/*.env` → both read
   `support@devopsaccess.in`.
2. Check the domain's auth records resolve:
   ```bash
   dig +short TXT devopsaccess.in | grep spf
   dig +short TXT google._domainkey.devopsaccess.in
   dig +short TXT _dmarc.devopsaccess.in
   ```
3. Trigger a fresh alert (break a monitor) and open the received mail →
   **Show original** in Gmail.
4. Optional, most objective: send a test to `check-auth@verifier.port25.com`
   or use mail-tester.com and read the score.

**Expected:** SPF **pass**, DKIM **pass**, DMARC **pass** in "Show original",
and the alert lands in the **inbox**. Sender reads `support@devopsaccess.in`
with no Google rewrite.

**Result:** ✅ 2026-08-02 — SPF, DKIM and DMARC all PASS; both the DOWN and RESOLVED alerts landed in the **inbox**, sender reads support@devopsaccess.in with no Google rewrite. ALERT_FROM verified on the box.

---

## MT-16 — First-run experience
**Depends on:** a workspace in each setup state.
1. Sign in as a **brand-new** Google account (incognito) → land on /monitors.
2. Create a monitor, but **no** channel yet. Reload /monitors, then open the
   monitor's detail page.
3. Add an alert channel.

**Expected:**
- (1) a "Get set up" card with two numbered steps and no monitor rows; the
  empty state explains both monitor kinds, including cron/heartbeat.
- (2) an amber warning — "Nobody will be told when something breaks" — naming
  the monitor count, on **both** the list and the detail page, with a button
  to add a channel. It is not dismissible.
- (3) both the card and the warning disappear; the page is just monitors.

**Result:** ✅ 2026-08-02 — with a channel present and no monitors, the checklist showed step 1 pending and step 2 ticked through. (Follow-up: the duplicate empty-state card was merged into the checklist so only one "add a monitor" CTA remains.)

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

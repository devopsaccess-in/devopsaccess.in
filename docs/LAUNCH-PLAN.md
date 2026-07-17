# Launch plan — finish Phase A + Phase B, ship, and get to First Users

> Written 2026-07-17. This is the execution doc: work top to bottom. Companion
> docs: `docs/HANDOFF-uptime-mvp.md` (context + decisions), `docs/SECRETS.md`
> (exact secret names), `FEATURES.md` (feature inventory ↔ e2e coverage).

## 0. Where things stand

- **Phase A (marketing site Next.js migration)** — code complete on
  `feat/web/nextjs-migration`, verified (build, typecheck, URL parity,
  fonts, consent gating). NOT merged, NOT deployed.
- **Phase B (uptime MVP)** — code complete on `feat/app/uptime-mvp`,
  verified (unit + vet green everywhere; black-box e2e suite green in ~10s
  incl. the incident pipeline and public status page; RLS integration test
  green). NOT merged, NOT deployed.
- CI: `e2e.yml` gates every PR. `infra.yml` (provision), `deploy-web.yml`,
  `deploy-app.yml`, `configure.yml` all live in this monorepo now.
- The production VM (Hetzner cx23) runs the OLD Astro site + contact service
  today, deployed from the old `devopsaccess-in` repo.

## 1. One-time setup (browser work, ~1–2 hours) — blocks everything

Do these in any order; all are prerequisites for the deploys below.

### 1.1 GitHub secrets on THIS repo
Copy the old repo's secrets/vars and add the five new ones — the exact list
with purposes is `docs/SECRETS.md`. (Browser only; `gh` is the office
account.)

### 1.2 Auth0 tenant (free tier)
1. Create tenant (region: EU or closest available; note the domain →
   `AUTH0_DOMAIN` secret, no scheme).
2. **API**: Applications → APIs → Create. Identifier
   `https://api.devopsaccess.in`, RS256. Leave defaults.
3. **Application**: Regular Web App.
   - Allowed Callback URLs: `https://app.devopsaccess.in/auth/callback`,
     `http://localhost:3001/auth/callback`
   - Allowed Logout URLs: `https://app.devopsaccess.in`,
     `http://localhost:3001`
   - Note Client ID / Client Secret → secrets.
4. **Connections**: enable Username-Password-Authentication + Google.
5. **Post-login Action** (Actions → Triggers → post-login → custom):
   ```js
   exports.onExecutePostLogin = async (event, api) => {
     const ns = "https://devopsaccess.in";
     if (event.user.email) api.accessToken.setCustomClaim(`${ns}/email`, event.user.email);
     if (event.user.name)  api.accessToken.setCustomClaim(`${ns}/name`,  event.user.name);
   };
   ```
   Without this, auto-provisioned tenants get generic names/slugs.
6. `AUTH0_SECRET` = `openssl rand -hex 32`.

### 1.3 Cloudflare (dashboard)
- Add **A record `app`** → VM IP, proxied (same as apex + grafana).
- Verify the Origin cert covers `*.devopsaccess.in` (it already serves
  `grafana.`, so it should) — SSL/TLS → Origin Server.

## 2. Ship Phase A (marketing site cutover)

1. Open PR `feat/web/nextjs-migration` → main. CI (e2e.yml) must be green.
   Squash-merge.
2. Run **Deploy Web** (auto on merge via paths, else dispatch) — builds the
   static export, rsyncs the webroot, purges the Cloudflare cache.
3. Run **Configure Server** with `tags=nginx` once so the new vhost config
   (incl. `/_next/static/` caching and the new app vhost) is live.
4. Verify on prod: the 3 forms (contact, site-check, waitlist) end-to-end;
   no analytics before consent; `/_next/static/` returns immutable cache
   headers.
5. PageSpeed Insights on `/` — CLS budget ≤ 0.02.
6. Search Console: resubmit `/sitemap.xml`.
7. **Disable the old repo's deploy workflow** (old repo → Actions →
   Deploy → disable). The old repo stays as rollback until step 4 below has
   survived a week.

## 3. Ship Phase B (the product)

1. Rebase `feat/app/uptime-mvp` on main (post-Phase-A merge), push, open PR.
   CI must be green. Squash-merge.
2. Run **Configure Server** with `tags=postgres` — creates the `uptime` DB +
   `uptime_api`/`uptime_scheduler` roles (needs `UPTIME_DB_PASSWORD` secret).
3. Run **Deploy App** — builds amd64 binaries + dashboard standalone,
   deploys `app_api`, `app_scheduler`, `app_dashboard`, refreshes nginx.
   The workflow smoke-checks `/healthz` (API) and the dashboard over
   loopback at the end.
4. Watch first boot: `journalctl -u uptime-api -u uptime-scheduler -u dashboard -f`
   — the API runs migrations on startup; all three should settle in seconds.

## 4. The manual E2E gate (defines "launchable")

Run against production, as a real user:

- [ ] Sign up at app.devopsaccess.in with a fresh Google account → lands on
      /monitors, tenant slug visible in the nav.
- [ ] Add an email channel + a Slack webhook channel; **Send test** on both
      → both arrive.
- [ ] Create a 60s monitor (threshold 2) on a disposable endpoint you
      control.
- [ ] Break the endpoint → within ~3 min: monitor shows **down**, incident
      appears, email + Slack alerts arrive.
- [ ] Fix the endpoint → recovery notice arrives, incident shows resolved
      with duration.
- [ ] `/status/{slug}` (logged out, other browser) shows the monitor,
      uptime %, and the incident history.
- [ ] Second Auth0 user cannot open the first tenant's monitor by URL (404)
      and sees an empty workspace.
- [ ] `journalctl -u uptime-scheduler` shows no error-level lines during the
      whole exercise.

When every box ticks, the product is live. Announce nothing yet — first do
§6 (logging/audit) so you can actually support users.

## 5. Hetzner sizing — what this needs to run

Everything (site, product, Postgres, monitoring) stays on the ONE existing
VM. Measured composition on the box after Phase B:

| Component | Steady RAM | Notes |
|---|---|---|
| nginx + Postfix + fail2ban + exporters | ~150 MB | already running |
| PostgreSQL (defaults) | ~200–300 MB | shared_buffers default 128MB is fine at this scale |
| contact (Go) | ~20 MB | |
| uptime-api (Go) | ~25 MB | |
| uptime-scheduler (Go) | ~30 MB | 20 workers, IO-bound |
| dashboard (node, standalone) | ~120–180 MB | single process |
| Prometheus + Grafana | ~400–500 MB | already running |
| **Total** | **~1.1–1.4 GB** | on a 4 GB box → ~2.5 GB headroom |

Load math (why CPU is a non-issue): 100 monitors at 60s = ~1.7 probes/s and
~1.7 row inserts/s; 500 monitors ≈ 8/s — trivial for 2 vCPU. Postgres data:
100 monitors ≈ 4.3M `monitor_results` rows at the 30-day retention ≈ ~1 GB
including indexes. cx23 disk is **40 GB** (verified July 2026) — with OS +
builds + backups budget ~15 GB for data, comfortable to ~500 monitors; add
a Hetzner volume (€0.05/GB/mo) before resizing if disk ever leads.

**Decision: stay on the current cx23 (2 vCPU / 4 GB / 40 GB, €5.49/mo ≈
₹550/mo at current pricing — ours is grandfathered lower until we rescale).**
Add the log store from §6 (~+150–250 MB) and it still fits. Resize triggers
(Hetzner resize is a reboot, minutes of downtime, keep the same IP):
- sustained RAM > 3 GB or swap activity → cx33-class (4 vCPU / 8 GB)
- > ~500 active monitors or noticeable p95 API latency → same
- paying customers > ~20 → consider splitting Postgres onto its own small
  VM (also the first BYOC-shaped step)

Budget check: VM ~₹400 + Cloudflare free + Auth0 free + Terraform Cloud
free + GitHub free = **well under the ₹10,000/mo cap**, leaving room for
the paid tiers of nothing. Good.

## 6. Logging & audit system (build right after shipping, before inviting users)

Three layers, smallest-possible footprint, all self-hosted:

### 6.1 Operational logs (what broke) — ship week 1 after launch
Already true: all four Go services emit structured JSON (zerolog) to stderr
→ systemd journal; nginx writes per-vhost access logs.

To add:
1. **API request logging** — the API currently logs only errors. Add an
   access-log middleware (request id, method, path, status, duration,
   tenant_id, auth0 sub when authed). Same for the scheduler's notify sends
   (already partially there).
2. **Log aggregation on the VM** — one lightweight log store + a collector
   scraping journald + nginx logs, wired as a Grafana datasource (Grafana is
   already running). Candidate chosen after version research: see §8 —
   criteria: < 250 MB RAM, single binary, journald ingestion, Grafana
   datasource.
3. **Retention caps** — journald `SystemMaxUse=500M`; log store retention
   14–30 days; nginx logrotate is already default on Ubuntu.
4. **Error alerting** — Grafana alert: error-level log rate > threshold →
   email (the Grafana → contact-point plumbing exists from the monitoring
   role).

New Ansible role `logs` mirroring the monitoring role; deployed via
Configure Server `tags=logs`.

### 6.2 Product audit trail (what did WHO do) — same PR as 6.1 or next
A first-class, tenant-scoped audit table — customers will ask for this, and
it's the debugging tool for "my monitor disappeared":

```sql
CREATE TABLE audit_log (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_sub  TEXT NOT NULL DEFAULT '',   -- Auth0 subject ('' = system/scheduler)
    action     TEXT NOT NULL,              -- monitor.create, monitor.update, monitor.delete,
                                           -- channel.create, channel.delete, channel.test,
                                           -- user.first_login, incident.open, incident.resolve
    entity_id  UUID,
    details    JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- RLS identical to the other tenant tables + FORCE; index (tenant_id, created_at DESC).
```

- Written **in the same transaction** as the mutation (store-layer helper),
  so the trail can't diverge from reality.
- Scheduler writes incident.open / incident.resolve rows (BYPASSRLS role
  needs INSERT grant).
- 90-day retention via the existing nightly housekeeping.
- Surface later as a simple list under /settings → "Activity" (feature row
  in FEATURES.md when built; e2e asserts audit rows exist after the
  pipeline test).
- Migration `0002_audit_log.sql` — the embedded runner picks it up
  automatically.

### 6.3 Analytics (what do users do) — decision, not code, this week
- Marketing site: PostHog + GA4 behind consent — already live, unchanged.
- Dashboard: add PostHog (free tier) with the SAME consent gating pattern
  as the site, tracking only: signup completed, monitor created, channel
  created, channel tested, incident viewed, status page shared. That list
  is the activation funnel for First Users; more events = noise.
- Status pages: no analytics (they're the customer's page, not ours).
- SEO: dashboard + status pages stay `noindex` (correct for an app);
  the marketing site owns SEO. Post-launch task: a `/uptime` product page
  on devopsaccess.in (with schema.org SoftwareApplication markup) linking
  to app.devopsaccess.in — that page is the SEO surface for the product.

## 7. Review findings (branch audit, 2026-07-17)

A high-effort multi-agent review (37 verified findings, top 10 reported) ran
over the whole `feat/app/uptime-mvp` diff. **Eight were fixed the same day**
(all tenant-reachable security/correctness bugs); the fixes ship with the e2e
assertions that lock them in. Two are deferred with rationale.

### Fixed

1. **SMTP header injection** (`shared/notify`) — a monitor name with CR/LF
   went into the alert `Subject:` unsanitized, letting a tenant inject
   `Bcc:`/headers and spam via our relay. Fix: mailer strips CR/LF from all
   header fields, AND monitor-name validation rejects control chars. e2e:
   validation test rejects a CRLF name.
2. **Monitor-URL leak on the public status page** (`scheduler/prober`,
   `api/status`) — probe errors embed the full URL (incl. secret tokens),
   which flowed into `incident.cause` and out the unauthenticated status
   endpoint. Fix: probe errors are rendered without the URL
   (`probeErrorMessage` unwraps `*url.Error`), AND the public status payload
   omits `cause` entirely. e2e asserts no cause leaks.
3. **Dropped DOWN/RESOLVED alerts** (`scheduler/notifier`) — `notify_state`
   advanced even when every send failed, so one transient SMTP/Slack blip
   permanently lost the alert. Fix: only advance when there was nothing to
   send or ≥1 channel succeeded; otherwise retry next tick, bounded by a
   1-hour window so a permanently broken channel can't spin forever.
4. **Orphaned incidents** (`api/store`) — editing a down monitor's URL (state
   → unknown) or pausing it left the open incident unresolved forever, which
   also blocked future incidents via the open-incident unique index. Fix:
   UpdateMonitor resolves the open incident in the same transaction on URL
   change or pause, marking it terminal so no spurious alert fires.
5. **UTF-8 truncation stall** (`scheduler/prober`) — `truncate()` cut error
   strings on a byte boundary; an invalid-UTF-8 cause made the
   `monitor_results` INSERT fail every tick, silently stalling that monitor.
   Fix: rune-safe truncate.
6. **Status pages public by default at a guessable slug** (`api/status` +
   migration `0002`) — biggest one. Slugs derive from the email localpart and
   the page needed no opt-in, so anyone could enumerate them. Fix: new
   `tenants.public_status_enabled` (default FALSE), endpoint 404s unless
   enabled, `PATCH /api/settings` toggle + a switch on the dashboard Settings
   page. e2e asserts 404-before-opt-in.
7. **Tenant-resolution mismatch** (`api/store`) — `TenantForSub` (data
   routes) ordered by `tenant_id` while `/api/me` ordered by `created_at`, so
   a multi-membership user could see one tenant and edit another. Fix: same
   ordering. (Latent today — no multi-membership flow yet — but free to fix.)
8. **Unverified email channels** (`api/channels`) — *partially* addressed:
   the CRLF/relay-abuse vector is closed by #1; the remaining "add any
   recipient" concern is deferred (below) since it needs a verification flow.

### Deferred (with rationale)

- **Email/Slack channel ownership verification** — a user can still add an
  arbitrary recipient and trigger sends. Real fix is a confirm-link flow
  (email) / one-time challenge (Slack). Deferred to post-launch; interim
  mitigations to add in the §6 work: per-tenant channel cap + rate-limit the
  `/test` endpoint (the contact service's `ratelimit.go` is the pattern).
  Tracked as a FEATURES.md row when built.
- **7d results window truncated at 5000 rows** — the `/results?window=7d`
  query caps at 5000, so a 60s monitor's 7d chart shows ~3.5 days. Not a
  correctness/security issue (uptime % uses a separate COUNT query that is
  not capped). Fix later with server-side downsampling (bucket to N points);
  raising the cap alone just moves the cliff. Tracked for the charts polish
  pass.

Lower-severity efficiency/convention notes from the review (per-probe
`safehttp.Client` allocation, JWKS lock granularity, dashboard N+1 fetches on
the monitors list, minor CI/nginx nits) are deferred as non-blocking polish —
none affect correctness or security.

## 8. Tech stack version audit (July 2026)

Verified against live sources (releases pages, endoflife.date, advisories)
on 2026-07-17. Context7 MCP was not available in the session; web
verification used instead.

**Applied on the branch same day:**

| Tech | Was | Now | Why |
|---|---|---|---|
| golang-jwt/jwt/v5 | v5.2.1 | **v5.3.1** | **CVE-2025-30204 / GHSA-mh63-6h87-95cp** — DoS via period-stuffed Bearer header; 5.2.1 vulnerable |
| @auth0/nextjs-auth0 | 4.9.0 | **^4.25.0** | 4.23.1 shipped an auth-freshness (max_age) security fix; 4.25 officially documents the Next 16 proxy.ts pattern we use |
| @tanstack/react-query | 5.62.0 | **^5.101.2** | routine catch-up, no advisories |
| Node on the VM + CI (app) | 22 | **24** | 22 is Maintenance LTS (EOL Apr 2027); 24 is Active LTS (Apr 2028). Changed before first deploy = free |

**Deliberate keeps (decisions, not oversights):**

- **Go 1.25** — CI's `setup-go: "1.25"` resolves the latest patch (≥1.25.11,
  which carries the June net/http + crypto fixes), so shipped binaries are
  patched. Move to 1.26 (Green Tea GC — nice for the scheduler) in a quiet
  window.
- **Next.js 16.2.10** — current LTS patch, includes the May security batch
  (incl. the proxy.ts bypass CVE — and our auth doesn't rely on proxy.ts
  alone; the Go API independently validates every JWT). **Watch the
  pre-announced July 20, 2026 Next.js security release and patch promptly.**
- **Tailwind 3.4** — supported until Feb 2027, zero advisories. Plan the v4
  migration (config→CSS) as one deliberate PR before Q1 2027, both apps at
  once.
- **nginx `listen 443 ssl http2;` syntax** — stock Ubuntu 24.04 nginx is
  1.24, which does NOT understand the newer standalone `http2 on;`
  directive, so the "deprecated" syntax is the only correct one for us.
  Revisit at the Ubuntu 26.04 upgrade.
- **PostgreSQL 16 (Ubuntu apt)** — supported until Nov 2028. Research
  suggested PGDG + PG 18 for fresh DBs, but the uptime DB lives on the SAME
  cluster as the existing contact DB on one 4 GB VM — running a second
  major version there costs RAM and ops complexity for zero product value.
  One cluster, PG 16, revisit when Postgres moves off the box (§5).
- **pnpm 9.14.4** — pinned via `packageManager`, so CI cannot float to the
  breaking v11. Migrate to 11 deliberately later (.npmrc → workspace-yaml
  settings move).
- **chi v5.2.0 / pgx v5.7.2 / zerolog v1.33.0** — no advisories; newer
  minors exist (5.3.1 / 5.10.0 / 1.35.1) — routine bumps at leisure.
- **PostHog free tier** — still 1M events/mo; our §6.3 funnel is thousands
  at most. Zero cost risk. (Umami noted as the self-host fallback.)
- **Log store: VictoriaLogs** (Apache-2.0, single binary, ~tens of MB RAM
  at our volume, native journald ingestion, Grafana datasource plugin) —
  chosen over Loki (AGPL; measured ~5–8× heavier in 2026 comparisons) for
  the §6 logging layer. Hetzner note: the June 2026 price adjustment means
  our grandfathered cx23 price survives only until we rescale (cx23 now
  €5.49, cx33 €6.99 — still trivially within budget).

## 9. After this doc: First Users (the next plan)

Not planned in detail yet — inputs to that plan: the audit/log system (§6)
running, the activation funnel events (§6.3), and the `/uptime` product
page. Sketch: waitlist invites in batches of 10 → watch funnel + logs →
fix the top friction → repeat; Razorpay payment link + monitor-count plan
limits before the first paid conversion.

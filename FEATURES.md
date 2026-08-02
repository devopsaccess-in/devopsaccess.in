# Feature inventory

The single list of what DevOps Access ships. **Every PR that adds, changes, or
removes user-facing behaviour updates this file AND the E2E coverage in
[`e2e/`](e2e/README.md)** — the PR template has a checkbox for it, and
`.github/workflows/e2e.yml` runs the suite on every PR.

Coverage legend: **E2E** = automated in `e2e/` (test name given) · **unit** =
covered by package tests · **manual** = part of the pre-launch manual gate in
`docs/HANDOFF-uptime-mvp.md` (needs real Auth0/DNS/Cloudflare).

## Uptime monitoring (the MVP product — app.devopsaccess.in)

| Feature | Status | Coverage |
|---|---|---|
| Signup via Auth0; user + personal tenant auto-provisioned on first login (unique slug from email) | built | E2E `TestSignupProvisioning` · manual (real Auth0 login) |
| Multi-tenant isolation: cross-tenant reads/writes indistinguishable from 404, Postgres RLS + app-layer scoping | built | E2E `TestTenantIsolation` · integration `TestTenantIsolationRLS` (`-tags integration`) |
| HTTP(S) monitors: CRUD, GET/HEAD, 60–300s interval, expected status, failure threshold 1–10, pause/resume | built | E2E `TestMonitorValidation`, `TestIncidentPipeline` · unit (field validation) |
| Edit a monitor after creation (name, URL, method, interval, expected status, threshold; heartbeat period/grace) via Settings → Edit; sends only changed fields | built | E2E `TestEditMonitor` (full + partial patch, rejects leave state unchanged, heartbeat cadence) |
| Heartbeat ("dead man's switch") monitors: your cron/backup/pipeline pings a secret URL; silence past period+grace opens an incident and alerts. Public `GET\|POST /api/ping/{token}`, copy-paste curl/crontab snippets in the dashboard | built | E2E `TestHeartbeatMonitor` (ping→up, silence→down+email, ping→recovery, unknown token 404) · unit `TestEvaluateHeartbeat` |
| SSRF-guarded probing and URL validation (public IPs only, ports 80/443, DNS-rebinding safe) | built | unit `TestValidateMonitorURL` · code: `safehttp` dialer re-checks at connect time |
| Prober: due-monitor claiming (`SKIP LOCKED`), worker pool, per-monitor timeout, results history | built | E2E `TestIncidentPipeline` |
| Deep probe: per-phase timings (DNS / TCP / TLS / server) recorded on every check, shown as a breakdown on the monitor page | built | unit `TestCheckCapturesPhaseTimings` · E2E asserts timings on every result |
| Failure diagnosis: failures are classified to a phase and a plain-English cause ("TLS certificate expired 2 days ago", "connection refused", "timed out during the TLS handshake") — used in incidents and alerts, never echoes the URL | built | unit `TestDiagnose` (13 cases + URL-leak guard), `TestCheckDiagnoses*` against real servers |
| TLS certificate tracking: expiry + issuer per monitor, chip on the monitor page, proactive expiry warnings at 14 and 3 days (idempotent, resets on renewal) | built | unit `TestCertRung`, `TestComposeCertAlert`, `TestCheckCapturesTLSCertificate` · manual MT-11 |
| Incident state machine: unknown/up/down, threshold crossing opens exactly one incident, first success resolves | built | unit `TestEvaluate` · E2E `TestIncidentPipeline` |
| Alert channels: email + Slack webhook, per-channel test send | built | E2E `TestIncidentPipeline` |
| Down + recovery notifications (restart-safe via `notify_state`, IST timestamps) | built | E2E `TestIncidentPipeline` |
| Uptime % and check-result windows (`24h`/`7d`/`30d`) | built | E2E `TestIncidentPipeline` · unit `TestParseWindow` |
| Public status page — opt-in per tenant (off by default; `PATCH /api/settings` + Settings toggle) | built | E2E `TestIncidentPipeline` (404-before-opt-in) |
| Public status page — API (`/api/status/{slug}`), no URL/cause leakage | built | E2E `TestIncidentPipeline` |
| Public status page — dashboard UI (`/status/{slug}`) | built | E2E `TestDashboardStatusPage` |
| Embeddable SVG badge (`/api/badge/{slug}/{id}.svg`, uptime/status, gated on the status opt-in) + copy-paste embed snippets in the dashboard | built | E2E `TestIncidentPipeline` (renders + no cross-slug leak) · unit (color/format/escaping) |
| Dashboard (authed): monitors list + sparklines, monitor detail + latency chart, incidents, channels, settings | built | manual (needs real Auth0) — pre-launch gate |
| Audit trail: every mutation (monitor/channel/settings) and system event (incident open/resolve) recorded per tenant, written in the same transaction as the change; Activity page in the dashboard; 90-day retention | built | E2E `TestAuditTrail` (attribution, survives delete, no webhook leak, cross-tenant) · integration RLS test |
| Structured API access logs (route pattern, status, duration, tenant, actor, request id) | built | manual MT-14 |
| Log aggregation: VictoriaLogs + vector shipping journald + nginx logs, queryable in Grafana; journald + log-store retention caps | built (infra) | manual MT-14 |
| 30-day results retention (nightly purge) | built | code-reviewed; no automated test (startup purge logged) |
| Billing: manual Razorpay payment link | decided, not built | — |

## Marketing site (devopsaccess.in) + contact service

| Feature | Status | Coverage |
|---|---|---|
| Next.js 16 static site, 19 pages, URL parity with the old Astro site | built (Phase A) | build-time checks in deploy-web.yml; PSI/CLS manual gate |
| Contact / site-check / waitlist forms → Go contact service (Turnstile, rate limits) | live | unit tests in `services/contact` |
| Razorpay payment webhook recording | live | unit tests in `services/contact` |

## Explicitly cut from v1 (do not build — see handoff §cut list)

Deploy engine, OTel/APM ingestor, gRPC/protos, CLI, multi-region probes,
Razorpay subscription automation, SMS/voice paging, on-call schedules,
non-HTTP check types, team invites, custom status domains, Hindi UI,
k3s/ArgoCD, Kafka/NATS/Redis.

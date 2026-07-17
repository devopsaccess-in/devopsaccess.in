# HANDOFF — Next.js migration (DONE) + Uptime Monitoring MVP (IN PROGRESS)

> Written 2026-07-17 for a fresh Claude session with zero context. Read this whole file
> before writing code. The approved plan also lives at
> `~/.claude-personal/plans/this-is-a-terraform-quiet-haven.md` (same content, less repo state).

## 1. The big picture

Vikram (solo, part-time founder, Bengaluru) is pivoting devopsaccess.in from a
consulting-bridge site to a real SaaS: **uptime monitoring + alerting** ("first paying
client in 90 days", Notion "Launch Command Center"). Decisions already made with him —
**do not relitigate**:

- v1 product slice: HTTP uptime monitors, incidents, email + Slack alerts, public status
  page, dashboard. Billing = manual Razorpay payment link (no subscription automation).
- All code lives in THIS monorepo (`~/Downloads/devopsaccess/code/devopsaccess`,
  remote `git@github-personal:devopsaccess-in/devopsaccess.in.git`).
- Deploys to the EXISTING single Hetzner VM (nginx + systemd via Ansible). **No k3s.**
- **Auth0 free tier** for dashboard auth (Go API validates Auth0 RS256 JWTs).
- Marketing site was migrated Astro → **Next.js 16 static export** (Phase A, done).
- Budget cap ₹10,000/month; cloud-agnostic (Postgres only, no proprietary stores).
- About page: only the CKA certification is real. Never re-add the other certs.

## 2. Repo layout + git state

- Old repo `~/Downloads/devopsaccess-in` = the LIVE site (Astro). Still deployable =
  rollback. Its `gh` CLI is the OFFICE account — **never use `gh` here; git push via the
  `github-personal` SSH alias works; PRs/secrets via browser.**
- Monorepo branches:
  - `feat/web/nextjs-migration` — **Phase A complete, pushed.** 6 commits: contact
    service + ansible/terraform imported into `services/contact` + `infra/`; `apps/web`
    Next.js 16 (output export, trailingSlash, Velite MDX, next/font/local, all 19 pages,
    llms.txt/llms-full.txt/sitemap.ts, consent banner gating PostHog+GA4, Turnstile
    explicit rendering, contact/site-check/waitlist forms → same-origin `/api/*`);
    `.github/workflows/deploy-web.yml` + `configure.yml`; nginx `/_next/static/` cache.
    Verified: build+typecheck green, 100% URL parity vs prod sitemap (20 URLs),
    size-adjust font fallbacks present, no analytics pre-consent, no Razorpay JS.
  - `feat/app/uptime-mvp` — **Phase B started, NOT pushed, nothing committed yet.**
    Uncommitted work-in-progress files (see §4).

## 3. Phase B design (approved) — build exactly this

### B1 — Postgres schema + RLS (new DB `uptime` on the VM's existing Postgres)
Tables: `tenants(id uuid pk, name, slug unique, created_at)`;
`users(id uuid pk, auth0_sub unique, email, name, created_at)`;
`tenant_members(tenant_id, user_id, role default 'owner', pk(tenant_id,user_id))`;
`monitors(id uuid pk, tenant_id, name, url, method default 'GET', interval_seconds int
check 60..300, timeout_ms default 10000, expected_status default 200, failure_threshold
default 2, enabled bool, state text default 'unknown' /*unknown|up|down*/,
consecutive_fails int default 0, last_checked_at timestamptz, created_at, updated_at)`;
`monitor_results(id bigserial, monitor_id, tenant_id, checked_at, ok bool, status_code,
latency_ms, error text)` + index `(monitor_id, checked_at desc)`, 30-day purge;
`incidents(id uuid, tenant_id, monitor_id, started_at, resolved_at nullable, cause,
notify_state default 'pending' /*pending|notified|recovered_notified*/)`;
`alert_channels(id uuid, tenant_id, type check in ('email','slack_webhook'),
config jsonb, enabled bool, created_at)`.

RLS on every tenant table: `USING (tenant_id = current_setting('app.tenant_id')::uuid)`.
Two DB roles: `uptime_api` (RLS enforced, no BYPASSRLS), `uptime_scheduler`
(**BYPASSRLS** — works across tenants by design). Migrations: plain numbered SQL files
embedded (`//go:embed migrations/*.sql`) + a ~40-line runner with a `schema_migrations`
table, run at API startup (decision: no goose — avoid the dep; contact service precedent
is embedded SQL).

### B2 — services/api (chi + pgx + zerolog, listen 127.0.0.1:8081)
- Module path: `github.com/devopsaccess-in/devopsaccess.in/services/api`
  (NOTE: contact uses `devopsaccess-in/devopsaccess-in`; shared uses
  `devopsaccess-in/devopsaccess.in` — harmless mismatch, keep new modules on
  `devopsaccess.in`).
- `internal/auth`: Auth0 JWT middleware — JWKS fetch + cache (use
  `github.com/golang-jwt/jwt/v5`, hand-rolled JWKS cache ~60 lines; validates iss
  `https://<AUTH0_DOMAIN>/`, aud `https://api.devopsaccess.in`, RS256). Env:
  `AUTH0_DOMAIN`, `AUTH0_AUDIENCE`.
- First-login provisioning: on `/api/me`, look up `users` by `auth0_sub`; if missing,
  create user + personal tenant (slug from email localpart, uniquified) + membership in
  one tx.
- Endpoints (JWT unless noted): `GET /api/me`; `GET|POST /api/monitors`;
  `GET|PATCH|DELETE /api/monitors/{id}`; `GET /api/monitors/{id}/results?window=24h`
  (downsample ok to skip in v1); `GET /api/monitors/{id}/uptime?window=7d|30d`;
  `GET /api/incidents?monitor_id=`; `GET /api/incidents/{id}`; `GET|POST /api/channels`;
  `DELETE /api/channels/{id}`; `POST /api/channels/{id}/test`; **public**
  `GET /api/status/{slug}`, `GET /healthz`.
- Tenant scoping: EVERY tenant query via `db.WithTenant` (shared module, already
  written) AND explicit `tenant_id` predicates (defense in depth — project rule).
- Monitor URL validation at create/update: parse, require http(s), resolve host,
  reject non-public IPs via `safehttp.IsPublicIP` (also enforced at probe time).
- Config via env like services/contact/config.go (`envOr` pattern): `API_LISTEN_ADDR`
  (default 127.0.0.1:8081), `DATABASE_URL`, `AUTH0_DOMAIN`, `AUTH0_AUDIENCE`.

### B3 — services/scheduler (one binary: prober + alerting + notifier)
- Loop every 10s: `SELECT ... FROM monitors WHERE enabled AND (last_checked_at IS NULL
  OR last_checked_at + interval_seconds * interval '1 second' <= now()) FOR UPDATE SKIP
  LOCKED` (or simple update-claim) → dispatch to ~20-goroutine worker pool.
- Per check: GET via `safehttp.Client(timeout)`, ok = status == expected_status,
  insert `monitor_results` row, update monitor state.
- State machine: unknown→up on first success; failure → consecutive_fails++; up→down
  when consecutive_fails >= failure_threshold → INSERT incident + notify all enabled
  channels; down→up on first success → set resolved_at + recovery notice.
  `incidents.notify_state` guards duplicate sends across restarts.
- Notify: email via the mailer pattern from `services/contact/mailer.go` (localhost:25
  Postfix relay, no auth — env SMTP_HOST=127.0.0.1 SMTP_PORT=25, from
  alerts@devopsaccess.in); Slack = POST JSON `{"text": ...}` to webhook URL from
  channel config.
- Housekeeping: nightly delete `monitor_results` older than 30 days.
- Connects as `uptime_scheduler` (BYPASSRLS) — its own `DATABASE_URL`.

### B4 — apps/dashboard (Next.js 16 App Router, runs as node server :3001)
- `output: "standalone"`, systemd `next start -p 3001`. Auth0 Next.js SDK v4
  (`@auth0/nextjs-auth0`), shadcn/ui, TanStack Query, Tailwind (copy design tokens from
  apps/web tailwind.config.ts + globals.css).
- Pages: `/monitors` (list: state badge, uptime %, sparkline), `/monitors/[id]`
  (latency chart, recent results, incidents), `/incidents`, `/channels`, `/settings`,
  public `/status/[slug]` (auth-exempt in middleware, fetches public API).
- Data: TanStack Query against same-origin `/api/*` (nginx proxies to :8081 on the
  app.devopsaccess.in vhost → no CORS).

### B5 — Infra
- nginx vhost `infra/ansible/roles/nginx/templates/app.conf.j2`: server_name
  app.devopsaccess.in; `location /api/` → 127.0.0.1:8081; `location /` → 127.0.0.1:3001;
  same TLS origin-cert + AOP + security-header patterns as site.conf.j2 (check the
  Cloudflare Origin cert covers `*.devopsaccess.in`; regenerate if not — user task).
- Ansible roles mirroring `contact_api` (systemd hardened units, env file from
  template): `app_api` (binary, :8081), `app_scheduler` (binary), `app_dashboard`
  (rsync `.next/standalone` + node systemd unit). Extend `postgres` role: create
  `uptime` DB + `uptime_api`/`uptime_scheduler` roles (BYPASSRLS for scheduler;
  `ALTER ROLE uptime_scheduler BYPASSRLS`).
- Terraform: add `app` A record (proxied) in `infra/terraform` (find the existing
  Cloudflare record resources and mirror them).
- CI `.github/workflows/deploy-app.yml`: build Go binaries **GOOS=linux GOARCH=amd64**
  (the VM is amd64 — configure.yml precedent, do NOT use arm64), build dashboard
  standalone, run the new roles by tag via the same Terraform-IP + Ansible pattern as
  deploy-web.yml.
- New CI secrets the user must add: `AUTH0_DOMAIN`, `AUTH0_AUDIENCE`,
  `AUTH0_CLIENT_ID`, `AUTH0_CLIENT_SECRET`, `AUTH0_SECRET` (session encryption),
  `UPTIME_DB_PASSWORD` (or reuse POSTGRES_PASSWORD pattern with new role passwords).

### E2E gate (defines "done")
Signup at app.devopsaccess.in → tenant auto-provisioned → add email + Slack channels →
create 60s monitor with threshold 2 on a disposable endpoint → break it → within ~3 min
incident opens + email + Slack arrive → fix it → recovery notice → `/status/{slug}`
shows history → second Auth0 user CANNOT read first tenant's monitor by id (expect 404;
RLS integration test with build tag too).

### Cut list (do NOT build)
Deploy engine, OTel/APM ingestor, gRPC/protos, CLI, multi-region probes, Razorpay
subscription automation, SMS/voice paging, on-call schedules, non-HTTP check types,
team invites, custom status domains, Hindi UI, k3s/ArgoCD, Kafka/NATS/Redis.

## 4. Exact current state of Phase B (uncommitted, on `feat/app/uptime-mvp`)

Done so far (committed on `feat/app/uptime-mvp`):
- `services/shared/` — module `github.com/devopsaccess-in/devopsaccess.in/services/shared`,
  tidied, builds + vets green:
  - `db/db.go` — `Connect()` (pool + ping retry) and `WithTenant()`
    (tx + `set_config('app.tenant_id', $1, true)`).
  - `safehttp/safehttp.go` — `IsPublicIP`, `Dialer`, `Client` (ported from
    services/contact/safehttp.go with configurable timeouts).
- `go.work` — `use (./services/contact ./services/shared)`; ADD services/api and
  services/scheduler to it as you create them.

Immediate next steps (in order):
1. Create `services/api` per B2 (go.mod, main.go, config.go, migrate.go +
   `migrations/0001_init.sql` with the full schema+RLS from B1, internal/auth,
   internal/store, handlers). Add `replace`/require of the shared module via go.work
   (workspace resolves it; go.mod still needs the require line —
   `go mod tidy` inside the workspace handles it).
3. Create `services/scheduler` per B3.
4. `go build ./... && go vet ./...` in each module; table-driven unit tests for the
   state machine and JWKS middleware (CLAUDE.md: every public function gets one).
5. apps/dashboard per B4 (pnpm workspace member; remember root uses pnpm-workspace.yaml).
6. Infra per B5.
7. Commit in logical units (Conventional Commits, NO Co-Authored-By, no emojis),
   push branch via `github-personal` remote.

## 5. Gotchas / house rules (from bitter experience this session)

- **Old repo `gh` = office account.** Git pushes: both repos use the `github-personal`
  SSH alias. PRs + GitHub secrets: browser only.
- The production VM is **amd64** (all CI Go builds use GOOS=linux GOARCH=amd64).
- The Edit/Write tools require Read first — read any file (even ones you just wrote
  via Bash) before editing.
- Ansible runs from `infra/ansible/`; `site_dist_local: "../../apps/web/out"`.
- pnpm workspace: root `pnpm-workspace.yaml` lists `apps/*`, `packages/*`.
- Next.js 16: `params` is a Promise — `await params` in pages/generateMetadata.
- Velite emits to `.velite/`, imported via `#velite` tsconfig path alias (web app).
- contact_api role expects CI to build the binary into
  `infra/ansible/roles/contact_api/files/contact` — same pattern for new roles.
- Tests: never delete/disable to pass. TestGrade-style thresholds live in
  services/contact — leave contact service behavior unchanged.
- Cost discipline: no new paid services. Auth0 free tier + existing VM only.

## 6. User-side pending tasks (remind Vikram, don't do)

- Phase A cutover: add secrets/vars to the monorepo repo (list in §2 of the deploy-web
  workflow + TF_CLOUD_TOKEN, HETZNER_SSH_KEY, CLOUDFLARE_*, SMTP_*, POSTGRES_PASSWORD,
  RAZORPAY_WEBHOOK_SECRET, TURNSTILE_SECRET, GRAFANA_ADMIN_PASSWORD, POSTHOG_KEY/HOST,
  vars TURNSTILE_SITEKEY, GA4_ID, RAZORPAY_PAYMENT_URL), merge
  `feat/web/nextjs-migration`, run Deploy Web + Configure Server (tags=nginx), test the
  3 forms on prod, re-run PSI (CLS ≤ 0.02), resubmit `/sitemap.xml` in Search Console,
  then disable the old repo's deploy workflow.
- Auth0 tenant setup (B6 of plan): Regular Web App, callbacks
  `https://app.devopsaccess.in/auth/callback` + localhost, API audience
  `https://api.devopsaccess.in` RS256, Google + email/password connections.
- Cloudflare: verify Origin cert covers `*.devopsaccess.in`; add `app` DNS record is
  in Terraform (CI applies it).

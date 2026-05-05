# DevOps Access — Project Memory

> This file is read by Claude Code at the start of every session. It defines project context, architecture, conventions, and "gotchas" specific to this codebase. For personal coding preferences, see `~/.claude-personal/CLAUDE.md`.

## What We're Building

DevOps Access is a unified DevOps lifecycle SaaS platform that replaces fragmented tools (Datadog, PagerDuty, Render/Railway, UptimeRobot, New Relic) with a single affordable product. Target market: Indian startups (5-50 engineers) that currently overspend on 4-5 separate tools.

**Core value proposition:**
- Single pane of glass for hosting + monitoring + alerting + APM + on-call + CI/CD
- Starts at $29/mo for small teams (vs. $200+ for equivalent stack elsewhere)
- India-first: INR pricing, timezone-aware on-call, Indian support hours
- Cloud-agnostic by design — customers can BYOC to AWS/GCP/Azure

**Near-term goal:** First paying customer within 90 days, ₹10,000/month total infrastructure + SaaS budget cap.

## Stack

### Backend (Go)
- Go 1.23+ with go workspace (`go.work`)
- `chi` for HTTP routing
- `pgx` v5 for Postgres
- `zerolog` for structured logging
- `otelhttp` + `otelgrpc` for OpenTelemetry instrumentation
- `cobra` for CLI (`packages/cli`)
- gRPC + Protobuf for inter-service comms (`packages/protos`)

### Frontend (Next.js)
- Next.js 14+ (App Router, NOT Pages)
- React 18 with Server Components
- Tailwind CSS + shadcn/ui
- Zod for input validation
- TanStack Query for client data fetching
- Auth0 for authentication (planned — not yet integrated)

### Infrastructure
- Hetzner Cloud (primary, cheap, EU-based) — see Cloud Strategy Decision page in Notion
- k3s (lightweight Kubernetes) managed via k3sup + Terraform
- Cloudflare: DNS, CDN, WAF, Workers (for probe network), R2 (for Terraform state + assets)
- ArgoCD for GitOps
- Prometheus + Grafana + Loki + Tempo + OpenTelemetry Collector for observability
- Kyverno for policy enforcement, Cilium for CNI+network policies
- Razorpay for payments (Indian market)

### Multi-tenancy model
- **Free tier:** Shared k3s cluster, namespace-per-tenant isolation
- **Paid tier:** Can opt for BYOC (customer deploys control plane to their own cloud) — our platform orchestrates via a customer-installed agent

## Architecture Overview
[Customer App] --OTel--> [Ingestor] --> [Postgres + Object Store]
↓
[Customer Dashboard] <--gRPC-- [API] <--gRPC-- [Query Layer]
↓
[Notifier] --> [Email, SMS, Slack]
↑
[Scheduler] (cron, SLO evaluation)
- `services/api` — main control plane API (auth, tenant mgmt, billing, CRUD)
- `services/ingestor` — accepts OpenTelemetry traces/logs/metrics from customer apps
- `services/notifier` — dispatches alerts via email/SMS/Slack/webhook
- `services/scheduler` — evaluates SLOs, runs probe schedules, cron jobs
- `apps/web` — marketing site (devopsaccess.in)
- `apps/dashboard` — SaaS dashboard (app.devopsaccess.in)

## Hard Constraints

1. **₹10,000/month infrastructure + SaaS budget** — verify every new dependency or service. Use the `cost-check` skill.
2. **Cloud-agnostic** — NO Aurora, BigQuery, Cosmos DB, DynamoDB, or any other proprietary datastore. Use Postgres, S3-compatible storage, Redis, Kafka/NATS. Migration to any cloud should take 1-2 days.
3. **No Co-Authored-By Claude in commits** — already enforced in `.claude/settings.json`.
4. **India-first** — default timezone Asia/Kolkata, default currency INR, support for English + Hindi UI strings.
5. **Multi-tenant isolation is non-optional** — every query must be scoped by `tenant_id`. Use Postgres RLS + app-layer checks.

## Conventions

### Git & PRs
- Branch naming: `<type>/<scope>/<short-description>` (e.g., `feat/api/rate-limiting`, `fix/ingestor/gzip-crash`)
- Conventional Commits for messages (see `commit-style` skill)
- PR template: context, changes, testing, risks, rollback plan
- Merge strategy: squash-merge to main, preserve PR description as the commit body

### Go code
- Context first parameter everywhere: `func Foo(ctx context.Context, ...)`
- Errors wrapped with context: `fmt.Errorf("creating tenant: %w", err)`
- No panics in library/service code; panics only acceptable in `main` for fatal startup failures
- Tests live alongside code (`foo_test.go` next to `foo.go`), package-internal
- Integration tests under `services/<svc>/internal/integration_test.go` with build tag `// +build integration`

### TypeScript / Next.js
- App Router conventions: `page.tsx`, `layout.tsx`, `loading.tsx`, `error.tsx`
- Server Components by default; `"use client"` only for interactivity
- API routes under `app/api/` for RPC-style; prefer server actions for form mutations
- Shared types in `packages/ui/src/types.ts` or `packages/protos/gen/ts/`

### Terraform
- Modules MUST be pure (no hardcoded secrets, all inputs via variables)
- State stored in Cloudflare R2 (S3-compatible, free up to 10GB)
- Workspaces for prod/staging, not separate repos
- Always run `terraform fmt && terraform validate` before commit
- Always run `terraform plan` and paste output in PR description before apply

### Testing philosophy
- Every public function in services gets at least one table-driven unit test
- Integration tests required for: auth flows, payment flows, tenant isolation, alert delivery
- E2E (Playwright) required for: signup, first deploy, first alert, payment
- Coverage is not a goal; meaningful tests are. Don't test framework code.

## Non-Goals (Don't Suggest These)

- Kubernetes operators before we have 100 paying customers
- GraphQL (we're using gRPC + REST; no need for a third paradigm)
- Microservices for the sake of microservices — start with 4 services, split when one is clearly doing too much
- Kafka before we genuinely need event sourcing — start with Postgres NOTIFY / LISTEN
- Service mesh (Istio/Linkerd) before it's painful to live without — Cilium's L7 policies cover most needs

## Where Things Live

- Task board → Notion: search for "Master Task Board"
- CEO + CTO roadmaps → project knowledge docs
- Cloud strategy decision → Notion page under Launch Command Center
- Architecture diagrams → `docs/architecture/`
- Runbooks → `docs/runbooks/` (also mirrored to Notion)
- ADRs (Architecture Decision Records) → `docs/adrs/NNNN-title.md`

## Open Questions (as of April 2026)

- [ ] Auth: Auth0 vs self-hosted Keycloak vs Supabase Auth — currently leaning Auth0 free tier
- [ ] Payments: Razorpay domestic-only vs Stripe India — currently Razorpay (2% per txn, no monthly fee)
- [ ] First feature to build post-scaffolding: incident triage (Week 2) or uptime monitoring (Week 1)?
- [ ] BYOC agent: Go binary or k8s operator? (Probably Go binary Phase 1, operator Phase 2.)

## Contacts

- Founder: Vikram (vikram@devopsaccess.in)
- Support: support@devopsaccess.in
- Status page: (TBD — will be status.devopsaccess.in on Cloudflare Workers)

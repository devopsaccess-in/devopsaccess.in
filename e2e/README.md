# End-to-end tests

Black-box tests that build and run the real `services/api` and
`services/scheduler` binaries (and optionally the dashboard's node server)
against a disposable Postgres, a fake Auth0 (local JWKS + minted RS256
tokens), a local SMTP sink, and a local Slack-webhook sink. They cover the
product features listed in [`FEATURES.md`](../FEATURES.md) — every feature PR
updates both.

## Run

Needs a **disposable** Postgres superuser (the harness creates/drops the
`uptime_e2e` database and manages the `uptime_api` / `uptime_scheduler`
roles — do not point it at a database you care about):

```sh
docker run --rm -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres --name e2e-pg postgres:16
go test -tags e2e -timeout 10m ./e2e
```

- `E2E_ADMIN_DATABASE_URL` — admin DSN
  (default `postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable`)
- `E2E_DASHBOARD_DIR` — directory containing the dashboard's standalone
  `server.js` (optional; the public status-page test is skipped without it)

CI runs this via `.github/workflows/e2e.yml` on every PR.

The services run with `UPTIME_ALLOW_PRIVATE_TARGETS=true` and
`SCHEDULER_TICK_SECONDS=1` — documented test hooks that let monitors point at
local target servers and keep the loop fast. The harness also rewinds
`monitors.last_checked_at` directly ("expedite") so the 60s minimum check
interval doesn't stretch the suite.

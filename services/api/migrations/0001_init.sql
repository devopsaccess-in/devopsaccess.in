-- 0001_init.sql — uptime MVP schema.
--
-- Tenancy model: identity tables (tenants, users, tenant_members) carry no
-- RLS because they must be readable before a tenant context exists — login
-- provisioning looks up users by auth0_sub, and the public status page
-- resolves a tenant by slug. Every tenant DATA table enforces RLS keyed on
-- app.tenant_id, which db.WithTenant sets per-transaction. FORCE ROW LEVEL
-- SECURITY makes the policies apply to the table owner too (migrations run as
-- uptime_api, which is also the query role).
--
-- Roles (created by the Ansible postgres role, NOT here):
--   uptime_api       — RLS enforced (no BYPASSRLS), owns this schema
--   uptime_scheduler — BYPASSRLS: probes and alerts work across all tenants

CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    slug       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth0_sub  TEXT NOT NULL UNIQUE,
    email      TEXT NOT NULL DEFAULT '',
    name       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tenant_members (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      TEXT NOT NULL DEFAULT 'owner',
    PRIMARY KEY (tenant_id, user_id)
);

CREATE TABLE monitors (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    url               TEXT NOT NULL,
    method            TEXT NOT NULL DEFAULT 'GET' CHECK (method IN ('GET', 'HEAD')),
    interval_seconds  INT NOT NULL DEFAULT 60 CHECK (interval_seconds BETWEEN 60 AND 300),
    timeout_ms        INT NOT NULL DEFAULT 10000 CHECK (timeout_ms BETWEEN 1000 AND 30000),
    expected_status   INT NOT NULL DEFAULT 200 CHECK (expected_status BETWEEN 100 AND 599),
    failure_threshold INT NOT NULL DEFAULT 2 CHECK (failure_threshold BETWEEN 1 AND 10),
    enabled           BOOLEAN NOT NULL DEFAULT true,
    state             TEXT NOT NULL DEFAULT 'unknown' CHECK (state IN ('unknown', 'up', 'down')),
    consecutive_fails INT NOT NULL DEFAULT 0,
    last_checked_at   TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX monitors_tenant_idx ON monitors (tenant_id);
-- Prober due-scan: WHERE enabled AND (last_checked_at IS NULL OR due).
CREATE INDEX monitors_due_idx ON monitors (last_checked_at) WHERE enabled;

CREATE TABLE monitor_results (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    monitor_id  UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    checked_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    ok          BOOLEAN NOT NULL,
    status_code INT,
    latency_ms  INT,
    error       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX monitor_results_monitor_time_idx ON monitor_results (monitor_id, checked_at DESC);
-- 30-day retention enforced by the scheduler's nightly purge.

CREATE TABLE incidents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    monitor_id   UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at  TIMESTAMPTZ,
    cause        TEXT NOT NULL DEFAULT '',
    notify_state TEXT NOT NULL DEFAULT 'pending'
                 CHECK (notify_state IN ('pending', 'notified', 'recovered_notified'))
);
CREATE INDEX incidents_tenant_idx ON incidents (tenant_id, started_at DESC);
CREATE INDEX incidents_monitor_idx ON incidents (monitor_id, started_at DESC);
-- At most one open incident per monitor; the state machine relies on this.
CREATE UNIQUE INDEX incidents_open_unique ON incidents (monitor_id) WHERE resolved_at IS NULL;

CREATE TABLE alert_channels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type       TEXT NOT NULL CHECK (type IN ('email', 'slack_webhook')),
    config     JSONB NOT NULL DEFAULT '{}',
    enabled    BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX alert_channels_tenant_idx ON alert_channels (tenant_id);

-- RLS: app.tenant_id is set per-transaction by db.WithTenant. The missing_ok
-- form of current_setting returns NULL when unset, so a connection without a
-- tenant context sees zero rows instead of erroring.
ALTER TABLE monitors ENABLE ROW LEVEL SECURITY;
ALTER TABLE monitors FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON monitors
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE monitor_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE monitor_results FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON monitor_results
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE incidents ENABLE ROW LEVEL SECURITY;
ALTER TABLE incidents FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON incidents
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

ALTER TABLE alert_channels ENABLE ROW LEVEL SECURITY;
ALTER TABLE alert_channels FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON alert_channels
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- Scheduler grants (BYPASSRLS is a role attribute, set by Ansible). Guarded so
-- the migration also applies on a bare local dev database without the role.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uptime_scheduler') THEN
        GRANT SELECT ON tenants TO uptime_scheduler;
        GRANT SELECT, UPDATE ON monitors TO uptime_scheduler;
        GRANT SELECT, INSERT, DELETE ON monitor_results TO uptime_scheduler;
        GRANT SELECT, INSERT, UPDATE ON incidents TO uptime_scheduler;
        GRANT SELECT ON alert_channels TO uptime_scheduler;
        GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO uptime_scheduler;
    END IF;
END $$;

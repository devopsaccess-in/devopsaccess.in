-- 0004_heartbeat_monitors.sql — heartbeat ("dead man's switch") monitors.
--
-- An http monitor is us calling you. A heartbeat monitor is you calling us:
-- a cron job, backup script, or ETL pipeline pings a secret URL at the end of
-- every run, and we alert when a ping does NOT arrive within
-- period_seconds + grace_seconds. It catches the failures nobody sees — the
-- nightly backup that silently stopped running three weeks ago.
--
-- Heartbeats reuse the whole existing pipeline: the scheduler evaluates them
-- on the same claim cadence (interval_seconds), writes the same
-- monitor_results rows, and drives the same state machine, incidents, and
-- notifications. Only the "is it healthy" step differs.

ALTER TABLE monitors
    ADD COLUMN kind           TEXT NOT NULL DEFAULT 'http'
        CHECK (kind IN ('http', 'heartbeat')),
    -- How often the caller promises to ping: 1 minute .. 7 days.
    ADD COLUMN period_seconds INT NOT NULL DEFAULT 3600
        CHECK (period_seconds BETWEEN 60 AND 604800),
    -- Slack allowed on top of the period before we consider it late.
    ADD COLUMN grace_seconds  INT NOT NULL DEFAULT 300
        CHECK (grace_seconds BETWEEN 30 AND 86400),
    -- The unguessable capability in the ping URL. NULL for http monitors.
    ADD COLUMN ping_token     TEXT,
    ADD COLUMN last_ping_at   TIMESTAMPTZ;

-- An http monitor needs a URL; a heartbeat needs a token instead.
ALTER TABLE monitors
    ADD CONSTRAINT monitors_http_needs_url
        CHECK (kind <> 'http' OR url <> ''),
    ADD CONSTRAINT monitors_heartbeat_needs_token
        CHECK (kind <> 'heartbeat' OR ping_token IS NOT NULL);

CREATE UNIQUE INDEX monitors_ping_token_idx ON monitors (ping_token)
    WHERE ping_token IS NOT NULL;

-- Public token -> tenant lookup.
--
-- monitors carries RLS, so a request with no tenant context (which is exactly
-- what an unauthenticated ping is) can never see a row there. This table is
-- deliberately RLS-free for the same reason tenants.slug is: it is what
-- ESTABLISHES the tenant context, and the token itself is the capability.
-- It holds no monitor data beyond the mapping, and cascades with the monitor
-- so a deleted heartbeat's token stops working immediately.
CREATE TABLE heartbeat_tokens (
    token      TEXT PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX heartbeat_tokens_monitor_idx ON heartbeat_tokens (monitor_id);

-- A late heartbeat is a new kind of failure, so widen the phase vocabulary
-- from 0003 (the CHECK there does not know about it).
ALTER TABLE monitor_results
    DROP CONSTRAINT monitor_results_failure_phase_check,
    ADD CONSTRAINT monitor_results_failure_phase_check
        CHECK (failure_phase IN ('', 'dns', 'tcp', 'tls', 'timeout', 'status',
                                 'request', 'blocked', 'redirect', 'response',
                                 'heartbeat'));

-- The ping endpoint updates last_ping_at and resolves open incidents; the
-- scheduler role already holds UPDATE on monitors and incidents.

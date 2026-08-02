-- 0005_audit_log.sql — tenant-scoped audit trail.
--
-- Answers "who changed what, when" for a customer asking why their monitor
-- disappeared or why they stopped getting alerts. Written in the SAME
-- transaction as the mutation it records, so the trail can never diverge from
-- reality: if the change rolled back, so did its audit row.
--
-- actor_sub is the Auth0 subject of the user who did it, or '' for actions
-- the system took on its own (the scheduler opening and resolving incidents).

CREATE TABLE audit_log (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_sub  TEXT NOT NULL DEFAULT '',
    actor_email TEXT NOT NULL DEFAULT '',
    action     TEXT NOT NULL,
    entity_id  UUID,
    -- Human-readable one-liner, rendered at write time so the log stays
    -- readable after the entity it refers to is deleted.
    summary    TEXT NOT NULL DEFAULT '',
    details    JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_tenant_time_idx ON audit_log (tenant_id, created_at DESC);

ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON audit_log
    USING (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::uuid);

-- The scheduler (BYPASSRLS) records incident open/resolve events.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uptime_scheduler') THEN
        GRANT SELECT, INSERT, DELETE ON audit_log TO uptime_scheduler;
        GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO uptime_scheduler;
    END IF;
END $$;

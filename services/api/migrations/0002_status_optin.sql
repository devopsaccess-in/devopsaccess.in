-- 0002_status_optin.sql — public status pages are opt-in.
--
-- Tenant slugs are derived from the signup email localpart, which is
-- guessable, so a status page must NOT be reachable until the owner turns it
-- on. Existing rows default to FALSE (private) — no tenant is exposed by the
-- upgrade. The /api/status/{slug} endpoint 404s unless this is true.
ALTER TABLE tenants
    ADD COLUMN public_status_enabled BOOLEAN NOT NULL DEFAULT false;

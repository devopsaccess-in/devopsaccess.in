-- 0003_deep_probe.sql — per-phase probe timings, TLS certificate tracking,
-- and machine-readable failure phases.
--
-- Every check now records where the time went (DNS -> TCP -> TLS -> first
-- byte) and, for https targets, the leaf certificate's expiry. That turns a
-- bare "down" into "TLS certificate expired 2 days ago" and lets us warn
-- before a cert takes the site down.

-- Phase timings are nullable: a phase that did not happen has no timing
-- (no DNS for an IP literal, no TLS for plain http, no TTFB when the request
-- never got a response).
ALTER TABLE monitor_results
    ADD COLUMN dns_ms        INT,
    ADD COLUMN connect_ms    INT,
    ADD COLUMN tls_ms        INT,
    ADD COLUMN ttfb_ms       INT,
    ADD COLUMN failure_phase TEXT NOT NULL DEFAULT ''
        CHECK (failure_phase IN ('', 'dns', 'tcp', 'tls', 'timeout', 'status',
                                 'request', 'blocked', 'redirect', 'response'));

-- Observed leaf certificate for https monitors.
-- tls_warned_threshold is the expiry-warning rung already sent (0 = none,
-- 14 = the 14-day warning, 3 = the 3-day warning). It resets to 0 when a
-- certificate with a later expiry is observed (i.e. the cert was renewed),
-- which makes the warnings idempotent across scheduler restarts.
ALTER TABLE monitors
    ADD COLUMN tls_expires_at       TIMESTAMPTZ,
    ADD COLUMN tls_issuer           TEXT NOT NULL DEFAULT '',
    ADD COLUMN tls_warned_threshold INT NOT NULL DEFAULT 0;

-- Scheduler scans for certificates nearing expiry.
CREATE INDEX monitors_tls_expiry_idx ON monitors (tls_expires_at)
    WHERE enabled AND tls_expires_at IS NOT NULL;

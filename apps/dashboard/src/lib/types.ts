// API response shapes — mirror services/api/internal/store JSON tags.

export type Monitor = {
  id: string;
  name: string;
  kind: "http" | "heartbeat";
  period_seconds: number;
  grace_seconds: number;
  ping_token: string | null;
  last_ping_at: string | null;
  url: string;
  method: string;
  interval_seconds: number;
  timeout_ms: number;
  expected_status: number;
  failure_threshold: number;
  enabled: boolean;
  state: "unknown" | "up" | "down";
  consecutive_fails: number;
  last_checked_at: string | null;
  tls_expires_at: string | null;
  tls_issuer: string;
  created_at: string;
  updated_at: string;
};

export type CheckResult = {
  id: number;
  monitor_id: string;
  checked_at: string;
  ok: boolean;
  status_code: number | null;
  latency_ms: number | null;
  error: string;
  dns_ms: number | null;
  connect_ms: number | null;
  tls_ms: number | null;
  ttfb_ms: number | null;
  failure_phase: string;
};

export type Incident = {
  id: string;
  monitor_id: string;
  monitor_name: string;
  started_at: string;
  resolved_at: string | null;
  cause: string;
};

export type Channel = {
  id: string;
  type: "email" | "slack_webhook";
  config: { to?: string; url?: string };
  enabled: boolean;
  created_at: string;
};

export type Me = {
  user: { id: string; email: string; name: string; created_at: string };
  tenant: {
    id: string;
    name: string;
    slug: string;
    public_status_enabled: boolean;
    created_at: string;
  };
};

export type Uptime = {
  window: string;
  ok: number;
  total: number;
  uptime_pct: number | null;
};

export type AuditEntry = {
  id: number;
  actor_email: string;
  action: string;
  entity_id: string | null;
  summary: string;
  details: Record<string, unknown>;
  created_at: string;
};

export type StatusPage = {
  name: string;
  slug: string;
  monitors: {
    id: string;
    name: string;
    state: "unknown" | "up" | "down";
    uptime_pct: number | null;
  }[];
  incidents: Incident[];
};

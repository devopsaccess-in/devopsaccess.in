// Curated DevOps & AI/automation tools we rate. Content-only (no backend).
// Add/curate freely — these are honest starter entries; expand with your own picks.
export interface Tool {
  name: string;
  category: "Automation" | "AI Agents" | "Observability" | "CI/CD" | "Platform";
  take: string; // one-line opinion
  url: string;
  selfHost: boolean;
  free: boolean;
}

export const tools: Tool[] = [
  {
    name: "n8n",
    category: "Automation",
    take: "Fair-code workflow automation you can self-host — the pragmatic glue for ops + AI steps without per-task SaaS billing.",
    url: "https://n8n.io",
    selfHost: true,
    free: true,
  },
  {
    name: "Windmill",
    category: "Automation",
    take: "Open-source developer platform: turn scripts into workflows, APIs and UIs. Great when n8n's node model gets limiting.",
    url: "https://windmill.dev",
    selfHost: true,
    free: true,
  },
  {
    name: "Prometheus + Grafana",
    category: "Observability",
    take: "The CNCF default for metrics + dashboards. Self-hosted, no per-host bill — what we reach for first on FinOps engagements.",
    url: "https://prometheus.io",
    selfHost: true,
    free: true,
  },
  {
    name: "Argo CD",
    category: "CI/CD",
    take: "GitOps for Kubernetes done right — declarative, auditable deploys with easy rollback.",
    url: "https://argo-cd.readthedocs.io",
    selfHost: true,
    free: true,
  },
  // TODO(vikram): add your AI-agent picks here (e.g. the agent frameworks you
  // wanted to feature). Keep entries to tools you'd genuinely recommend.
];

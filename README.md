# DevOps Access

Unified DevOps lifecycle platform for Indian startups — hosting, monitoring, alerting, APM, on-call paging, and CI/CD in one affordable product.

Live: [devopsaccess.in](https://devopsaccess.in)

## Repo Layout

- `apps/` — Next.js frontends (marketing + dashboard)
- `services/` — Go backend services
- `packages/` — Shared libraries and CLI
- `infra/` — Terraform (Hetzner + Cloudflare), Helm charts, ArgoCD apps
- `docs/` — Technical docs + user docs
- `scripts/` — Dev tooling and one-off scripts

## Prerequisites

- Node.js 20.17+ (see `.nvmrc`)
- Go 1.23+ (see `.tool-versions`)
- pnpm 9+
- Docker (for local services)
- kubectl, helm, terraform (for infra work)
- `mise` or `asdf` recommended for version management

## Quick Start

```bash
# Install JS deps
pnpm install

# Bootstrap Go modules (once services are scaffolded)
go work sync

# Run web app
pnpm --filter web dev

# Run API service locally
cd services/api && go run ./cmd/api

# Run everything (via Turbo)
pnpm dev
```

## Contributing

Solo founder project at this stage. See `CLAUDE.md` for coding standards.

## License

Proprietary. All rights reserved.

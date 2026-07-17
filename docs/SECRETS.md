# GitHub secrets & variables — monorepo setup

Everything the four workflows (Provision Infrastructure, Configure Server,
Deploy Web, Deploy App, E2E) need. The old `devopsaccess-in` repo already
holds most of these — copy the values across in the browser (repo → Settings
→ Secrets and variables → Actions). **Never use the `gh` CLI for this: it is
signed in to the office account.**

## Secrets to copy from the old devopsaccess-in repo (same names)

| Secret | Used by | Purpose |
|---|---|---|
| `TF_CLOUD_TOKEN` | all deploy/provision | Terraform Cloud API token (state) |
| `HCLOUD_TOKEN` | infra.yml | Hetzner Cloud API |
| `HETZNER_SSH_KEY` | all deploy/provision | private key for deploy@VM |
| `CLOUDFLARE_ORIGIN_CERT` | nginx via ansible | Origin cert (wildcard: apex + `*.devopsaccess.in` — grafana + app vhosts reuse it) |
| `CLOUDFLARE_ORIGIN_KEY` | nginx via ansible | Origin cert private key |
| `CLOUDFLARE_API_TOKEN` | deploy-web.yml | cache purge |
| `CLOUDFLARE_ZONE_ID` | deploy-web.yml | cache purge |
| `POSTGRES_PASSWORD` | postgres role | contact-service DB role password |
| `SMTP_USER` / `SMTP_PASS` | mail_relay, contact | Postfix smarthost auth to Gmail |
| `RAZORPAY_WEBHOOK_SECRET` | contact | payments webhook |
| `TURNSTILE_SECRET` | contact | form verification |
| `GRAFANA_ADMIN_PASSWORD` | monitoring | Grafana admin |
| `POSTHOG_KEY` / `POSTHOG_HOST` | deploy-web.yml | analytics (build-time) |

## Variables to copy (same names)

| Variable | Used by |
|---|---|
| `TURNSTILE_SITEKEY` | deploy-web.yml |
| `GA4_ID` | deploy-web.yml |
| `RAZORPAY_PAYMENT_URL` | deploy-web.yml |

## NEW secrets (uptime app — do not exist anywhere yet)

| Secret | Used by | How to get it |
|---|---|---|
| `UPTIME_DB_PASSWORD` | postgres/app roles | generate (one password for both `uptime_api` and `uptime_scheduler` roles) |
| `AUTH0_DOMAIN` | app_api, app_dashboard | Auth0 tenant domain, e.g. `devopsaccess.eu.auth0.com` (no scheme) |
| `AUTH0_CLIENT_ID` | app_dashboard | Auth0 Regular Web App |
| `AUTH0_CLIENT_SECRET` | app_dashboard | Auth0 Regular Web App |
| `AUTH0_SECRET` | app_dashboard | session encryption — `openssl rand -hex 32` |

`AUTH0_AUDIENCE` needs no secret: it defaults to `https://api.devopsaccess.in`
in every config.

## Cloudflare (dashboard, manual — DNS is not in Terraform)

- A records, all proxied, pointing at the VM IP (Terraform output
  `server_ipv4`): apex `devopsaccess.in`, `grafana`, **`app` (new)**.
- Origin cert: the existing one already serves `grafana.devopsaccess.in` via
  the same `/etc/nginx/tls/origin.pem`, so it is the default wildcard and
  covers `app.` — verify once in Cloudflare → SSL/TLS → Origin Server.
- Firewall: nothing to do — the Hetzner firewall already admits all
  Cloudflare IPs on 80/443, fetched live at every terraform apply.

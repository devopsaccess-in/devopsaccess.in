---
name: terraform-hetzner
description: Use this skill whenever the user asks to write, modify, review, or deploy Terraform code that provisions Hetzner Cloud resources for DevOps Access. Covers hcloud provider patterns, k3s bootstrap, firewall rules, load balancer config, and Cloudflare DNS integration. Triggers on phrases like "Terraform for Hetzner", "hcloud resource", "provision a server", "add a firewall", "Hetzner module", "k3s cluster on Hetzner".
---

# DevOps Access Terraform + Hetzner Conventions

## Provider setup

Use the official `hetznercloud/hcloud` provider v1.48+ and `cloudflare/cloudflare` v4.40+.

```hcl
terraform {
required_version = ">= 1.9"
required_providers {
hcloud = {
source  = "hetznercloud/hcloud"
version = "~> 1.48"
}
cloudflare = {
source  = "cloudflare/cloudflare"
version = "~> 4.40"
}
}
backend "s3" {
# Cloudflare R2 (S3-compatible). Credentials via env vars:
#   AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY (R2 access key pair)
bucket                      = "devopsaccess-tfstate"
key                         = "environments/prod/terraform.tfstate"
region                      = "auto"
endpoints                   = { s3 = "https://<ACCOUNT_ID>.r2.cloudflarestorage.com" }
skip_credentials_validation = true
skip_region_validation      = true
skip_metadata_api_check     = true
skip_requesting_account_id  = true
use_path_style              = true
}
}provider "hcloud" {
HCLOUD_TOKEN env var
}provider "cloudflare" {
CLOUDFLARE_API_TOKEN env var
}
## Standard resource patterns

### k3s control node

```hcl
resource "hcloud_server" "k3s_master" {
name        = "k3s-master-${var.environment}"
image       = "ubuntu-24.04"
server_type = var.master_server_type  # CAX11 dev, CAX21 prod
location    = "fsn1"                  # Falkenstein — cheapest, lowest India latency
ssh_keys    = [hcloud_ssh_key.default.id]
firewall_ids = [hcloud_firewall.k3s_cluster.id]user_data = templatefile("${path.module}/cloud-init/k3s-master.yaml", {
k3s_version = var.k3s_version
})labels = {
role        = "k3s-master"
environment = var.environment
managed_by  = "terraform"
}
}
### Cloudflare DNS linking to Hetzner load balancer

```hcl
resource "cloudflare_record" "api" {
zone_id = var.cloudflare_zone_id
name    = "api"
value   = hcloud_load_balancer.main.ipv4
type    = "A"
proxied = true  # WAF + DDoS + CDN
ttl     = 1     # auto, required when proxied
}
## Module structure

Every module under `infra/terraform/modules/<name>/`:

- `main.tf` — resources
- `variables.tf` — all inputs with descriptions and validation
- `outputs.tf` — exported values
- `versions.tf` — required_providers pinning
- `README.md` — usage example, required vars, outputs
- `cloud-init/` (if module bootstraps VMs)

## Hard rules

1. Never hardcode `hcloud_token` or `cloudflare_api_token`. Read from `HCLOUD_TOKEN` and `CLOUDFLARE_API_TOKEN` env vars.
2. Every resource gets a `labels` block with at minimum `environment` and `managed_by = "terraform"`.
3. Default location is `fsn1` (Falkenstein). Use `nbg1` or `hel1` only for redundancy.
4. Prefer ARM (CAX series) over AMD (CPX series) — ~40% cheaper, sufficient perf for control plane.
5. Enable `delete_protection` on production load balancers and primary databases.
6. Always run `terraform plan` and paste output in PR description before apply.

## Common pitfalls

- Hetzner rate-limits API calls at 3600/hour. Wait 10 min if `plan`/`apply` hits the limit.
- Cloudflare `proxied = true` breaks direct IP health checks. Use CF Worker health probes instead.
- `hcloud_load_balancer_service` target_type `label_selector` requires all targets in the same network. Attach LB to private network before adding services.
- SSH keys must be uploaded to Hetzner BEFORE the server references them. Add `depends_on` if ordering gets tangled.
- R2 backend requires a dedicated API token with R2 Read & Write scope. Don't reuse your main `CLOUDFLARE_API_TOKEN`.

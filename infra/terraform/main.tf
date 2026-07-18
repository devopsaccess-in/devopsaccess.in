terraform {
  required_version = ">= 1.6.0"

  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.49"
    }
    http = {
      source  = "hashicorp/http"
      version = "~> 3.4"
    }
  }

  # ---------------------------------------------------------------------------
  # Remote state — Terraform Cloud free tier.
  # Create a workspace at app.terraform.io → set to "API-driven workflow".
  # Add TF_CLOUD_TOKEN as a GitHub secret (user API token from Terraform Cloud).
  # ---------------------------------------------------------------------------
  cloud {
    organization = "devopsaccess" # ← replace with your Terraform Cloud org name

    workspaces {
      name = "devopsaccess-tf"
    }
  }
}

provider "hcloud" {
  token = var.hcloud_token
}

# ---------------------------------------------------------------------------
# SSH key — uploaded from the public key you store in the HETZNER_SSH_KEY
# GitHub secret (public half). The private half stays in GitHub secrets and
# is used by Ansible to connect.
# ---------------------------------------------------------------------------
resource "hcloud_ssh_key" "default" {
  name       = "${var.server_name}-key"
  public_key = var.ssh_public_key
}

# ---------------------------------------------------------------------------
# Cloudflare published IP ranges, fetched live on every apply so the firewall
# is always current with zero drift (no hardcoded list to rot). Web ports
# (80/443) accept traffic only from these, so the origin can't be hit directly
# — all requests must pass through Cloudflare's proxy/WAF.
# ---------------------------------------------------------------------------
data "http" "cf_ips_v4" {
  url = "https://www.cloudflare.com/ips-v4"
}

data "http" "cf_ips_v6" {
  url = "https://www.cloudflare.com/ips-v6"
}

locals {
  cloudflare_ips = concat(
    split("\n", trimspace(data.http.cf_ips_v4.response_body)),
    split("\n", trimspace(data.http.cf_ips_v6.response_body)),
  )
}

# ---------------------------------------------------------------------------
# Firewall — SSH (custom port) open; HTTP/HTTPS restricted to Cloudflare.
# ---------------------------------------------------------------------------
resource "hcloud_firewall" "web" {
  name = "${var.server_name}-fw"

  # SSH stays open (CI provisioning connects directly, not via Cloudflare).
  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = var.ssh_port
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "80"
    source_ips = local.cloudflare_ips
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "443"
    source_ips = local.cloudflare_ips
  }

  # Allow ICMP (ping) for diagnostics
  rule {
    direction  = "in"
    protocol   = "icmp"
    source_ips = ["0.0.0.0/0", "::/0"]
  }
}

# ---------------------------------------------------------------------------
# The server itself. cx22 = 2 vCPU / 4GB. For 1 vCPU / 2GB use cx11 (older)
# or cax11 (ARM). Default kept configurable via var.server_type.
# ---------------------------------------------------------------------------
resource "hcloud_server" "web" {
  name        = var.server_name
  image       = var.image
  server_type = var.server_type
  location    = var.location

  ssh_keys = [hcloud_ssh_key.default.id]

  # NOTE: the firewall is intentionally NOT attached at creation. A fresh box
  # runs sshd on port 22, but the firewall only allows the hardened port
  # (var.ssh_port). Attaching here would block port 22 before Ansible can move
  # sshd off it ("SSH never came up"). The firewall is attached afterwards via
  # hcloud_firewall_attachment, gated by var.attach_firewall. See infra.yml.

  labels = {
    project = "devopsaccess"
    managed = "terraform"
  }

  # Minimal cloud-init: ensure python3 exists for Ansible, nothing else.
  # All real configuration is done by Ansible afterwards.
  user_data = <<-EOF
    #cloud-config
    package_update: true
    packages:
      - python3
  EOF
}

# ---------------------------------------------------------------------------
# Firewall attachment — deferred. Phase-1 apply (var.attach_firewall = false)
# creates the server with port 22 open so Ansible can connect and harden sshd
# onto var.ssh_port. Phase-3 apply (var.attach_firewall = true) then attaches
# the firewall, blocking port 22 at the network edge.
# ---------------------------------------------------------------------------
resource "hcloud_firewall_attachment" "web" {
  count       = var.attach_firewall ? 1 : 0
  firewall_id = hcloud_firewall.web.id
  server_ids  = [hcloud_server.web.id]
}

resource "hcloud_rdns" "web_v4" {
  server_id  = hcloud_server.web.id
  ip_address = hcloud_server.web.ipv4_address
  dns_ptr    = var.domain_name
}

variable "hcloud_token" {
  description = "Hetzner Cloud API token"
  type        = string
  sensitive   = true
}

variable "ssh_public_key" {
  description = "Public SSH key uploaded to Hetzner (public half of HETZNER_SSH_KEY)"
  type        = string
}

variable "server_name" {
  description = "Name of the server"
  type        = string
  default     = "devopsaccess-web"
}

variable "server_type" {
  description = "Hetzner server type. cx22 = 2vCPU/4GB. For 1vCPU/2GB use cax11 (ARM, cheapest) or cx11."
  type        = string
  default     = "cx23"
}

variable "image" {
  description = "OS image"
  type        = string
  default     = "ubuntu-24.04"
}

variable "location" {
  description = "Hetzner datacenter location (nbg1, fsn1, hel1, ash, hil)"
  type        = string
  default     = "fsn1"
}

variable "ssh_port" {
  description = "SSH port (hardened away from 22)"
  type        = string
  default     = "2222"
}

variable "domain_name" {
  description = "Primary domain, used for reverse DNS"
  type        = string
  default     = "devopsaccess.in"
}

variable "attach_firewall" {
  description = "Whether to attach the firewall. Set false for the bootstrap apply (before Ansible moves sshd off port 22), true afterwards."
  type        = bool
  default     = true
}

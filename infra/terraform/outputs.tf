output "server_ipv4" {
  description = "Public IPv4 address of the server"
  value       = hcloud_server.web.ipv4_address
}

output "server_ipv6" {
  description = "Public IPv6 address of the server"
  value       = hcloud_server.web.ipv6_address
}

output "server_name" {
  value = hcloud_server.web.name
}

output "server_status" {
  value = hcloud_server.web.status
}

output "ssh_port" {
  description = "Hardened SSH port the server listens on after Ansible runs"
  value       = var.ssh_port
}

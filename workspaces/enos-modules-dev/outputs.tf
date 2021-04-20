# Consul
output "consul_instance_ids" {
  description = "IDs of Consul instances"
  value       = module.consul_cluster.instance_ids
}

output "consul_instance_private_ips" {
  description = "Private IPs of Consul instances"
  value       = module.consul_cluster.instance_private_ips
}

output "consul_instance_public_ips" {
  description = "Public IPs of Consul instances"
  value       = module.consul_cluster.instance_public_ips
}

# Vault
output "vault_instance_ids" {
  description = "IDs of vault instances"
  value       = module.vault_cluster.instance_ids
}

output "vault_instance_private_ips" {
  description = "Private IPs of Vault instances"
  value       = module.vault_cluster.instance_private_ips
}

output "vault_instance_public_ips" {
  description = "Public IPs of Vault instances"
  value       = module.vault_cluster.instance_public_ips
}

output "vault_token" {
  description = "value"
  value       = module.vault_cluster.vault_token
}
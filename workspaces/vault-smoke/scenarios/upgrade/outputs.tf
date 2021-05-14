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

output "vault_root_token" {
  description = "The Vault cluster root token. Keep it secret. Keep it safe."
  value       = module.vault_cluster.vault_root_token
}

output "vault_recovery_keys_b64" {
  description = "The Vault cluster recovery keys. Keep them secret. Keep them safe."
  value       = module.vault_cluster.vault_recovery_keys_b64
}

output "vault_artifactory_release" {
  value = {
    url = data.enos_artifactory_item.vault.results[0].url
    sha256 = data.enos_artifactory_item.vault.results[0].sha256
  }
}

output "enos_vault_instance_ids" {
  description = "IDs of Vault instances"
  value       = aws_instance.vault_instance.*.id
}

output "enos_vault_instance_publicips" {
  description = "Public IPs of Vault instances"
  value       = aws_instance.vault_instance.*.public_ip
}

output "enos_vault_instance_privateips" {
  description = "Private IPs of Vault instances"
  value       = aws_instance.vault_instance.*.private_ip
}
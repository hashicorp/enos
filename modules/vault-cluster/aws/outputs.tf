output "instance_ids" {
  description = "IDs of Vault instances"
  value       = aws_instance.vault_instance.*.id
}

output "instance_public_ips" {
  description = "Public IPs of Vault instances"
  value       = aws_instance.vault_instance.*.public_ip
}

output "instance_private_ips" {
  description = "Private IPs of Vault instances"
  value       = aws_instance.vault_instance.*.private_ip
}
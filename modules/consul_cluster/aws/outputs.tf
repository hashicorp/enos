output "enos_consul_instance_ids" {
  description = "IDs of Consul instances"
  value       = aws_instance.consul_instance.*.id
}

output "enos_consul_instance_privateips" {
  description = "Private IPs of Vault instances"
  value       = aws_instance.consul_instance.*.private_ip
}

output "enos_consul_instance_publicips" {
  description = "Public IPs of Vault instances"
  value       = aws_instance.consul_instance.*.public_ip
}
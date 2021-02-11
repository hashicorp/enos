output "instance_ids" {
  description = "IDs of Consul instances"
  value       = aws_instance.consul_instance.*.id
}

output "instance_private_ips" {
  description = "Private IPs of Consul instances"
  value       = aws_instance.consul_instance.*.private_ip
}

output "instance_public_ips" {
  description = "Public IPs of Consul instances"
  value       = aws_instance.consul_instance.*.public_ip
}
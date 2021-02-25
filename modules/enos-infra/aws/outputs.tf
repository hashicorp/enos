output "aws_region" {
  value = data.aws_region.current.name
}

output "vpc_id" {
  value = aws_vpc.enos_vpc.id
}

output "ubuntu_ami_id" {
  value = data.aws_ami.ubuntu.id
}

output "availability_zone_names" {
  value = data.aws_availability_zones.available.names
}

output "account_id" {
  value = data.aws_caller_identity.current.account_id
}
output "kms_key_arn" {
  value = aws_kms_key.enos_key.arn
}

output "kms_key_alias" {
  value = aws_kms_alias.enos_key_alias.name
}

output "vault_license" {
  value     = data.aws_kms_secrets.enos.plaintext["vault_license"]
  sensitive = true
}

variable "project_name" {
  description = "Name of the project."
  type        = string
}

variable "environment" {
  description = "Name of the environment."
  type        = string
}

variable "common_tags" {
  description = "Tags to set for all resources"
  type        = map(string)
}

variable "instance_type" {
  description = "EC2 Instance"
  type        = string
  default     = "t2.micro"
}

variable "instance_count" {
  description = "Number of EC2 instances in each subnet"
  type        = number
  default     = 3
}

variable "ssh_aws_keypair" {
  description = "SSH keypair used to connect to EC2 instances"
  type        = string
}

variable "ubuntu_ami_id" {
  description = "Ubuntu LTS AMI from enos-infra"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID from enos-infra"
  type        = string
}

variable "package_url" {
  type        = string
  default     = "https://releases.hashicorp.com/vault/1.6.2+ent/vault_1.6.2+ent_linux_amd64.zip"
  description = "(optional) describe your variable"
}

variable "consul_ips" {
  type        = list(any)
  description = "(optional) describe your variable"
}

variable "vault_license" {
  type        = string
  sensitive   = true
  description = "vault license"
}

variable "kms_key_arn" {
  type        = string
  description = "ARN of KMS Key from enos-infra"
}

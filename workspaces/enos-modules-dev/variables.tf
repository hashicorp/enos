variable "aws_region" {
  description = "AWS default Region"
  type        = string
  default     = "us-east-1"
}

variable "aws_availability_zone" {
  description = "AWS availability zone"
  type        = string
  default     = ""
}

variable "project_name" {
  description = "Name of the project."
  type        = string
  default     = "qti-enos"
}

variable "environment" {
  description = "Name of the environment."
  type        = string
  default     = "dev"
}

variable "common_tags" {
  description = "Tags to set for all resources"
  type        = map(string)
  default = {
    "Project Name" : "qti-enos",
    "Environment" : "dev"
  }
}

variable "ssh_aws_keypair" {
  description = "SSH keypair used to connect to EC2 instances"
  type        = string
  default     = "qti-aws-keypair"
}

variable "base_vault_version" {
  type        = string
  description = "The starting version of Vault to install"
  default     = "1.6.2"
}

variable "upgrade_vault_version" {
  type        = string
  description = "The desired version of vault to upgrade to"
  default     = "1.7.0"
}

variable "vault_instance_count" {
  type        = string
  description = "The number of instances to provision for the Vault cluster"
  default     = 3
}

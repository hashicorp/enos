variable "aws_region" {
  description = "AWS default Region"
  type        = string
  default     = "us-east-1"
}

variable "aws_availability_zone" {
  description = "AWS availability zone"
  type        = string
  default     = "us-east-1a"
}

variable "aws_ssh_key_pair_name" {
  description = "SSH key pair used to connect to EC2 instances"
  type        = string
}

variable "aws_ssh_private_key_path" {
  description = "The path to the private key of the key pair used to connect to EC2 instances"
  type        = string
}

variable "project_name" {
  description = "Name of the project."
  type        = string
  default     = "vault-enterprise-smoke-verify-license"
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
    "Project Name" : "vault-enterprise-smoke-verify-license",
    "Environment" : "dev"
  }
}

variable "consul_install_dir" {
  type        = string
  description = "The directory where the Consul binary will be installed"
  default     = "/opt/consul/bin"
}

variable "consul_release" {
  type = object({
    version = string
    edition = string
  })
  description = "Consul release version and edition to install from releases.hashicorp.com"
  default = {
    version = "1.9.5"
    edition = "oss"
  }
}

variable "consul_license_path" {
  type        = string
  description = "The path to the Consul Enterprise license if using the ent edition"
  default     = null
}

variable "vault_install_dir" {
  type        = string
  description = "The directory where the Vault binary will be installed"
  default     = "/opt/vault/bin"
}

variable "vault_license_path" {
  type        = string
  description = "The path to the Vault Enterprise license if using the ent edition"
  default     = null
}

variable "vault_instance_count" {
  type = number
  description = "The number of instances to provision for Vault"
  default = 3
}

variable "vault_artifactory_release" {
  type = object({
    username = string
    token = string
    host = string
    repo = string
    path = string
    name = string
    properties = map(string)
  })
  description = "Vault release version and edition to install from artifactory.hashicorp.engineering"
  default = null
}

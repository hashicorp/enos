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
  default     = "vault-enterprise-smoke-upgrade"
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
    "Project Name" : "vault-enterprise-smoke-upgrade",
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

variable "vault_enterprise_initial_release" {
  type = object({
    edition = string
    version = string
  })
  description = "Vault Enterprise release version to install prior to upgrade"
  default = {
    edition = "ent"
    version = "1.7.0"
  }
}

variable "vault_oss_initial_release" {
  type = object({
    edition = string
    version = string
  })
  description = "Vault release version to install prior to upgrade"
  default = {
    edition = "oss"
    version = "1.7.0"
  }
}

variable "artifactory_username" {
  type = string
  description = "The Artifactory username for authenticating with Artifactory"
  default = null
}

variable "artifactory_token" {
  type = string
  description = "The Artifactory token for authenticating with Artifactory"
  default = null
}

variable "vault_enterprise_product_revision" {
  type = string
  description = "The product revision used when staging the release. This is probably the git SHA"
  default = null
}

variable "vault_oss_product_revision" {
  type = string
  description = "The product revision used when staging the release. This is probably the git SHA"
  default = null
}

variable "vault_product_version" {
  type = string
  description = "The product revision used when staging the release. This is probably the git SHA"
  default = null
}

variable "vault_product_editions_to_test" {
  type = list(string)
  description = "The product editions to test"
  default = ["oss", "ent", "ent.hsm", "prem", "prem.hsm", "pro"]
}

variable "vault_artifactory_release_query" {
  type = object({
    host = string
    repo = string
    name = string
    properties = map(string)
  })
  description = "Vault release version and edition to upgrade from artifactory.hashicorp.engineering"
  default = {
    host       = "https://artifactory.hashicorp.engineering/artifactory"
    repo       = "hashicorp-packagespec-buildcache-local*"
    name       = "*.zip"
    properties = {
      "GOARCH"          = "amd64"
      "GOOS"            = "linux"
      "artifactType"    = "package"
      # EDITION should be the enterprise edition
      # productRevision should be the git SHA of the staged release
      # productVersion should be the version of Vault
    }
  }
}

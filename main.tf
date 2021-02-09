terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "hashicorp-qti"

    workspaces {
      prefix = "enos-modules-"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = var.aws_region
}

module "consul_cluster" {
  source = "./modules/consul-cluster/aws"

  project_name = var.project_name
  environment = var.environment
  common_tags = var.common_tags
  ssh_aws_keypair = var.ssh_aws_keypair
}

module "vault_cluster" {
  source = "./modules/vault-cluster/aws"

  project_name = var.project_name
  environment = var.environment
  common_tags = var.common_tags
  ssh_aws_keypair = var.ssh_aws_keypair
}
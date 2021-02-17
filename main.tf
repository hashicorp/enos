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

module "enos_infra" {
  source       = "./modules/enos-infra/aws"
  project_name = var.project_name
  environment  = var.environment
  common_tags  = var.common_tags
}

module "consul_cluster" {
  source     = "./modules/consul-cluster/aws"
  depends_on = [module.enos_infra]

  project_name    = var.project_name
  environment     = var.environment
  common_tags     = var.common_tags
  ssh_aws_keypair = var.ssh_aws_keypair
  ubuntu_ami_id   = module.enos_infra.ubuntu_ami_id
  vpc_id          = module.enos_infra.vpc_id
  kms_key_arn     = module.enos_infra.kms_key_arn
}

module "vault_cluster" {
  source     = "./modules/vault-cluster/aws"
  depends_on = [module.enos_infra]

  project_name    = var.project_name
  environment     = var.environment
  common_tags     = var.common_tags
  ssh_aws_keypair = var.ssh_aws_keypair
  ubuntu_ami_id   = module.enos_infra.ubuntu_ami_id
  vpc_id          = module.enos_infra.vpc_id
}

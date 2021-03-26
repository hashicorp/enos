terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }

   enos = {
      version = "~> 0.1"
      source  = "hashicorp.com/qti/enos"
    }
  }

  backend "remote" {
    hostname     = "app.terraform.io"
    organization = "hashicorp-qti"

    workspaces {
      name = "enos-modules-dev"
    }
  }
}

provider "enos" {
  transport = {
    ssh = {
      # You can also override any of the transport settings from the environment,
      # e.g.: ENOS_TRANSPORT_PRIVATE_KEY_PATH=/path/to/key.pem
      user = "ubuntu"
      private_key_path = "./qti-aws-keypair.pem"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = var.aws_region
}

module "enos_infra" {
  source            = "./modules/enos-infra/aws"

  project_name      = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags
  availability_zone = var.aws_availability_zone
}

module "consul_cluster" {
  source     = "./modules/consul-cluster/aws"
  depends_on = [module.enos_infra]

  project_name      = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags
  ssh_aws_keypair   = var.ssh_aws_keypair
  ubuntu_ami_id     = module.enos_infra.ubuntu_ami_id
  vpc_id            = module.enos_infra.vpc_id
  availability_zone = var.aws_availability_zone
  kms_key_arn       = module.enos_infra.kms_key_arn
  consul_license    = file("${path.root}/consul.hclic")
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
  kms_key_arn     = module.enos_infra.kms_key_arn
  consul_ips      = module.consul_cluster.instance_private_ips
  vault_license   = file("${path.root}/vault.hclic")
}

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    enos = {
      version = ">= 0.1.1"
      source  = "hashicorp.com/qti/enos"
    }
  }
}

provider "enos" {
  transport = {
    ssh = {
      # You can also override any of the transport settings from the environment,
      # e.g.: ENOS_TRANSPORT_PRIVATE_KEY_PATH=/path/to/key.pem
      user             = "ubuntu"
      private_key_path = var.aws_ssh_private_key_path
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = var.aws_region
}

# Build our core infrastructure
module "enos_infra" {
  source  = "app.terraform.io/hashicorp-qti/aws-infra/enos"
  version = ">= 0.0.2"

  project_name      = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags
  availability_zone = var.aws_availability_zone
}

# Find the staged build in artifactory
data "enos_artifactory_item" "vault" {
  username   = var.vault_artifactory_release.username
  token      = var.vault_artifactory_release.token
  host       = var.vault_artifactory_release.host
  repo       = var.vault_artifactory_release.repo
  path       = var.vault_artifactory_release.path
  name       = var.vault_artifactory_release.name
  properties = var.vault_artifactory_release.properties
}

# Build the Consul backend
module "consul_cluster" {
  # source  = "../../../../../terraform-enos-aws-consul"
  source  = "app.terraform.io/hashicorp-qti/aws-consul/enos"
  version = ">= 0.1.7"

  depends_on = [module.enos_infra]

  project_name      = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags

  ssh_aws_keypair    = var.aws_ssh_key_pair_name
  ubuntu_ami_id      = module.enos_infra.ubuntu_ami_id
  vpc_id             = module.enos_infra.vpc_id
  availability_zone  = var.aws_availability_zone
  kms_key_arn        = module.enos_infra.kms_key_arn
  consul_install_dir = var.consul_install_dir
  consul_license     = var.consul_license_path != null ? file(var.consul_license_path) : null
  consul_release     = var.consul_release
}

# Build the Vault cluster
# Note: we don't set a license for this Vault cluster because the verify license
# smoke tests # are designed to verify the default license.
module "vault_cluster" {
  #source  = "../../../../../terraform-enos-aws-vault"
  source  = "app.terraform.io/hashicorp-qti/aws-vault/enos"
  version = ">= 0.0.8"

  depends_on = [
    module.enos_infra,
    module.consul_cluster,
  ]

  project_name              = var.project_name
  environment               = var.environment
  common_tags               = var.common_tags
  ssh_aws_keypair           = var.aws_ssh_key_pair_name
  ubuntu_ami_id             = module.enos_infra.ubuntu_ami_id
  vpc_id                    = module.enos_infra.vpc_id
  kms_key_arn               = module.enos_infra.kms_key_arn
  instance_count            = var.vault_instance_count
  consul_ips                = module.consul_cluster.instance_private_ips
  vault_license             = var.vault_license_path != null ? file(var.vault_license_path) : null
  vault_install_dir         = var.vault_install_dir
  vault_release             = null
  vault_artifactory_release = {
    url      = data.enos_artifactory_item.vault.results[0].url
    sha256   = data.enos_artifactory_item.vault.results[0].sha256
    username = var.vault_artifactory_release.username
    token    = var.vault_artifactory_release.token
  }
}

# Run the smoke tests
resource "enos_remote_exec" "smoke-verify-license" {
  depends_on = [module.vault_cluster]
  for_each   = toset([for idx in range(var.vault_instance_count) : tostring(idx)])

  content = templatefile("${path.module}/templates/smoke-verify-license.sh", {
    vault_install_dir = var.vault_install_dir,
    vault_token       = module.vault_cluster.vault_root_token,
    vault_version     = var.vault_artifactory_release.properties["productVersion"]
    vault_edition     = var.vault_artifactory_release.properties["EDITION"]
  })

  transport = {
    ssh = {
      host = module.vault_cluster.instance_public_ips[tonumber(each.value)]
    }
  }
}

resource "enos_remote_exec" "smoke-write-test-data" {
  depends_on = [enos_remote_exec.smoke-verify-license]

  content = templatefile("${path.module}/templates/smoke-write-test-data.sh", {
    test_key          = "smoke"
    test_value        = "fire"
    vault_install_dir = var.vault_install_dir,
    vault_token       = module.vault_cluster.vault_root_token,
  })

  transport = {
    ssh = {
      host = module.vault_cluster.instance_public_ips[0]
    }
  }
}

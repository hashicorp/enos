terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    enos = {
      version = ">= 0.1.3"
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

locals {
  // vault_instances is a map of each Vault instance we're going to create, the
  // edition of vault that will be installed, and the vault cluster where it can
  // be found.
  vault_instances = {for i in setproduct(var.vault_product_editions_to_test, range(var.vault_instance_count)):
    "${i[0]}-${i[1]}" => {
      edition     = i[0]
      instance_idx = i[1]
      cluster_idx  = index(var.vault_product_editions_to_test, i[0])
    }
  }
}

# Find the staged bundles in artifactory
data "enos_artifactory_item" "vault" {
  count = length(var.vault_product_editions_to_test)

  username   = var.artifactory_username
  token      = var.artifactory_token
  name       = var.vault_artifactory_release_query.name
  host       = var.vault_artifactory_release_query.host
  repo       = var.vault_artifactory_release_query.repo
  path       = var.vault_product_editions_to_test[count.index] == "oss" ? "cache-v1/vault/*" : "cache-v1/vault-enterprise/*"
  properties = var.vault_product_editions_to_test[count.index] == "oss" ? merge(var.vault_artifactory_release_query.properties, {
      "productRevision" = var.vault_oss_product_revision
      "productVersion"  = var.vault_product_version
    }) : merge(var.vault_artifactory_release_query.properties, {
      "EDITION"         = var.vault_product_editions_to_test[count.index]
      "productRevision" = var.vault_enterprise_product_revision
      "productVersion"  = var.vault_product_version
    })
}

# Build our core infrastructure
module "infra" {
  #source  = "../../../../../terraform-enos-aws-infra"
  source  = "app.terraform.io/hashicorp-qti/aws-infra/enos"
  version = ">= 0.0.2"

  project_name      = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags
  availability_zone = var.aws_availability_zone
}

# Build the Consul backend
module "consul" {
  #source  = "../../../../../terraform-enos-aws-consul"
  source  = "app.terraform.io/hashicorp-qti/aws-consul/enos"
  version = ">= 0.1.8"

  depends_on = [module.infra]
  count      = length(var.vault_product_editions_to_test)

  project_name      = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags

  ssh_aws_keypair    = var.aws_ssh_key_pair_name
  ubuntu_ami_id      = module.infra.ubuntu_ami_id
  vpc_id             = module.infra.vpc_id
  availability_zone  = var.aws_availability_zone
  kms_key_arn        = module.infra.kms_key_arn
  consul_install_dir = var.consul_install_dir
  consul_license     = var.consul_license_path != null ? file(var.consul_license_path) : null
  consul_release     = var.consul_release
}

# Build the Vault cluster
module "vault" {
  #source  = "../../../../../terraform-enos-aws-vault"
  source  = "app.terraform.io/hashicorp-qti/aws-vault/enos"
  version = ">= 0.0.9"

  depends_on = [
    module.infra,
    module.consul,
  ]
  count = length(var.vault_product_editions_to_test)

  project_name              = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags
  ssh_aws_keypair   = var.aws_ssh_key_pair_name
  ubuntu_ami_id     = module.infra.ubuntu_ami_id
  vpc_id            = module.infra.vpc_id
  kms_key_arn       = module.infra.kms_key_arn
  instance_count    = var.vault_instance_count
  consul_ips        = module.consul[count.index].instance_private_ips
  vault_license     = var.vault_license_path != null ? file(var.vault_license_path) : null
  vault_install_dir = var.vault_install_dir
  vault_release     = var.vault_product_editions_to_test[count.index] == "oss" ? merge(var.vault_oss_initial_release, {
    product = "vault"
  }) : merge(var.vault_enterprise_initial_release, {
    product = "vault"
  })
}

resource "enos_bundle_install" "upgrade_vault_binary" {
  depends_on = [module.vault]
  for_each   = local.vault_instances

  destination = var.vault_install_dir
  artifactory = {
    url      = data.enos_artifactory_item.vault[each.value.cluster_idx].results[0].url
    sha256   = data.enos_artifactory_item.vault[each.value.cluster_idx].results[0].sha256
    username = var.artifactory_username
    token    = var.artifactory_token
  }

  transport = {
    ssh = {
      host = module.vault[each.value.cluster_idx].instance_public_ips[each.value.instance_idx]
    }
  }
}

# The documented process for a vault cluster upgrade: stop and upgrade the standbys, then do
# the same on the primary node after the standbys are back up and running
resource "enos_remote_exec" "upgrade_standby" {
  depends_on = [enos_bundle_install.upgrade_vault_binary]
  for_each   = local.vault_instances

  content = templatefile("${path.module}/templates/vault-upgrade.sh", {
    vault_install_dir = var.vault_install_dir,
    upgrade_target    = "standby"
  })

  transport = {
    ssh = {
      host = module.vault[each.value.cluster_idx].instance_public_ips[each.value.instance_idx]
    }
  }
}

resource "enos_remote_exec" "upgrade_active" {
  depends_on = [enos_remote_exec.upgrade_standby]
  for_each   = local.vault_instances

  content = templatefile("${path.module}/templates/vault-upgrade.sh", {
    vault_install_dir = var.vault_install_dir,
    upgrade_target    = "active"
  })

  transport = {
    ssh = {
      host = module.vault[each.value.cluster_idx].instance_public_ips[each.value.instance_idx]
    }
  }
}

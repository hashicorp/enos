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
  path       = var.vault_product_editions_to_test[count.index] == "oss" ?  "cache-v1/vault/*" : "cache-v1/vault-enterprise/*"
  properties = var.vault_product_editions_to_test[count.index] == "oss" ?  merge(var.vault_artifactory_release_query.properties, {
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
  source  = "app.terraform.io/hashicorp-qti/aws-infra/enos"
  version = ">= 0.0.2"

  #count = length(var.vault_product_editions_to_test)

  project_name      = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags
  availability_zone = var.aws_availability_zone
}

# Build the Consul backend
module "consul" {
  source  = "../../../../../terraform-enos-aws-consul"
  #source  = "app.terraform.io/hashicorp-qti/aws-consul/enos"
  #version = ">= 0.1.7"

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
# Note: we don't set a license for this Vault cluster because the verify license
# smoke tests # are designed to verify the default license.
module "vault" {
  source  = "../../../../../terraform-enos-aws-vault"
  #source  = "app.terraform.io/hashicorp-qti/aws-vault/enos"
  #version = ">= 0.0.8"

  count      = length(var.vault_product_editions_to_test)
  depends_on = [
    module.infra,
    module.consul,
  ]

  project_name              = var.project_name
  environment               = var.environment
  common_tags               = var.common_tags
  ssh_aws_keypair           = var.aws_ssh_key_pair_name
  ubuntu_ami_id             = module.infra.ubuntu_ami_id
  vpc_id                    = module.infra.vpc_id
  kms_key_arn               = module.infra.kms_key_arn
  instance_count            = var.vault_instance_count
  consul_ips                = module.consul[count.index].instance_private_ips
  vault_license             = var.vault_license_path != null ? file(var.vault_license_path) : null
  vault_install_dir         = var.vault_install_dir
  vault_release             = null
  vault_artifactory_release = {
    url      = data.enos_artifactory_item.vault[count.index].results[0].url
    sha256   = data.enos_artifactory_item.vault[count.index].results[0].sha256
    username = var.artifactory_username
    token    = var.artifactory_token
  }
}

# Verify that a fresh install of our edition with a Consul backend works. Verify
# that the default license (if applicable) has the correct capabilities.
resource "enos_remote_exec" "smoke-verify-license" {
  depends_on = [module.vault]
  for_each   = local.vault_instances

  content = templatefile("${path.module}/templates/smoke-verify-license.sh", {
    vault_install_dir = var.vault_install_dir,
    vault_token       = module.vault[each.value.cluster_idx].vault_root_token,
    vault_version     = var.vault_product_version
    vault_edition     = each.value.edition
  })

  transport = {
    ssh = {
      host = module.vault[each.value.cluster_idx].instance_public_ips[each.value.instance_idx]
    }
  }
}

resource "enos_remote_exec" "smoke-enable-secrets-kv" {
  depends_on = [enos_remote_exec.smoke-verify-license]
  for_each   = local.vault_instances

  content = templatefile("${path.module}/templates/smoke-enable-secrets-kv.sh", {
    instance_id       = each.value.instance_idx
    vault_install_dir = var.vault_install_dir,
    vault_token       = module.vault[each.value.cluster_idx].vault_root_token,
  })

  transport = {
    ssh = {
      host = module.vault[each.value.cluster_idx].instance_public_ips[each.value.instance_idx]
    }
  }
}
# Verify that we can enable the k/v secrets engine and write data to it.
resource "enos_remote_exec" "smoke-write-test-data" {
  depends_on = [enos_remote_exec.smoke-enable-secrets-kv]
  for_each   = local.vault_instances

  content = templatefile("${path.module}/templates/smoke-write-test-data.sh", {
    test_key          = "smoke${each.value.instance_idx}"
    test_value        = "fire"
    vault_install_dir = var.vault_install_dir,
    vault_token       = module.vault[each.value.cluster_idx].vault_root_token,
  })

  transport = {
    ssh = {
      host = module.vault[each.value.cluster_idx].instance_public_ips[each.value.instance_idx]
    }
  }
}

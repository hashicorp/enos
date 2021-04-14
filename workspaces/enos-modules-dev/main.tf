terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    enos = {
      version = "0.1.0"
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
      user             = "ubuntu"
      private_key_path = "${path.root}/qti-aws-keypair.pem"
    }
  }
}

# Configure the AWS Provider
provider "aws" {
  region = var.aws_region
}

module "enos_infra" {
  source  = "app.terraform.io/hashicorp-qti/aws-infra/enos"
  version = "0.0.1"

  project_name      = var.project_name
  environment       = var.environment
  common_tags       = var.common_tags
  availability_zone = var.aws_availability_zone
}

module "consul_cluster" {
  source  = "app.terraform.io/hashicorp-qti/aws-consul/enos"
  version = "0.0.2"

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

// Depend on consul_cluster while we use consul as backend for our scenarios
// Remove or update this dependency when you change the backend
module "vault_cluster" {
  source  = "app.terraform.io/hashicorp-qti/aws-vault/enos"
  version = "0.0.3"

  depends_on = [
    module.enos_infra,
    module.consul_cluster,
  ]

  project_name    = var.project_name
  environment     = var.environment
  common_tags     = var.common_tags
  ssh_aws_keypair = var.ssh_aws_keypair
  ubuntu_ami_id   = module.enos_infra.ubuntu_ami_id
  vpc_id          = module.enos_infra.vpc_id
  kms_key_arn     = module.enos_infra.kms_key_arn
  instance_count  = var.vault_instance_count
  consul_ips      = module.consul_cluster.instance_private_ips
  vault_license   = file("${path.root}/vault.hclic")
  vault_version   = var.base_vault_version
}

resource "enos_remote_exec" "verify_vault_version" {
  depends_on = [module.vault_cluster]

  content = <<EOF
#!/bin/bash -e

version=$(vault -version | cut -d ' ' -f2)

if [[ "$version" != "v${var.base_vault_version}+ent" ]]; then
  echo "Vault version mismatch. Expected ${var.base_vault_version}, got '$version'" >&2
  exit 1
fi

exit 0
EOF

  for_each = toset([for idx in range(var.vault_instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = module.vault_cluster.instance_public_ips[each.value]
    }
  }
}

resource "enos_remote_exec" "upgrade_standby" {
  depends_on = [module.vault_cluster, enos_remote_exec.verify_vault_version]

  content = <<EOF
#!/bin/bash
exec >> /tmp/upgrade.log 2>&1
export VAULT_ADDR=http://localhost:8200

if vault status | grep "HA Mode" | grep standby;
then
    sudo systemctl stop vault
    cd /tmp
    wget https://releases.hashicorp.com/vault/${var.upgrade_vault_version}+ent/vault_${var.upgrade_vault_version}+ent_linux_amd64.zip
    sudo unzip -o vault_${var.upgrade_vault_version}+ent_linux_amd64.zip -d /usr/local/bin
    sudo setcap cap_ipc_lock=+ep /usr/local/bin/vault
    sudo systemctl start vault
    until vault status
    do
        sleep 1s
    done
fi
EOF

  for_each = toset([for idx in range(var.vault_instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = module.vault_cluster.instance_public_ips[each.value]
    }
  }
}

resource "enos_remote_exec" "upgrade_active" {
  depends_on = [module.vault_cluster, enos_remote_exec.upgrade_standby]

  content = <<EOF
#!/bin/bash
exec >> /tmp/upgrade.log 2>&1
export VAULT_ADDR=http://localhost:8200

if vault status | grep "HA Mode" | grep active;
then
    cd /tmp;
    sudo systemctl stop vault
    wget https://releases.hashicorp.com/vault/${var.upgrade_vault_version}+ent/vault_${var.upgrade_vault_version}+ent_linux_amd64.zip 
    sudo unzip -o vault_${var.upgrade_vault_version}+ent_linux_amd64.zip -d /usr/local/bin
    sudo systemctl start vault
    until vault status
    do
        sleep 1s
    done
fi
EOF

  for_each = toset([for idx in range(var.vault_instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = module.vault_cluster.instance_public_ips[each.value]
    }
  }
}
resource "enos_remote_exec" "verify_upgrade" {
  depends_on = [module.vault_cluster, enos_remote_exec.upgrade_active]

  content = <<EOF
#!/bin/bash -e

version=$(vault -version | cut -d ' ' -f2)

if [[ "$version" != "v${var.upgrade_vault_version}+ent" ]]; then
  echo "Vault upgrade version mismatch. Expected ${var.upgrade_vault_version}, got '$version'" >&2
  exit 1
fi

# The
if [ -f /etc/vault.d/tokens* ]
then
  export VAULT_ADDR=http://localhost:8200
  export VAULT_TOKEN=$(cat /etc/vault.d/tokens*)
  vault read secret/test || exit 1
fi

exit 0
EOF

  for_each = toset([for idx in range(var.vault_instance_count) : tostring(idx)])
  transport = {
    ssh = {
      host = module.vault_cluster.instance_public_ips[each.value]
    }
  }
}
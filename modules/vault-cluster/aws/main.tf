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
}

data "enos_environment" "localhost" {}

data "template_file" "server_hcl_template" {
  template = file("${path.module}/files/server.hcl.tpl")

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])

  vars = {
    local_ipv4 = aws_instance.vault_instance[tonumber(each.value)].private_ip
    kms_key    = data.aws_kms_key.kms_key.id
  }
}

data "template_file" "install_template" {
  template = file("${path.module}/files/install-vault.sh.tpl")

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])

  vars = {
    local_ipv4  = aws_instance.vault_instance[tonumber(each.value)].private_ip
    package_url = var.package_url
  }
}

data "template_file" "configure_template" {
  template = file("${path.module}/files/configure-vault.sh.tpl")

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])

  vars = {
    vault_license = var.vault_license
  }
}

data "template_file" "configure_consul_agent" {
  template = file("${path.module}/files/configure-consul-agent.sh.tpl")

  for_each = toset([for idx in range(var.instance_count) : tostring(idx)])

  vars = {
    consul_ips = join(" ", var.consul_ips)
  }
}

locals {
  name_suffix = "${var.project_name}-${var.environment}"
}

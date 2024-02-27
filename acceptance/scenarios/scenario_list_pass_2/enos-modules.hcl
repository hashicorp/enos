# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module "consul" {
  source = "hashicorp/consul/aws"
}

module "vault" {
  source = "hashicorp/vault/aws"
}

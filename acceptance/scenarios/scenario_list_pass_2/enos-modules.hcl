# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module "consul" {
  source = "hashicorp/consul/aws"
}

module "vault" {
  source = "hashicorp/vault/aws"
}

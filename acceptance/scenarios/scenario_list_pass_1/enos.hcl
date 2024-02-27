# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module "consul" {
  source = "hashicorp/consul/aws"
}

scenario "test" {
  step "backend" {
    module = module.consul
  }
}

# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module "consul" {
  source = "hashicorp/consul/aws"
}

module "raft" {
  source = "hashicorp/raft/aws"
}

scenario "test" {
  matrix {
    backend = ["raft", "consul"]
  }

  step "backend" {
    module = matrix.backend
  }
}

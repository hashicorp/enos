# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module "sleep" {
  source = "./modules/sleep"
}

scenario "timeout" {
  step "sleep" {
    module = module.sleep
  }
}

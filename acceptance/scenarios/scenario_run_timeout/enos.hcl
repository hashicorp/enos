# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module "sleep" {
  source = "./modules/sleep"
}

scenario "timeout" {
  step "sleep" {
    module = module.sleep
  }
}

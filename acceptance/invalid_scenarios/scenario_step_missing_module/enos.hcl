# Copyright IBM Corp. 2021, 2026
# SPDX-License-Identifier: MPL-2.0

module "foo" {
  source = "./modules/foo"

  input        = "fooinput"
  anotherinput = ["anotherfoo"]
}

module "bar" {
  source = "./modules/does_not_exist"
}

scenario "test" {
  step "foo" {
    module = module.foo
  }

  step "bar" {
    module = module.bar
  }
}

# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

module "foo" {
  source = "../scenario_generate_pass_0/modules/foo"
}

module "bar" {
  source = "../scenario_generate_pass_0/modules/bar"
}

scenario "smoke" {
  matrix {
    arch   = ["amd64", "arm64"]
    distro = ["ubuntu", "rhel"]
  }

  step "one" {
    module = module.foo

    variables {
      input        = matrix.arch
      anotherinput = matrix.distro
    }
  }
}

sample "smoke_empty_frame" {
  subset "smoke" {
    matrix {
      // Since we're filtering on a variant that does not exist our "smoke" frame will be empty.
      // That will cause validate to fail since there's no reason to include empty frames in a
      // sample.
      arch = ["not_a_variant"]
    }
  }
}

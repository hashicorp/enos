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
    arch   = ["amd64", "arm64", "aarch64", "s390x"]
    distro = ["ubuntu", "rhel", "sles", "amz"]
  }

  step "one" {
    module = module.foo

    variables {
      input        = matrix.arch
      anotherinput = matrix.distro
    }
  }
}

scenario "upgrade" {
  matrix {
    arch   = ["amd64", "arm64", "aarch64", "s390x"]
    distro = ["ubuntu", "rhel", "sles", "amz"]
  }

  step "one" {
    module = module.bar

    variables {
      input        = matrix.arch
      anotherinput = matrix.distro
    }
  }
}

sample "smoke_by_subset_name" {
  subset "smoke" {}
}

sample "smoke_by_scenario_name" {
  subset "by_scenario_name" {
    scenario_name = "smoke"
  }
}

sample "all_by_scenario_filter" {
  subset "smoke" {
    scenario_filter = "arch:aarch64 distro:amz"
  }

  subset "upgrade" {
    scenario_filter = "arch:aarch64"
  }
}

sample "all" {
  attributes = {
    aws-region        = ["us-west-1", "us-east-1"]
    continue-on-error = false
  }

  subset "smoke" {
    matrix {
      arch = ["arm64", "amd64"]
    }

    attributes = {
      notify-on-fail = true
    }
  }

  subset "smoke_allow_failure" {
    scenario_name = "smoke"

    matrix {
      arch = ["s390x"]
    }

    attributes = {
      notify-on-fail    = true
      continue-on-error = true
    }
  }

  subset "upgrade" {
  }
}

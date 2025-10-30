# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module "test" {
  source = "../scenario_generate_pass_0/modules/foo"
}

quality "the_tests_pass" {
  description = "The tests all pass!"
}

quality "the_data_is_durable" {
  description = <<-EOF
    The data is durable
    after an upgrade.
  EOF
}

scenario "singular_verifies" {
  description = <<EOF
This is a multiline description
of the upgrade scenario.
EOF

  matrix {
    arch   = ["amd64", "arm64"]
    distro = ["ubuntu", "rhel"]
  }

  step "test" {
    description = <<-EOF
      This is an indented
      multiline step description.
    EOF

    verifies = quality.the_tests_pass

    module = module.test

    variables {
      input        = matrix.arch
      anotherinput = matrix.distro
    }
  }
}

scenario "multiple_verifies" {
  description = <<EOF
This is a multiline description
of the upgrade scenario.
EOF

  matrix {
    arch   = ["amd64", "arm64"]
    distro = ["ubuntu", "rhel"]
  }

  step "test" {
    description = <<-EOF
      This is an indented
      multiline step description.
    EOF

    verifies = [
      quality.the_tests_pass,
      {
        name : "inline",
        description : "an inline quality that isn't reused",
      },
      quality.the_data_is_durable,
    ]

    module = module.test

    variables {
      input        = matrix.arch
      anotherinput = matrix.distro
    }
  }
}

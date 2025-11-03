# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

schema = "1"

project "enos" {
  team = "quality-team"
  slack {
    notification_channel = "C021AHV0R9S"
  }
  github {
    organization     = "hashicorp"
    repository       = "enos"
    release_branches = ["main"]
  }
}

# This "build" event depends on the enos "build" Github Actions workflow.
# The "build" workflow is run when a PR is merged to `main` and the version is updated.
# The "build" workflow calls the "validate" workflow, so the artifact must also pass
# acceptance testing in order to successfully complete and trigger this "build" event.

event "build" {}

event "prepare" {
  depends = ["build"]
  action "prepare" {
    organization = "hashicorp"
    repository   = "crt-workflows-common"
    workflow     = "prepare"
    depends      = ["build"]
  }

  notification {
    on = "fail"
  }
}

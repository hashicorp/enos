# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "time_sleep" "wait_5s" {
  create_duration = "5s"
}

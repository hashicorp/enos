# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

resource "random_string" "random" {
  length  = 8
  special = true
  numeric = true
}

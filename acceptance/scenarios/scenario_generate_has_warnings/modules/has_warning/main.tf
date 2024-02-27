# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

resource "random_string" "random" {
  length  = 8
  special = true
  number  = true // deprecated, should be numeric, so we'll generate a warning
}

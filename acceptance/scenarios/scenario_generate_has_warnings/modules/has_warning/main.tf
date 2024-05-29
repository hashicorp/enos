# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "3.6.1"
    }
  }
}

resource "random_string" "random" {
  length  = 8
  special = true
  number  = true // deprecated, should be numeric, so we'll generate a warning
}

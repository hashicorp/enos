# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

terraform {
  required_providers {
    enos = {
      source = "hashicorp.com/qti/enos"
    }
  }
}

variable "upstream_address" {
  type    = string
  default = "something"
}

resource "random_id" "our_address" {
  byte_length = 8
}

output "upstream_address" {
  value = random_id.our_address
}

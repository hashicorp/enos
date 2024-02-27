# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "input" {
  value = var.input
}

variable "input" {
  type    = string
  default = "notset"
}

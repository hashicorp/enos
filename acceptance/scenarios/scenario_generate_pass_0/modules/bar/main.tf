# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "input" {
  value = var.input
}

output "anotherinput" {
  value = var.input
}

variable "input" {
  type    = string
  default = "notset"
}

variable "anotherinput" {
  type    = string
  default = "notset"

  # Add a validation to ensure https://github.com/hashicorp/terraform-json/issues/106 is fixed.
  validation {
    condition     = length(var.anotherinput) > 0
    error_message = "ensure checks in state are valid"
  }
}

# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

output "input" {
  value = var.input
}

output "anotherinput" {
  value = var.input
}

variable "input" {
  type = string
  default = "notset"
}

variable "anotherinput" {
  type = list(string)
  default = ["one"]
}

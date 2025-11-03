# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

variable "ami" {
  type = string
}

output "ami" {
  value = var.ami
}

output "ips" {
  value = ["127.0.0.1"]
}

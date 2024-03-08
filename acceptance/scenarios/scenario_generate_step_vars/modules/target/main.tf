# Copyright (c) HashiCorp, Inc.
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

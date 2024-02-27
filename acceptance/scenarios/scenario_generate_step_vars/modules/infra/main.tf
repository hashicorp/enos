# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "az" {
  type    = string
  default = "us-east-1"
}

output "amis" {
  value = {
    "ubuntu" = {
      "arm" = "ubuntu-arm"
      "amd" = "ubuntu-amd"
    }
    "rhel" = {
      "arm" = "rhel-arm"
      "amd" = "rhel-amd"
    }
  }
}

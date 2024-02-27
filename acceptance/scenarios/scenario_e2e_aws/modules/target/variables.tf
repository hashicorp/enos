# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "distro" {
  type    = string
  default = null
}

variable "tags" {
  type    = map(string)
  default = null
}

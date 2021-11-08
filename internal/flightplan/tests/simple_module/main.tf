variable "length" {
  type = number
  default = 8
}

resource "random_string" "cluster_id" {
  length  = var.length
  lower   = true
  upper   = false
  number  = false
  special = false
}

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

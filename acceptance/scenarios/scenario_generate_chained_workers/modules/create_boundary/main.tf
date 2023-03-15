terraform {
  required_providers {
    enos = {
      source = "hashicorp.com/qti/enos"
    }
  }
}

variable "address" {
  type    = string
  default = "192.0.0.1:8200"
}

resource "enos_local_exec" "address" {
  inline = "echo ${var.address}"
}

output "upstream_address" {
  value = enos_local_exec.address.stdout
}

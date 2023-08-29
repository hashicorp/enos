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
}

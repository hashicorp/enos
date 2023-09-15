variable "ami" {
  type = string
}

output "ami" {
  value = var.ami
}

output "ips" {
  value = ["127.0.0.1"]
}

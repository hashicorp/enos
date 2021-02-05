variable "vpc_name" {
  type        = string
  default     = "enos-vpc"
  description = "Descriptive name of the VPC"
}

variable "vpc_cidr" {
  type        = string
  default     = "10.13.0.0/16"
  description = "CIDR for the VPC"
}

variable "project_name" {
  description = "Name of the project."
  type        = string
  default     = "qti-enos"
}

variable "environment" {
  description = "Name of the environment."
  type        = string
  default     = "dev"
}

variable "common_tags" {
  description = "Tags to set for all resources"
  type        = map(string)
  default = {
    project_name = "qti-enos"
    environment  = "dev"
  }
}

variable "ssh_pub_key" {
  description = "Public key used for login to EC2 instances"
  type = string
  default = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCqL27pBtHpg9b3y5fyqND8ODwvck0V9rG+2AJb1a6QX10PZnNiG4ItvDqppqVyDPg7StfwzUAMgMeHkGe/ahY/Pr2yKFpJgkzmmYjOiF7HtEt4IjsXT3AHI6gVLzTULdXbojOjBCOGjWxMg2PAyTVmahRMqBFZrq6kidi56prRDiDmH5HUT0MAHQUlGN7LtD7PwTNczEiqql08s51NNzNsNCzYIaDXhTvoxEgLmbs/7O5r1VrHKx3ZDTcbXTo/IyaEpjCurXhE1pVQaCCMKK9DthxGhXqi9Urw+Mi/jIHB1BAmu+lWkIcS/xiqtaLBXhjwnnNnpkeUefRkYgmiaA11"
}
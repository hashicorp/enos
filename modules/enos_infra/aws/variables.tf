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


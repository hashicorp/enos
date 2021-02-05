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
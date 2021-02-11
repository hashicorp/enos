variable "aws_region" {
  description = "AWS default Region"
  type        = string
  default     = "us-east-1"
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
    "Project Name" : "qti-enos",
    "Environment" : "dev"
  }
}

variable "ssh_aws_keypair" {
  description = "SSH keypair used to connect to EC2 instances"
  type        = string
  default     = "qti-aws-keypair"
}
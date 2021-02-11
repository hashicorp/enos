variable "project_name" {
  description = "Name of the project."
  type        = string
}

variable "environment" {
  description = "Name of the environment."
  type        = string
}

variable "common_tags" {
  description = "Tags to set for all resources"
  type        = map(string)
}

variable "instance_type" {
  description = "EC2 Instance"
  type        = string
  default     = "t3.micro"
}

variable "instance_count" {
  description = "Number of EC2 instances in each subnet"
  type        = number
  default     = 3
}

variable "ssh_aws_keypair" {
  description = "SSH keypair used to connect to EC2 instances"
  type        = string
}

variable "ubuntu_ami_id" {
  description = "Ubuntu LTS AMI from enos-infra"
  type = string
}

variable "vpc_subnet_ids" {
  description = "List of VPC Subnets from enos-infra"
  type = list
}

variable "vpc_id" {
  description = "VPC ID from enos-infra"
  type = string
}
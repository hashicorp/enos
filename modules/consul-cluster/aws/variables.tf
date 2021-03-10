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
  default     = "t2.micro"
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
  type        = string
}

variable "vpc_id" {
  description = "VPC ID from enos-infra"
  type        = string
}

variable "kms_key_arn" {
  type        = string
  description = "ARN of KMS Key from enos-infra"
}

variable "consul_license" {
  type        = string
  sensitive   = true
  description = "consul license from enos-infra"
}

variable "package_url" {
  description = "URL of Consul ZIP package to install"
  type        = string
  default     = "https://releases.hashicorp.com/consul/1.9.3+ent/consul_1.9.3+ent_linux_amd64.zip"

}

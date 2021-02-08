terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
  backend "remote" {
    hostname = "app.terraform.io"
    organization = "hashicorp-qti"

    workspaces {
      prefix = "enos-modules-"
    }
  }
}
# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

variable "tags" {
  description = "Tags to add to AWS resources"
  type        = map(string)
  default     = null
}

terraform_cli "default" {
}

terraform "default" {
  required_version = ">= 1.0.0"

  required_providers {
    enos = {
      source  = "hashicorp-forge/enos"
      version = "0.6.2"
    }

    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "aws" "default" {
  region = "us-east-1"
}

provider "aws" "west_2" {
  region = "us-west-2"
}

provider "enos" "ubuntu" {
  transport = {
    ssh = {
      user             = "ubuntu"
      private_key_path = abspath(joinpath(path.root, "../../support/private_key.pem"))
    }
  }
}

provider "enos" "rhel" {
  transport = {
    ssh = {
      user             = "ec2-user"
      private_key_path = abspath(joinpath(path.root, "../../support/private_key.pem"))
    }
  }
}

module "ec2_instance" {
  source = "./modules/target"
  tags   = var.tags
}

scenario "e2e" {
  // this matrix is overly complex to get coverage across all the stanzas
  matrix {
    distro     = ["ubuntu", "rhel"]
    aws_region = ["east"]

    include {
      distro     = ["ubuntu"]
      aws_region = ["west"]
    }

    exclude {
      distro     = ["ubuntu"]
      aws_region = ["east"]
    }
  }

  locals {
    enos_provider = {
      rhel   = provider.enos.rhel
      ubuntu = provider.enos.ubuntu
    }

    aws_provider = {
      east = provider.aws.default // should cause a warning since it's a default
      west = provider.aws.west_2
    }
  }

  terraform_cli = terraform_cli.default
  terraform     = terraform.default
  providers = [
    local.aws_provider[matrix.aws_region],
    local.enos_provider[matrix.distro],
  ]

  step "target" {
    module = module.ec2_instance

    providers = {
      aws  = local.aws_provider[matrix.aws_region]
      enos = local.enos_provider[matrix.distro]
    }

    variables {
      distro = matrix.distro
    }
  }
}

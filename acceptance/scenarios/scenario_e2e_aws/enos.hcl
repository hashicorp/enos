variable "tags" {
  description = "Tags to add to AWS resources"
  type        = map(string)
  default     = null
}

terraform_cli "default" {
  provider_installation {
    network_mirror {
      url     = "https://enos-provider-current.s3.amazonaws.com/"
      include = ["hashicorp.com/qti/enos"]
    }
    direct {
      exclude = [
        "hashicorp.com/qti/enos"
      ]
    }
  }
}

terraform "default" {
  required_version = ">= 1.0.0"

  required_providers {
    enos = {
      version = ">= 0.1.13"
      source  = "hashicorp.com/qti/enos"
    }

    aws = {
      source = "hashicorp/aws"
    }
  }
}

provider "aws" "east_1" {
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
      east = provider.aws.east_1
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

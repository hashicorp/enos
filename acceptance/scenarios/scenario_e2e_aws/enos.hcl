variable "tags" {
  description = "Tags to add to AWS resources"
  type        = map(string)
  default     = null
}

terraform_cli "default" {
  provider_installation {
    network_mirror {
      url = "https://enos-provider-current.s3.amazonaws.com/"
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

provider "aws" "east" {
  region = "us-east-1"
}

provider "aws" "west" {
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
  terraform_cli = terraform_cli.default
  terraform     = terraform.default

  providers = [
    provider.aws.east,
    provider.aws.west,
    provider.enos.rhel,
    provider.enos.ubuntu
  ]

  step "ubuntu_target" {
    module = module.ec2_instance

    providers = {
      aws  = provider.aws.east
      enos = provider.enos.ubuntu
    }

    variables {
      distro = "ubuntu"
    }
  }

  step "rhel_target" {
    module = module.ec2_instance

    providers = {
      aws  = provider.aws.west
      enos = provider.enos.rhel
    }

    variables {
      distro = "rhel"
    }
  }
}

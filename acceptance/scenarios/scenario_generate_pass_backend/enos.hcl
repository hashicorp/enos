terraform "qti_backend" {
  required_version = ">= 1.0.0"
  experiments      = ["something"]

  required_providers {
    aws = {
      version = ">= 2.7.0"
      source  = "hashicorp/aws"
    }
  }

  provider_meta "enos" {
    hello = "world"
  }

  backend "remote" {
    organization = "hashicorp-qti"

    workspaces {
      name = "enos"
    }
  }
}

terraform_cli "default" {
  env = {
    TF_LOG_CORE     = "off"
    TF_LOG_PROVIDER = "debug"
  }
}

module "test" {
  source = "./modules/test"
}

scenario "test" {
  terraform = terraform.qti_backend

  step "test" {
    module = module.test
  }
}

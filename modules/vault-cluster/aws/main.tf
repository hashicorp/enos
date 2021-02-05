provider "aws" {
  region = var.aws_region
}

locals {
  name_suffix = "${var.project_name}-${var.environment}"
}

module "enos_infra" {
  source = "../../enos_infra/aws"
}
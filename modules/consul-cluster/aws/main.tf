locals {
  name_suffix = "${var.project_name}-${var.environment}"
}

module "enos_infra" {
  source = "../../enos-infra/aws"

  project_name = var.project_name
  environment = var.environment
  common_tags = var.common_tags
}
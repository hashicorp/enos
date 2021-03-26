terraform {
  required_providers {
    # We need to specify the provider source in each module until we publish it
    enos = {
      version = "~> 0.1"
      source  = "hashicorp.com/qti/enos"
    }
  }
}

locals {
  name_suffix = "${var.project_name}-${var.environment}"
}

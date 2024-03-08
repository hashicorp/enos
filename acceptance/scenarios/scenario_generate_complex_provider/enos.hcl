# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

terraform "default" {
  required_version = ">= 1.0.0"

  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
  }
}

module "kubernetes" {
  source = "./modules/kubernetes"
}

provider "kubernetes" "default" {
  host                   = "http://example.com"
  cluster_ca_certificate = "base64cert"
  exec {
    api_version = "client.authentication.k8s.io/v1alpha1"
    args        = ["eks", "get-token", "--cluster-name", "foo"]
    command     = "aws"
  }
}

scenario "kubernetes" {
  providers = [
    provider.kubernetes.default
  ]

  step "kubernetes" {
    module = module.kubernetes
  }
}

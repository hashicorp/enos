# Copyright IBM Corp. 2021, 2025
# SPDX-License-Identifier: MPL-2.0

module "create_boundary" {
  source = "./modules/create_boundary"
}

module "create_worker" {
  source = "./modules/create_worker"
}

variable "boundary_address" {
  type = string
}

scenario "worker_chain" {
  step "create_boundary" {
    module = module.create_boundary

    variables {
      address = var.boundary_address
    }
  }

  step "create_worker" {
    depends_on = [step.create_boundary]
    module     = module.create_worker

    variables {
      upstream_address = step.create_boundary.upstream_address
    }
  }

  step "create_worker_downstream" {
    depends_on = [
      step.create_boundary,
      step.create_worker,
    ]

    module = module.create_worker

    variables {
      upstream_address = step.create_worker.upstream_address
    }
  }

  output "last_upstream" {
    value = step.create_worker_downstream.upstream_address
  }
}

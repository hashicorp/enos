variable "project" {
  type = string
}

module "infra" {
  source = "./modules/infra"
  az     = "us-east-1"
}

module "target" {
  source = "./modules/target"
}

scenario "step_vars" {
  matrix {
    distro = ["ubuntu", "rhel"]
    arch   = ["arm", "amd"]
  }

  step "infra_default" {
    module = module.infra
  }

  step "infra_west" {
    module = module.infra

    variables {
      az = "us-west-1"
    }
  }

  step "target" {
    module = module.target

    variables {
      ami = step.infra_default.amis[matrix.distro][matrix.arch]
    }
  }

  output "absolute" {
    description = "an absolute value"
    sensitive   = true
    value       = "something"
  }

  output "from_variables" {
    description = "something set as a variable"
    value       = var.project
  }

  output "module_default" {
    description = "a known value inherited through module defaults"
    value       = step.infra_default.az
  }

  output "step_known" {
    description = "a known value set at a step"
    value       = step.infra_west.az
  }

  output "step_reference_output_ref" {
    description = "a reference to a step output that is not known"
    value       = step.target.ami
  }

  output "step_reference_unknown" {
    description = "a reference to a step output that is unknown"
    value       = step.target.ips
  }
}

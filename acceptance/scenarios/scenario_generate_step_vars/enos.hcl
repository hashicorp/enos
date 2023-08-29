variable "project" {
  type = string
}

module "setupize" {
  source = "./modules/setupize"
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

  step "setup" {
    module = module.setupize
  }

  step "infra_default" {
    depends_on = ["setup"]

    module = module.infra
  }

  step "infra_west" {
    depends_on = [step.setup]

    module = module.infra

    variables {
      az = "us-west-1"
    }
  }

  step "target" {
    module     = module.target
    depends_on = concat([step.setup], [matrix.distro == "ubuntu" ? step.infra_west : step.infra_default])

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

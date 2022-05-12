module "infra" {
  source = "./modules/infra"
}

module "target" {
  source = "./modules/target"
}

scenario "step_vars" {
  matrix {
    distro = ["ubuntu", "rhel"]
    arch   = ["arm", "amd"]
  }

  step "infra" {
    module = module.infra
  }

  step "target" {
    module = module.target

    variables {
      ami = step.infra.amis[matrix.distro][matrix.arch]
    }
  }
}

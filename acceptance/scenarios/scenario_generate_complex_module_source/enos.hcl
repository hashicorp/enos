module "forward" {
  source = "./modules/forward"
}

module "backward" {
  source = "../scenario_generate_pass_0/modules/bar"
}

scenario "path" {
  matrix {
    skip = ["skip", "keep"]
  }

  step "forward" {
    skip_step = matrix.skip == "skip"
    module    = module.forward
  }

  step "backward" {
    module = module.backward
  }
}

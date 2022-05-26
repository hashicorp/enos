module "forward" {
  source = "./modules/forward"
}

module "backward" {
  source = "../scenario_generate_pass_0/modules/bar"
}

scenario "path" {
  step "forward" {
    module = module.forward
  }

  step "backward" {
    module = module.backward
  }
}

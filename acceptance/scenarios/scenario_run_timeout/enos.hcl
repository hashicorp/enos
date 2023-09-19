module "sleep" {
  source = "./modules/sleep"
}

scenario "timeout" {
  step "sleep" {
    module = module.sleep
  }
}

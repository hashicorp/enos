module "consul" {
  source = "hashicorp/consul/aws"
}

scenario "test" {
  step "backend" {
    module = module.consul
  }
}

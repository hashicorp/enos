module "has_warning" {
  source = "./modules/has_warning"
}

module "valid" {
  source = "./modules/valid"
}

scenario "warning" {
  matrix {
    mod = ["has_warning", "valid"]
  }

  step "test" {
    module = matrix.mod
  }
}

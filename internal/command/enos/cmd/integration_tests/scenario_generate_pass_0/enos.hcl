module "foo" {
  source = "./modules/foo"

  input        = "foo"
  anotherinput = ["anotherfoo"]
}

module "bar" {
  source = "./modules/bar"

  input        = "bar"
  anotherinput = "anotherbar"
}

scenario "test" {
  step "foo" {
    module = module.foo
  }

  step "bar" {
    module = module.bar
  }

  step "fooover" {
    module = module.foo

    variables {
      input = "fooover"
      anotherinput = ["fooover"]
    }
  }

  step "barover" {
    module = module.bar

   variables {
      anotherinput = "anotherbarover"
    }
  }
}

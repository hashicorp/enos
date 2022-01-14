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

terraform_cli "debug" {
  env = {
    TF_LOG_CORE     = "off"
    TF_LOG_PROVIDER = "debug"
  }
}

// TODO: add 'transport' stanza when we can
// NOTE: We can't add 'transport' stanzas until required_providers support exists

scenario "test" {
  terraform_cli = terraform_cli.debug

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

variable "defaultinput" {
  type = object({
    value = string
  })
  default = {
    value = "defaultval"
  }
  sensitive = false
}

variable "setinput" {
  type = object({
    value = string
  })
  default = {
    value = "defaultval"
  }
  sensitive = false
}

terraform_cli "debug" {
  env = {
    TF_LOG_CORE     = "off"
    TF_LOG_PROVIDER = "debug"
  }
}

module "foo" {
  source = "./modules/foo"

  input        = var.defaultinput.value
  anotherinput = ["anotherfoo"]
}

module "bar" {
  source = "./modules/bar"

  input        = "bar"
  anotherinput = "anotherbar"
}

module "fooref" {
  source = "./modules/foo"
}

module "barref" {
  source = "./modules/bar"
}

scenario "test" {
  terraform_cli = terraform_cli.debug

  matrix {
    foo = ["matrixfoo", "matrixbar"]
  }

  step "foo" {
    module = module.foo
  }

  step "bar" {
    module = module.bar
  }

  step "fooover" {
    module = module.foo

    variables {
      input        = matrix.foo
      anotherinput = ["fooover"]
    }
  }

  step "barover" {
    module = module.bar

   variables {
      anotherinput = var.setinput.value
    }
  }

  step "fooref" {
    module = module.fooref
  }

  step "barref" {
    module = module.barref

   variables {
      input        = step.fooref.input
      anotherinput = step.fooref.anotherinput
    }
  }
}

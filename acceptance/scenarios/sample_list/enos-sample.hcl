module "upgrade" {
  source = "./modules/upgrade"
}

module "replication" {
  source = "./modules/replication"
}

scenario "upgrade" {
  matrix {
    region = ["us-west", "us-east", "eu-west"]
    az     = ["a", "b", "c"]
  }

  step "upgrade" {
    module = module.upgrade

    variables {
      az = matrix.az
    }
  }
}

scenario "replication" {
  matrix {
    region = ["eu-west", "us-east", "us-west"]
    az     = ["c", "b", "a"]
  }

  step "replication" {
    module = module.replication
    variables {
      az = matrix.az
    }
  }
}

sample "minimal" {
  subset "alias" {
    scenario_filter = "vault foo:bar bar:baz"
  }
}

globals {
  replication_consul_attrs = {
    things = "others"
  }
}

sample "complex" {
  attributes = {
    aws-region        = ["us-west-1", "us-west-2"]
    continue-on-error = false
  }

  subset "upgrade_consul" {
    scenario_name = "upgrade"

    attributes = {
      continue-on-error = true
    }

    matrix {
      arch    = ["amd64", "arm64"]
      backend = ["consul"]
    }
  }

  subset "upgrade_raft" {
    scenario_name = "replication"
    attributes    = global.replication_consul_attrs

    matrix {
      arch    = ["amd64", "arm64"]
      backend = ["raft"]
    }
  }
}

module "upgrade" {
  source = "./modules/upgrade"
}

module "replication" {
  source = "./modules/replication"
}

scenario "upgrade" {
  matrix {
    arch            = ["amd64", "arm64"]
    artifact_source = ["local", "crt", "artifactory"]
    artifact_type   = ["bundle", "package"]
    backend         = ["consul", "raft"]
    consul_version  = ["1.12.3", "1.13.6", "1.14.9", "1.15.5", "1.16.1"]
    distro          = ["ubuntu", "rhel"]
    edition         = ["ce", "ent", "ent.fips1402", "ent.hsm", "ent.hsm.fips1402"]
    initial_version = ["1.8.12", "1.9.10", "1.10.11", "1.11.12", "1.12.11", "1.13.6", "1.14.2"]
    seal            = ["awskms", "shamir"]

    # Our local builder always creates bundles
    exclude {
      artifact_source = ["local"]
      artifact_type   = ["package"]
    }

    # HSM and FIPS 140-2 are only supported on amd64
    exclude {
      arch    = ["arm64"]
      edition = ["ent.fips1402", "ent.hsm", "ent.hsm.fips1402"]
    }

    # FIPS 140-2 editions began at 1.10
    exclude {
      edition         = ["ent.fips1402", "ent.hsm.fips1402"]
      initial_version = ["1.8.12", "1.9.10"]
    }
  }

  step "upgrade" {
    module = module.upgrade
  }

  variables {
    az = matrix.distro
  }
}

scenario "replication" {
  matrix {
    arch            = ["amd64", "arm64"]
    artifact_source = ["local", "crt", "artifactory"]
    artifact_type   = ["bundle", "package"]
    backend         = ["consul", "raft"]
    consul_version  = ["1.12.3", "1.13.6", "1.14.9", "1.15.5", "1.16.1"]
    distro          = ["ubuntu", "rhel"]
    edition         = ["ce", "ent", "ent.fips1402", "ent.hsm", "ent.hsm.fips1402"]
    initial_version = ["1.8.12", "1.9.10", "1.10.11", "1.11.12", "1.12.11", "1.13.6", "1.14.2"]
    seal            = ["awskms", "shamir"]

    # Our local builder always creates bundles
    exclude {
      artifact_source = ["local"]
      artifact_type   = ["package"]
    }

    # HSM and FIPS 140-2 are only supported on amd64
    exclude {
      arch    = ["arm64"]
      edition = ["ent.fips1402", "ent.hsm", "ent.hsm.fips1402"]
    }

    # FIPS 140-2 editions began at 1.10
    exclude {
      edition         = ["ent.fips1402", "ent.hsm.fips1402"]
      initial_version = ["1.8.12", "1.9.10"]
    }
  }

  step "replication" {
    module = module.replication
  }

  variables {
    az = matrix.distro
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

    matrix {
      arch    = ["amd64", "arm64"]
      backend = ["raft"]
    }
  }
}

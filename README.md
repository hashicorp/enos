# enos
Enos is a CLI tool, which is part of the larger Enos framework, that powers Software Quality as Code. Users of Enos framework compose Quality Requirement scenarios using a declarative HCL DSL and reusable Terraform modules. They perform actions with those scenarios using the Enos CLI.

## Components
Enos can be thought of as framework to power Quality as Code as a part of your software delivery lifecycle. It has several components to help authors compose, execute, automate, and debug Quality Requirements.

#### DSL
The Enos DSL can be thought of as a Quality Requirement first, multiplexed Terraform root module. It differs from Terraform in that the Terraform DSL is designed to create a single root module by combining all datasources, resources, and module calls into a single directed graph of actions which it will use to achieve a desired state. Enos takes a view of creating one-or-more scenarios, who in turn might have variants with differing behavior, that can assert a quality requirement, i.e. test some behavior.

Enos Quality Requirement Scenarios are built upon Terraform and as such you can use any Terraform provider necessary to build or configure any dependencies.

#### CLI
The Enos CLI is responsible for decoding the enos.hcl and enos.vars.hcl into individual scenarios, generating Terraform modules of each scenario, performing Terraform actions against those modules, and tracking history of actions.

#### Provider
The [Enos Provider](https://github.com/hashicorp/enos-provider) is a Terraform provider that providers Terraform resources and datasources that are useful for using Terraform as a testing framework. There are helpers for running commands locally or on remote machines, discovering artifacts or local system information, installing artifacts, or setting up clusters of HashiCorp products.

#### Github Action
The [Setup Enos Github Action](https://github.com/hashicorp/action-setup-enos) is a Github Action for installing and executing `enos` scenarios as part of Github Actions pipelines.

#### Homebrew Formula
The internal [Enos homebrew formula](https://github.com/hashicorp/homebrew-internal/tree/main/HomebrewFormula) is available to easily install releases of `enos` on your local machine. Github releases are also available for download in this repository.

## Features
### DSL
The `enos` DSL is similar to Terraform's root module syntax, but differs in a few significant ways. Rather than a single root module, an author can define reusable top-level resources to be shared between one-or-many scenarios.

Enos configurations are to be defined in `enos.hcl` or in multiple files that begin with `enos-` and end with `.hcl`, e.g. `enos-scenarios.hcl`. Variable inputs are defined in `enos.vars.hcl`.

#### Module
The `module` block maps conceptually to a Terraform module that you want to make available to scenarios. It allows you to give it a name, specify the name with a block label and has `source` and `version` attributes to specify where it is located. The `version` and `source` behave exactly as they do for [module calls in Terraform](https://www.terraform.io/language/modules/syntax). Any other attributes that are set are considered default values. Every scenario step in a module must map to a module defined in the root scope.

Example:
```hcl
module "ec2_instance" {
  source = "./modules/target"
  tags   = var.tags
}

module "test_app" {
  source = "./modules/test_app"
  tags   = var.tags
}

scenario "test" {
  step "target" {
    module = module.ec2_instance
  }

  step "test_app" {
    module = module.test_app
  }
}
```

#### Provider
The `provider` block is similar to the [provider configuration in Terraform](https://www.terraform.io/language/providers/configuration) but has alias built-in by label. The top-level provider blocks define provider configurations but scenarios are responsible for specifying which providers to use for which provider type.

Example:
```hcl
provider "aws" "east" {
  region = "us-east-1"
}

provider "enos" "ubuntu" {
  transport = {
    ssh = {
      user             = "ubuntu"
      private_key_path = abspath(joinpath(path.root, "../../support/private_key.pem"))
    }
  }
}

scenario "e2e" {
  providers = [
    provider.aws.east,
    provider.enos.ubuntu
  ]
}
```

#### Terraform CLI
Terraform is generaally configured by any combination of environment variables, CLI flags, and rc configuration files. In order to support configuration group sets, Enos has a `terraform_cli` block that allows namespaced configuration sets to be used during operations of scenarios. All configuration that is currently supported in [configuration file](https://www.terraform.io/cli/config/config-file) should be supported in the `terraform_cli` block. In addtion to those configuration options and `env` attribute is available to specify a map of key/value pairs that should be set in the environment during execution, along with a `path` attribute that specifies where the `terraform` binary to execute resides. By default Enos will resolve `terraform` from the environment. A `terraform_cli` configuration block with the name of `default` will automatically be used for scenarios that do not set the `terraform_cli` attribute.

Example:
```hcl
terraform_cli "enos_from_s3" {
  provider_installation {
    network_mirror {
      url = "https://enos-provider-current.s3.amazonaws.com/"
      include = ["hashicorp.com/qti/enos"]
    }
    direct {
      exclude = [
        "hashicorp.com/qti/enos"
      ]
    }
  }
}

scenario "test" {
  terraform_cli = terraform_cli.enos_from_s3
}
```

#### Terraform Settings
Enos also has a concept of named Terraform settings, which can be selectively applied to scenarios. The configuration is [exactly the same as in Terraform](https://www.terraform.io/language/settings), but a few non-HCL2 configuration options have changed slightly to be valid plain HCL2. For example, `experiments=[example]` in Terraform would need to be written as `experiments=["example"]` in Enos. Note that scenarios will not automatically inherit `terraform` blocks that are named `default`.

Example:
```hcl
terraform "default" {
  required_version = ">= 1.0.0"

  required_providers {
    enos = {
      version = ">= 0.1.13"
      source  = "hashicorp.com/qti/enos"
    }

    aws = {
      source = "hashicorp/aws"
    }
  }
}

scenario "e2e" {
  terraform     = terraform.default
}
```

#### Variable
Variables in Enos have nearly the same [behavior as those in Terraform](https://www.terraform.io/language/values/variables), the only exception is that validations are not currently implemented. Variable inputs are defined in `enos.hcl` and values that are passed in are defined in `enos.vars.hcl`.

Example:
```hcl
variable "tags" {
  description = "Tags to add to AWS resources"
  type        = map(string)
  default     = null
}

module "ec2_instance" {
  source = "./modules/target"
  tags   = var.tags
}
```

#### Scenario
The scenario can be considered one of the possible root terraform modules that Enos might execute. The `scenario` is comprised of one-or-more `step` blocks which perform some bit of policy. Each step block must have a `module` attribute that maps to the name of a defined `module` or to the `module` object.

Example:
```hcl
module "ec2_instance" {
  source = "./modules/target"
}

module "test_app" {
  source = "./modules/test_app"
}

scenario "test" {
  step "target" {
    module = module.ec2_instance
  }

  step "test_app" {
    module = module.test_app
  }
}
```

Scenarios can also pass information from one step to the next, as one might do
in Terraform. A step variable must reference a known value or an output from 
a `step` module. Step variables must reference a variable in the `step`'s module.

Example:
```hcl
module "ec2_instance" {
  source = "./modules/target"
}

module "test_app" {
  source = "./modules/test_app"
}

scenario "test" {
  step "target" {
    module = module.ec2_instance
  }

  step "test_app" {
    module = module.test_app

    variables {
      target_addr = step.target.addr
    }
  }
}
```

For complex scenarios, you can use a `matrix` to define variants. You can also
dynamically compose which module to use for a `step`. You can also build complex
maps using the `local` block in a scenario to make logical decisions. The following
example would generate 145 scenarios that you could run depending on the desired
variant combinations you want to test.

Example:
```hcl
variable "version" {
  type = string
}

variable "upgrade_initial_version" {
  type = string
  default = "1.8.2"
}

variable "ent_sha" {
  type = string
}

variable "oss_sha" {
  type = string
}

variable "private_key_path" {
  type = string
}

module "backend_raft" {
  source = "./modules/backend_raft"
}

module "backend_consul" {
  source = "./modules/backend_consul"
}

module "vault" {
  source = "./modules/vault_cluster"

  version = var.version
  sha     = var.version
}

module "test_upgrade" {
  source = "./modules/test_upgrade"
}

module "test_fresh_install" {
  source = "./modules/test_fresh_install"
}

module "test_license" {
  source = "./modules/test_license"
}

provider "aws" "east" {
  region = "us-east-1"
}

provider "enos" "ubuntu" {
  transport = {
    ssh = {
      user             = "ubuntu"
      private_key_path = abspath(var.private_key_path)
    }
  }
}

provider "enos" "rhel" {
  transport = {
    ssh = {
      user             = "ec2-user"
      private_key_path = abspath(var.private_key_path)
    }
  }
}

scenario "matrix" {
  matrix {
    // Matrix of variants. A Cartesian product of the matrix will be produced and
    // a scenario for product will be available.
    backend       = ["raft", "consul"]
    arch          = ["arm64", "amd64"]
    edition       = ["ent", "ent.hsm", "oss"]
    artifact_type = ["bundle", "package"]
    test          = ["upgrade", "fresh_install", "license"]
    distro        = ["ubuntu", "rhel"]

    // Manual addtions to the matrix.
    include {
      backend       = ["raft"]
      arch          = ["amd64"]
      edition       = ["fips1402"]
      artifact_type = ["bundle"]
      test          = ["fresh_install"]
      distro        = ["ubuntu"]
    }

    // Exclude any variant vector that matches. Any vector that has matching
    // elements will be excluded. Increase the elements to add specificity.
    exclude {
      artifact_type = ["package"]
      edition       = ["fips1402"]
    }
  }

  locals {
    // Logical decisions via locals are a frequent pattern.
    enos_provider = {
      rhel   = provider.enos.rhel
      ubuntu = provider.enos.ubuntu
    }

    vault_version = {
      upgrade       = var.upgrade_initial_version
      fresh_install = var.version
      license       = var.version
    }

    sha = {
      oss       = var.oss_sha
      ent       = var.ent_sha
      "ent.hsm" = var.ent_sha
      fips1402  = var.ent_sha
    }

    skip_test = {
      license       = semverconstraint(var.version, ">= 1.9.0-dev")
      fresh_install = false
      upgrade       = false
    }
  }

  providers = [
    provider.aws.east,
    local.enos_provider[matrix.distro],
  ]

  step "backend" {
    // You can also reference attributes in "scenario" blocks via a string reference
    // to make dynamic imports of modules easy. Just be sure that variable inputs
    // and outputs are common.
    module = "backend_${matrix.backend}"
  }

  step "vault" {
    module = module.vault

    variables {
      arch          = matrix.arch
      edition       = matrix.edition
      artifact_type = matrix.artifact_type
      version       = local.vault_version[matrix.test]
      sha           = local.sha[matrix.edition]
    }
  }

  step "test" {
    module = "test_${matrix.test}"

    variables {
      skip = local.skip_test[matrix.test]
    }
  }
}
```

### CLI
The `enos` CLI is how you'll decode, execute, and clean up any resources that were created for your scenario.

By default, all `scenario` sub-commands work on all scenarios that it will decode. You can filter
to gain specificity using inclusive or exlusive filters. Remember to quote your exclusive filters
so that your shell doesn't try to expand it.

Please note that the CLI is actively being developed and as such the output
is likely to change significantly.

Example:
```
enos scenario run '!artifact_type:bundle' backend:raft edition:fips1402'
```

#### List
The `list` sub-command lists all decoded scenarios, along with any variant spefic information.

Example:
```
$ enos scenario list
SCENARIO

matrix [arch:amd64 artifact_type:bundle backend:consul distro:rhel edition:ent test:fresh_install]
matrix [arch:amd64 artifact_type:bundle backend:consul distro:rhel edition:ent test:license]
matrix [arch:amd64 artifact_type:bundle backend:consul distro:rhel edition:ent test:upgrade]
...
```

By default, all `scenario` sub-commands work on all scenarios that it will decode. You can filter
to gain specificity using inclusive or exlusive filters. Remember to quote your exclusive filters
so that your shell doesn't try to expand it.

Example:
```
$ enos scenario list '!artifact_type:bundle' backend:consul '!edition:ent'
SCENARIO

matrix [arch:amd64 artifact_type:package backend:consul distro:rhel edition:ent.hsm test:fresh_install]
matrix [arch:amd64 artifact_type:package backend:consul distro:rhel edition:ent.hsm test:license]
matrix [arch:amd64 artifact_type:package backend:consul distro:rhel edition:ent.hsm test:upgrade]
...
```

#### Generate
The `generate` sub-command generates the Terraform root modules any any associated
Terraform CLI configuration. All other sub-commands that need a Terraform root
module and configuration will generate this if necessary, but this command exists
primary for troubleshooting.

Example:
```
$ enos scenario generate --chdir acceptance/scenarios/scenario_e2e_aws/
creating directory /.../.../.enos/6d8749c4fe00b757c1bc1c376a99f31d1bd38b422d44f5b6aa2d6a6c5e975cba
writing to /.../.../.enos/6d8749c4fe00b757c1bc1c376a99f31d1bd38b422d44f5b6aa2d6a6c5e975cba/terraform.rc
writing to /.../.../.enos/6d8749c4fe00b757c1bc1c376a99f31d1bd38b422d44f5b6aa2d6a6c5e975cba/scenario.tf
```

#### Validate
The `validate` sub-command generates the Terraform root modules any any associated
Terraform CLI configuration and then passed the results to Terraform for module
validation. This will attempt to download and required modules and providers and
plan it.

Example:
```
$ enos scenario validate --chdir acceptance/scenarios/scenario_e2e_aws/
writing to /.../.../.enos/6d8749c4fe00b757c1bc1c376a99f31d1bd38b422d44f5b6aa2d6a6c5e975cba/terraform.rc
writing to /.../.../.enos/6d8749c4fe00b757c1bc1c376a99f31d1bd38b422d44f5b6aa2d6a6c5e975cba/scenario.tf
{
  "terraform_version": "1.1.7",
  "platform": "darwin_amd64",
  "provider_selections": {
    "hashicorp.com/qti/enos": "0.1.21",
    "registry.terraform.io/hashicorp/aws": "4.9.0"
  },
  "terraform_outdated": true
}
Upgrading modules...
...
```

#### Launch
The `launch` sub-command applies the Terraform plan. You would usually do this
after you've validated a scenario.

Example:
```
$ enos scenario launch
...
```

#### Destroy
The `destroy` sub-command destroys the Terraform plan. You would usually do this
after you've launched a scenario.

Example:
```
$ enos scenario destroy
...
```

#### Run
The `run` sub-command generates, validates, launches a scenario. In the event
that it is succcessful it will also destroy the resources afterwards.

Example:
```
$ enos scenario run
...
```

#### Exec
The `exec` sub-command allows you to run any Terraform sub-command within the
context of a Scenario. This is useful for debugging.

Example:
```
$ enos scenario exec test arch:arm64 backend:consul distro:rhel --cmd "state show target.addr"
...
```

## Contrubuting

All contributions are welcome! Please report any bugs or feature requests using
the standard procedures on this repository. Feel free to drop into #team-quality
or #talk-quality if you have any questions or questions.

***

## General tips

### Save temporary Doormat credentials to local AWS credentials file

- If you don’t already have active Doormat credentials, run

    `doormat --refresh`
- You will need your AWS account name, account ID, and access level. You can do this by checking them at https://doormat.hashicorp.services/, or by running `doormat aws --list` to see your eligible roles on the accounts you have access to.

- Now, run the following, replacing `<account_number>`, `<account_name>`, and `<access_level>` with yours:
 
    `doormat aws --role arn:aws:iam::<account_number>:role/<account_name>-<access_level> -m`

    (`-m` = manage (for AWS configs))

- To check if it was successful:

    `cd ~/.aws`
    
    `cat credentials`

- It should show your updated `aws_access_key_id`, `aws_secret_access_key`, and `aws_session_token`.

***

## Release

### Require Changelog Label Workflow

The `Require Changelog Label` workflow verifies whether a PR has at least one of the four designated `changelog/` labels applied to it. These labels are used to automatically create release notes.

### Validate Workflow

The `validate` workflow is a re-usable GitHub workflow that is called by `PR_build` workflow, when a PR is created against the `main` branch and is also called by the `build` workflow after a PR is merged to `main` branch. This workflow runs Lint, Unit and Acceptance tests. The Acceptance tests are run on `linux/amd64` artifacts created by the caller workflows (`PR_build` or `build`).

### PR Build Workflow

The `PR_build` workflow is run when a PR is created against the `main` branch.  This workflow creates and uploads `linux/amd64` artifact. It then calls the `validate` workflow which downloads this artifact and runs Lint, Unit, and Acceptances tests on it. This workflow also verifies that at least one of the four designated `changelog/` labels has been applied to the PR, in order to facilitate automatic generation of release notes (see **Enos Release** section below for more details).

### Build Workflow
The `build` workflow is run after PR merge to `main` and only if `version.go` is updated. The `build` workflow creates build artifacts for `Linux` and `Darwin` on `amd64` and `arm64` architectures. It also creates `rpm`, `deb` packages, and `Docker` images. All created artifacts are uploaded to GH workflow. It then calls the `validate` workflow which downloads the `linux/amd64` artifact and runs Lint, Unit, and Acceptance tests on it.

### CRT Workflow
The `ci.hcl` is responsible for configuring the CRT workflow orchestration app. The orchestration app will read this configuration and trigger common workflows from the CRT repo. These workflows are responsible for uploading the release artifacts to Artifactory, notarizing macOS binaries, signing binaries and packages with the appropriate HashiCorp certificates, security scanning, and binary verification. The `build` workflow is a required prerequisite as it is responsible for building the release artifacts and generating the required metadata.

### Enos Release
Enos is made available as a Github release, via the `create_release` workflow. This workflow is manually triggered via the Github UI. It has three required inputs: the git SHA of the artifacts in Artifactory to be used in this release; the version number to be used in this release; and the Artifactory channel from which to download the release assets (this defaults to `stable`). It also has one optional input: a `pre-release` checkbox. If `pre-release` is selected, the release will be marked on the Github UI with a `pre-release` label and its tag will take the form `v<version>-pre+<first 5 characters of SHA>` i.e. `v0.0.1-pre+bd25a`. Regular release tags will take the form `v<version>` i.e. `v.0.0.1`.

To create the release, the workflow downloads the release assets from Artifactory, which have been previously notarized, signed, scanned, and verified by CRT workflows. Then, it creates a Github release and uploads these artifacts as the release assets. It automatically generates release notes, which are organized into the following categories via the four designated `changelog/` labels. At least one of these labels must be applied to each PR.
- **New Features 🎉** category includes PRs with the label `changelog/feat`
- **Bug Fixes 🐛** category includes PRs with the label `changelog/bug`
- **Other Changes 😎** category includes PRs with the label `changelog/other`
- PRs with the label `changelog/none` will be excluded from release notes.

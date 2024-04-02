<!-- vim: ts=2 sw=2 -->
# enos

Enos is a tool for powering Software Quality as Code by writing Terraform-based quality requirement scenarios using a composable, modular, and declarative language.

![Enos_2023-transparent (1)](https://github.com/hashicorp/enos/assets/65058/4551b240-61d2-49f0-b037-90bc54e88b30)

## What problem does Enos solve?

You can use enos to _define and verify quality requirements of complex distributed software_.

Before we go deeper we'll define some terms:
  - *Software quality*
    The degree to which a software product satisfies stated and implied needs when used under specified conditions.
  - *Functional requirements*
    The software’s primary domain function that performs one or many well defined tasks.
  - *Nonfunctional requirements*
    The software’s supporting systems which are not primary domain functions but are required to use the software, e.g. security, compliance, high availability, disaster recovery, durability, compatibility, etc.
  - *Service-level objective*
    An agreed upon means of measuring the performance of a service.
  - *Quality characteristic*
    An attribute of software that we wish to measure for quality purposes, e.g. reliability.
  - *Product factors*
    An observable property or function of a software product.
  - *Quality requirement*
    A stated desire to have a product factor fulfill a quality characteristic.
  - *Quality requirement scenario*
    An executable scenario that verifies one-or-more quality requirements of software.

To understand why a tool like Enos exists we need to understand the differences that software delivery models make on both our quality requirements and the tools and strategies that we can utilize to ship quality software.

For example, lets consider a hyphothetical software service application. What tools and strategies can we use to measure and verify the quality of our service? Say our application has lots of smaller unit tests that we execute regularly to verify some functional requirements before we deploy it. We run these often enough to give ourselves a sense of confidence in the correctness a unit level, but we don't verify our integrated product before changes in the source or deployment. Many of our services non-functional supporting systems like backups, high availability, and disaster recovery are provided by a platform team. Some of those requirements are fulfilled by external Cloud provider services and their SLAs. Our application is only deployed by us so we rarely have to worry about long term compatibility or data durability because we're only running a singular version of the application at one given time and it's regularly being updated to the latest version. We measure the quality of our service by defining various service level objectives and use instrumenation and observability tools to measure our SLO's and identify issues _after deployment_. We measure our velocity, which includes our time to recover from deployed bug. Our pipline is fast enough that we rely on being able to quickly patch issues and ship them.

I want to emphasize that running a service is by no means trivial and our example omits plenty of challenges, but the tools and services we have available to measure quality (SLO's, velocity, observability) and move quickly with relative safety (automation, test verification, external tools and services with their SLAs) exist. Because our deployment model is limited and many of our non-functional requirements have been outsourced, our primary quality focus is limited to functional suitability and velocity. With these constraints a tool like Enos _could_ be useful for you but it was not designed for these constraints or requirements.

Now lets consider a complex on-premises software product that we ship via binaries and packages. Lets say our product could be best described as a platform and that it services many different kinds of workloads, all with different access patterns and resource considerations. It has a vast array of deployment options and integrations. It has close to no telemetry and is often deployed in air-gapped environments, so even if the telemetry did exist it will never reach you. The deployment cycles for this can be up to a year in highly regulated environments, so its up the upmost importance that all functional, non-functional, and lifecycle actions behave as expected at all times, regardless of the many different ways the software might be deployed or used.

This is by no means an apples to apples comparison with our SaaS example, but that's the point. Our software delivery method has changed by our quality requirements and our methods of quality verification. Instead of primarily focusing on the functional requirements of the system and velocity, we now have a larger responsibility of verification before the product is released. And on top of it, we also have so many other deployments variables like edition, version, platform, architecture, runtime, storage, network, cluster size, CPUs, memory, integrations, auditing, logging, HA, DR, etc, that we have to consider and ought to verify. In effect, we are now responsible for all functional, non-functional, and lifecycle quality, and since we no longer control most of the environmental variables we have to figure out how to verify our software in all sorts of environmental combinations.

How do we ensure that our software behaves as expected under so many unique circumstances? What tools do we have for this? 

  - *Unit tests*
    Unit tests are really good at what they're supposed to do: verify the correctness of a routine. They're fast, easy, and relatively cheap, but they're isolated. Our application doesn't run in isolation, it runs integrated with everything else. We cannot rely on unit tests for reliability, efficiency, or non-functional quality requirements.
  - *End-to-end tests, Acceptance Tests, Integration tests*
    All of these are useful because they integrate more than one routine to give us more confidence in our application, we're actually testing our software in manner that more closely resembles reality. These tests might be a bit slow relative to our unit tests, but we'll still do a lot of these because they easy for building faster feedback loops during development. They're still not a good proxy for what we're going to ship because we're testing a binary on our local machine, or perhaps we're deployming a test build into containers. But a real application like we're building often is not sharing CPUs, doesn't use in-memory storage, and isn't relying on in-kernel networking. Not to mention the various load profiles, lifecycle changes, and external integrations. We can't rely on these for any non-functional requirements.
  - *Interative Testing*
    Manual testing is wonderful but it's really slow, expensive, prone to errors, and not repetable. We should include some of this but can't rely on it for most quality characteristics.
  - *Load Testing, Performance Testing, Stress Testing*
    These strategies are great testing our non-functional requirements. It's really useful but also quite slow and often expensive. We also have a problem of our deployment matrix. How do get a representative load test when there are so many different ways to deploy or software and so many different ways to use it?
  - *Black box testing*
    This is great for testing actual artifacts that we intend to ship, but we also have no good ways to testing all of our artifacts in representative ways.
  - *Observability*
    I wish, but we don't get telemetry. Most of these deployments are air-gapped and even if they were not, people might not want to spend on all that egress.
  - *Simulation testing*
    TBD. This might be a valid strategy at some point, but we don't get to control the environment.
  - *Proofs?*
    :weary: Maybe for some small routines. We simply don't have the budget or expertise to formally verify all of our software.

This is not an exhaustive list of strategies but it does cover the most common answers. All of theses surely have their place in some parts of our quality workflow, but we've identified several gaps:

#### Constraints
* We have lots of different unique artifacts and they all need to have their quality characteristics verified.
* We need to verify with myriad deployment options like architecture, runtime, operating system.
* We need to verify myriad configuration options.
* We need to verify myriad external integrations.
* We need to verify our integrated systems in real-world scenarios.
* We need to verify our non-functional requirements with various different deployments, runtimes, and workloads.
* We need to verify the compatibility and durability when upgrading from various prior versions.
* We need to verify our our migration behaviors.
* We have a limited budget so we need choose the best cost/benefit solution.
* We have limited time so we have to choose solutions that are relatively fast.
* We woud like to automate whatever possible to improve our velocity and get sustained rewards for our time investment.

#### Problems
We needed a general purpose tool that allows us to author and execute fully end-to-end scenarios. It needed to support large matrices of options, handle all infrastructure set up and tear down, and allow sampling over scenario matrix products that could be in the millions.

That's why we built Enos. It's designed to solve those problems specifically for HashiCorp products, but there are general purpose resources that might be valuable for your workflow.

## What is Enos?

Enos is a Terraform-based framework. In practice, an Enos scenario consists of one or more HCL configuration files that make use of Terraform providers and modules. These files are read and executed by the Enos CLI to spin up the specified resources and perform the specified verifications on them.

The Enos framework is made up of several components:

* **Terraform** is the engine of Enos. All steps of the quality requirement live within a Terraform root modules graph of resouces.
* The **Domain Specific Language (DSL)** allows us to describe the scenario we want to run, including resources we want to spin up and actions or tests we want to perform upon them. Its syntax is very close to Terraform, with some differences that allow us to abstract away some of the complexities that would otherwise exist if we were to try to enable a matrix of scenarios using Terraform alone.
* The **Command Line Interface (CLI)** allows us to execute actions within the context of the scenarios defined by the DSL. It provides a scenario based user interface, test scenario execution isolation, scenario sampling, dynamic module selection during generation, and many other features.
* The **Enos Terraform provider** gives us access to Terraform resources and data sources that are useful for common Enos tasks like: running commands locally or on remote machines, discovering artifacts or local system information, downloading and installing artifacts, or setting up clusters of HashiCorp products. Cloud-specific Terraform providers like the AWS provider allow us to interact with resources supported by that platform. You can mix and match any Terraform resources to build a scenario.
* **Terraform modules** allow us to group Terraform resources together. Scenarios are comprised of steps that implemented by such modules.
* The **`action-setup-enos` Github Action** allows for easy installation of Enos in Github Actions workflows.

## Should I use Enos?

Are you developer working on HashiCorp products that are shipped as binaries or packages? If so, then yes, the product you're working on might already use enos.

Are you working on something else? If so, then probably not but it's complicated. If you're into complication feel free to read on.

The `enos` CLI itself is completely test target agnostic and can be used to execute any Terraform module(s). Quite a bit of the functionality of Enos scenarios is actually in the enos Terraform provider and its resources. The enos provider does provide some general purpose resources that you could use for non-HashiCorp software, but the vast majority of them have been built for specific HashiCorp needs and products.

We want to be clear that *Enos and the Enos provider exist solely for HashiCorp products* and there are *no guarantees* that features for anything else will ever be priorizitied or built. You should expect *no support whatsoever if you choose to use the tools*, as they are intended for *internal use only* and should *never be used in production Terraform*.

## How can I get started with Enos?

If you're a HashiCorp developer, you can follow the Enos tutorials in the Engineering Handbook.

If you're not, we don't currently have any examples or tutorials published yet, but you can install [Binaries](https://github.com/hashicorp/enos/releases/) and read about the various components below. There is also plenty of advanced Enos prior art in the Vault or Boundary repositories.

## How is this different from `terraform test`?

`terraform test` is a wonderful tool but has little overlap with Enos. Enos was started years before `terraform test` became a real thing and the responsibility of each is quite different. `terraform test` is a great way for testing your _Terraform Module_ while Enos is a great way to test _your application_ using Terraform as an engine. Enos _dynamically_ generates Terraform HCL and is intended for short to medium term test verification. `terrafrom test` is intended to test your static Terraform HCL.

## Components
Enos can be thought of as framework to power Quality as Code as a part of your software delivery lifecycle. It has several components to help authors compose, execute, automate, and debug Quality Requirements.

#### DSL
The Enos DSL can be thought of as a Quality Requirement first, multiplexed Terraform root module. It differs from Terraform in that the Terraform DSL is designed to create a single root module by combining all datasources, resources, and module calls into a single directed graph of actions which it will use to achieve a desired state. Enos takes a view of creating one-or-more scenarios, who in turn might have variants with differing behavior, that can assert a quality requirement, i.e. test some behavior.

Enos Quality Requirement Scenarios are built upon Terraform and as such you can use any Terraform provider necessary to build or configure any dependencies.

#### CLI
The Enos CLI is primarily responsible for decoding the enos.hcl and enos.vars.hcl into individual scenarios, generating Terraform modules of each scenario, performing Terraform actions against those modules, and tracking history of actions. It is also used for advanced use cases like sampling, which is pseudo random execution of compatible scenarios for various test artifacts.

#### Provider
The [Enos Provider](https://github.com/hashicorp-forge/terraform-provider-enos) is a Terraform provider that provider Terraform resources and datasources that are useful for using Terraform as a testing framework. There are helpers for running commands locally or on remote machines, discovering artifacts or local system information, installing artifacts, or setting up clusters of HashiCorp products.

#### Github Action
The [Setup Enos Github Action](https://github.com/hashicorp/action-setup-enos) is a Github Action for installing and executing `enos` scenarios as part of Github Actions pipelines.

#### Homebrew Formula
The hashicorp/internal homebrew tap allows you to install Enos on macOS. Binaries for various platforms are also published at Github Releases.

## Features
### DSL
The `enos` DSL is similar to Terraform's root module syntax, but differs in a few significant ways. Rather than a single root module, an author can define reusable top-level resources to be shared between one-or-many scenarios.

Enos configurations are to be defined in `enos.hcl` or in multiple files that begin with `enos-` and end with `.hcl`, e.g. `enos-scenarios.hcl`. Variable inputs are defined in `enos.vars.hcl`.

#### Module
The `module` block maps conceptually to a Terraform module that you want to make available to scenarios. It allows you to give it a name, specify the name with a block label and has `source` and `version` attributes to specify where it is located. The `version` and `source` behave exactly as they do for [module calls in Terraform](https://www.terraform.io/language/modules/syntax). Any other attributes that are set are considered default values. Every scenario step in a module must map to a module defined in the root scope.

Example:
```hcl
module "ec2_instance" {
  source = "./modules/ec2_instance"
  tags   = var.tags
}

module "load_balancer" {
  source = "./modules/envoy"
  tags   = var.tags
}

module "deploy" {
  source = "./modules/deploy_app"
  tags   = var.tags
}

module "e2e_tests" {
  source = "./modules/test_app"
  tags   = var.tags
}

scenario "test" {
  step "create_db_instance" {
    module = module.ec2_instance
  }

  step "create_app_instances" {
    module = module.ec2_instance

    variables {
      instances = 3
    }
  }

  step "create_proxy_instance" {
    module = module.ec2_instance
  }

  step "deploy_app" {
    module = module.deploy

    variables {
      app_addrs = step.create_app_instances.addrs
      db_addr = step.create_db_instance.addrs[0]
      proxy_addrs = step.create_proxy_instance.addrs
    }
  }

  step "e2e_tests" {
    depends_on = [
      step.deploy_app
    ]

    module = module.test_app

    variables = {
      addr = step.create_proxy_instance.addrs[0]
    }
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
Terraform is generaally configured by any combination of environment variables, CLI flags, and rc configuration files. In order to support configuration group sets, Enos has a `terraform_cli` block that allows namespaced configuration sets to be used during operations of scenarios. All configuration that is currently supported in [configuration file](https://www.terraform.io/cli/config/config-file) should be supported in the `terraform_cli` block. In addition to those configuration options and `env` attribute is available to specify a map of key/value pairs that should be set in the environment during execution, along with a `path` attribute that specifies where the `terraform` binary to execute resides. By default Enos will resolve `terraform` from the environment. A `terraform_cli` configuration block with the name of `default` will automatically be used for scenarios that do not set the `terraform_cli` attribute.

Example:
```hcl
terraform_cli "with_private_modules" {
  credentials "app.terraform.io" {
    token = var.tfc_api_token // Credentials to install private modules from TFC
  }
}

scenario "test" {
  terraform_cli = terraform_cli.with_private_modules
}
```

#### Terraform Settings
Enos also has a concept of named Terraform settings, which can be selectively applied to scenarios. The configuration is [exactly the same as in Terraform](https://www.terraform.io/language/settings), but a few non-HCL2 configuration options have changed slightly to be valid plain HCL2. For example, `experiments=[example]` in Terraform would need to be written as `experiments=["example"]` in Enos. Note that scenarios will not automatically inherit `terraform` blocks that are named `default`.

Example:
```hcl
terraform "default" {
  required_version = ">= 1.6.0"

  required_providers {
    enos = {
      version = ">= 0.0.1"
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

#### Globals
Globals in Enos are similar to `locals` in a `scenario` except they are global to all scenarios. Globals are evaluated after variables and must be known values at decode time.

Example:
```hcl
globals {
  tags = merge({
    "Project Name" : var.project_name
    "Environment" : "ci"
  }, var.tags)
}

module "test_my_app" {
  source = "./modules/test_my_app"

  tags = global.tags
}
```

#### Quality
Quality blocks are a way to define quality characteristics that you intend to validate with your scenario. When a step in your scenario verifies a quality requirement you can assign a quality to that steps `verifies` attribute to make the association. This allow us to track all the qualities that are validated by a scenario step. The full outline of this can be seen with the `enos scenario outline` command.

Example:
```hcl
quality "can_create_post" {
  description = "We can use the the /api/post API to write data"
}

quality "post_data_is_valid_after_upgrade" {
  description = <<-EOF
    After we upgrade the application we verify that all previous state is valid and durable
EOF
}

quality "has_correct_version_after_upgrade" {
  description = <<EOF
After we upgrade the application has the correct version
EOF
}

scenario "post" {
  // ...
  step "create_post" {
    // ...
    verifies = quality.can_create_post
  }

  step "upgrade" {
    // ...
    verifies = [
      quality.has_correct_version_after_upgrade,
      quality.post_data_is_valid_after_upgrade,
    ]
  }
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

module "test_kv_data" {
  source = "./modules/test_kv_data"
}

quality "data_is_durable_after_upgrade_migration" {
  description = "The application state is valid after the upgrade migrates it"
}

quality "can_create_kv_data" {
  description = "We can successfully create kv data"
}

quality "can_create_kv_data" {
  description = "We can successfully create kv data"
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
    test          = ["upgrade", "fresh_install", "kv_data"]
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

    verifies = [
      quality.can_create_kv_data,
      quality.data_is_durable_after_upgrade_migration,
    ]

    variables {
      skip = local.skip_test[matrix.test]
    }
  }
}
```

#### Sample
Enos scenarios support multi-variant matrices which commonly include parameters like architecture, Linux distro, storage backend, expected version, expected edition, and many more configurations. These matrices allow us to test across every possible combination of these variants, which is part of what makes Enos such a powerful tool for testing.

Of course, as our matrices grow, so does the total number of possible combinations — exponentially. It's not uncommon for a scenario to reach hundreds of thousands of possible variant combinations. Samples help us deal with some of the challenges that such large matrices introduce:

- Not all scenarios and/or variant combinations are supported for a given test artifact. How do we filter our scenarios on a per-artifact basis?
- Running all scenario and variant combinations is prohibitively expensive at any given point in a CI pipeline. Is there a way to get an acceptance sampling strategy by algorithmically selecting scenario variants at each multiple stages of the pipeline?
- How can we automate our selection and execution of scenarios in the pipeline?

Samples allow us to handle all of those challenges by defining named sample groups. Within hese sample groups you to define subsets with different scenario filters, matrices, and attributes that describe the total allowed sample field, which can be tailored anywhere from all scenarios and variant combinations to a single scenario variant.

The Enos CLI is then able to interact with the Enos server to "observe" a given sample, that is, to choose scenario specimens that we can test. All you need to do is provide the sample boundaries, i.e. minimum number of elements, maximum number of elements, or a percentage of total elements in the frame, and then Enos handles 
shaping the sample frame and selecting which scenario variants to test using its sampling algorithm.

To ensure that we get coverage over all scenarios, Enos uses its own purposive stratified sampling algorithm. Depending on our sample size limitations, it favors breadth across all samples before dividing the subsets by size and sampling based on overall proportions.

Samples also support injecting additional metadata into sample observations and subsets, which is then distributed to each sample element during observation. This allows us to dynamically configure the Enos variables for a sample and pass any other additional data through to our execution environment.

When taking an observation, the Enos CLI supports human or machine readable output. The machine readable output can be used to generate a Github Actions matrix to execute scenarios on a per-workflow basis.

Example:
```hcl
module "upgrade" {
  source = "./modules/upgrade"
}

scenario "upgrade" {
  matrix {
    backend = ["raft", "consul"]
    seal    = ["shamir", "awskms", "cloudhsm"]
  }

  step "upgrade" {
    module = module.upgrade
  }

  variables {
    backend = matrix.raft
    seal    = matrix.seal
  }
}

sample "simple" {
  subset "raft_cloudhsm" {
    scenario_filter = "upgrade backend:raft seal:cloudhsm"
  }
}

globals {
  upgrade_attrs = {
    initial_version = "1.13.2"
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
      backend = ["consul"]
    }
  }

  subset "upgrade_raft" {
    scenario_name = "replication"
    attributes    = global.upgrade_attrs

    matrix {
      arch    = ["amd64", "arm64"]
      backend = ["raft"]
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

#### Scenario List
The `scenario list` sub-command lists all decoded scenarios, along with any variant spefic information.

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

#### Scenario Generate
The `scenario generate` sub-command generates the Terraform root modules any any associated
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

#### Scenario Validate
The `scenario validate` sub-command generates the Terraform root modules any any associated
Terraform CLI configuration and then passes the results to Terraform for module
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

#### Scenario Launch
The `scenario launch` sub-command applies the Terraform plan. You would usually do this
after you've validated a scenario.

Example:
```
$ enos scenario launch
...
```

#### Scenario Destroy
The `scenario destroy` sub-command destroys the Terraform plan. You would usually do this
after you've launched a scenario.

Example:
```
$ enos scenario destroy
...
```

#### Scenario Run
The `scenario run` sub-command generates, validates, launches a scenario. In the event
that it is succcessful it will also destroy the resources afterwards.

Example:
```
$ enos scenario run
...
```

#### Scenario Exec
The `scenario exec` sub-command allows you to run any Terraform sub-command within the
context of a Scenario. This is useful for debugging as you can inspect the state for any resource
that is created during launch.

Example:
```
$ enos scenario exec test arch:arm64 backend:consul distro:rhel --cmd "state show target.addr"
...
```

#### Scenario Sample List
The `scenario sample` sub-command allows you to list which samples are available in your Enos
directory.

Example:
```
$ enos scenario sample list
...
```

#### Scenario Sample Observe
The `scenario sample observe` sub-command allows you to take a sample "observation". That is, decode
the sample blocks and scenarios to define the total sample field, take your given boundaries like
maximum and minimum scenario elements, and then uses the sampling algorithm to select specimens for
testing.

Example:
```
$ enos scenario sample observe <sample-name> --min 1 --max 5 --format json
...
```

#### Scenario Outline
The `scenario outline` sub-command allows you to generate outlines of the scenarios and quality
characteristics that you have defined in your Enos directory. The outline provides a way to quickly
get up to speed with what a scenario does and which quality characteristics it verifies.

You can you generate both a text, JSON, HTML output of the outline.

Example:
```
$ enos scenario outline <scenario-name> --format html > outline.html
$ open outline.html
```

## Contrubuting

Feel free to contribute if you wish. You'll need to sign the CLA and adhere to the [Code of Conduct](https://www.hashicorp.com/community-guidelines).

## Release

### Require Changelog Label Workflow

The `Require Changelog Label` workflow verifies whether a PR has at least one of the four designated `changelog/` labels applied to it. These labels are used to automatically create release notes.

### Validate Workflow

The `validate` workflow is a reusable GitHub workflow that is called by `PR_build` workflow, when a PR is created against the `main` branch and is also called by the `build` workflow after a PR is merged to `main` branch. This workflow runs Lint, Unit and Acceptance tests. The Acceptance tests are run on `linux/amd64` artifacts created by the caller workflows (`PR_build` or `build`).

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

After the release workflow completes, it automatically triggers another workflow. This workflow creates an updated version of the Enos Homebrew formula file and opens a PR for it in HashiCorp's internal Homebrew tap, `hashicorp/homebrew-internal`.

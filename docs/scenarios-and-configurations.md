# Enos scenarios and configurations

An Enos scenario describes a series of actions to be performed for testing or reproduction purposes. These actions commonly include things like:

- Spinning up AWS resources
- Setting configurations
- Installing software/packages on remote instances
- Executing a series of steps to test functionality, behavior, or integrations of software
- Tearing down clusters when testing is complete

We use these scenarios:

- In CI (to continuously test our software along the development and release lifecycle)
- In local development (to perform initial testing of our features or fixes before we even push to Github)
- For customer support purposes (to more easily reproduce a specific customer environment, software installation, or bug)

In the following sections, we'll review the various parts of an Enos scenario, as well as the supporting modules and configurations.

>**Note:** Enos is Terraform-based. If you are not yet familiar with basic Terraform concepts like [providers](https://developer.hashicorp.com/terraform/language/providers), [resources](https://developer.hashicorp.com/terraform/language/resources), and [modules](https://developer.hashicorp.com/terraform/language/modules), we recommend that you review the Terraform docs.

## Scenario

### Description

One of the first attributes within a scenario that you might see is the `description`. The description gives an overview of what the scenario does and any specific details required for running it, like input variables. It's a useful reference if you want to quickly understand the general purpose of a particular scenario.

### Matrix

Next, you'll likely see the `matrix`. The matrix is what makes Enos a powerful testing tool. It consists of several variants, each with a defined set of values. Enos can run the scenario using any valid combination of these variants and values.

"Valid combinations" include the [Cartesian product](https://en.wikipedia.org/wiki/Cartesian_product) of all the matrix variants and values, minus any exclusions defined in `exclude` blocks. `exclude` blocks are used to exclude any combinations of variants that are not supported. For example, in Vault, the PKCS11 seal type can only be used with HSM editions of Vault; therefore, we [exclude](https://github.com/hashicorp/vault/blob/fdc871370d632cb7b144339ed20e907b88f4533b/enos/enos-scenario-agent.hcl#L40) the combination of `pkcs11` seal type and any non-HSM Vault edition.

### Steps

Each scenario is composed of a series of steps. Each `step` block represents an action or group of actions that will be taken during the scenario. This action is defined by the `module` that the step calls, and the input variables that it sets.

#### Step attributes

- `description`: Provides a description of what the step does.
- `module`: Indicates which Terraform module is used in the step. The contents of that module is what will be performed in the step.
- `skip_step`: Used to skip the step entirely, according to a certain condition. For example, you might skip the `read_license` step when using a CE edition of a software.
- `depends_on`: Indicates explicit dependencies on previous steps. If you're using output from a previous step, it's recommended to include an explicit `depends_on` for that step.
- `verifies`: Indicates which quality requirement (`quality`) the step verifies. This is helpful for understanding why a certain step is included in the scenario, especially from a software quality perspective. Not all steps verify a quality requirement — for example, steps that set up AWS infrastructure do not verify any behavior of our software, but rather provision the resources we will use to later run and test our software.
- `variables`: The module used in the step may have some required or optional input variables. These are passed through as `variables` in the step.
- `providers`: Indicates which of the Terraform providers that we previously defined will be used in this step. See the below [Providers](#providers) section for further explanation.

### Outputs

You can define `output` blocks within a scenario to make values easily accessible from the Enos CLI, using the `enos scenario output` command. A common example of this is the `public_ips` output, which allows you to easily view the public IPs of the AWS instances where the Vault cluster has been spun up during the `create_vault_cluster` step. After launching the scenario, you can then run `enos scenario output <filter>` to view the public IPs of your instances and SSH in.

  ```sh
  output "public_ips" {
      description = "The Vault cluster public IPs"
      value       = step.create_vault_cluster.public_ips
  }
  ```

## Other Enos configs

Outside of the scenarios themselves, there are several other configurations required to make Enos work, which you will usually find in the `enos` directory.

By convention, we typically organize our Enos configurations into files, such as `enos-variables.hcl`, `enos-modules.hcl`, `enos-providers.hcl` and so on, all in the same directory. However, this is just a logical organization to make reading the code easier. As far as the Enos CLI is concerned, all of our scenarios and configs could all live in one big file together and it would read them the same way — as long as it is an `.hcl` file and prefixed with `enos-`. When you run an Enos command, Enos looks for all valid Enos configurations in the current or indicated directory.

In this section, we'll review a few of these configurations, why we need them, and how they affect our scenarios.

### Terraform configs (`enos-terraform.hcl`)

Here, we set any configs related to Terraform itself. These include:

- Required Terraform version: Enos requires at least Terraform 1.2.0
- Required Terraform providers. See below [Providers](#providers) section for further explanation.

### Modules (`enos-modules.hcl`)

In order to use a Terraform module in a scenario step, you must first define it in a `module` block outside of the scenario. By convention, we often group these definitions in `enos-modules.hcl`. The definition block must include the `source`, or where to find the module. Currently we use mostly local modules, so their source looks something like `"./modules/module-name"`. It's also possible to use remote modules; the source for these will be a URL to the Terraform registry where the module lives.

You can also include input variables in the module definition. We do this for modules that are used in multiple scenarios and/or steps, that have input variables that are the same across all implementations of the module. For example, take a module with the `vault_install_dir` input variable. Since we will use the same install directory for Vault across all implementations of this module, we can set this variable just once in the module definition. This way, we don't have to set it each time we use the module in a scenario step. For example:

  ```sh
  module "generate_secondary_token" {
    source = "./modules/generate_secondary_token"

    vault_install_dir = var.vault_install_dir
  }
  ```

### Providers

By now you may have noticed that we have `provider` blocks in a few places. This is because we need to define and enable Terraform providers at several different levels/scopes.

>**Note:** Future quality of life improvements will include the simplification of these provider configurations!

- **`terraform` block, commonly found in `enos-terraform.hcl`:** Here, we define some configs for Terraform. These include the minimum required version of Terraform, and any required Terraform providers. We define the latter here because Terraform needs to know which providers it needs to find, verify access to, and download. For example, we commonly require the AWS and Enos providers:

  ```sh
  terraform "default" {
    required_version = ">= 1.2.0"

    required_providers {
      aws = {
        source = "hashicorp/aws"
      }

      enos = {
        source  = "registry.terraform.io/hashicorp-forge/enos"
        version = ">= 0.4.0"
      }
    }
  }
  ```

- **`provider` block, commonly found in `enos-providers.hcl`:** Here, we set the configs that will allow Enos to use our required providers in the way that we want. There can be several definitions for the same provider, each with different configs. A common example of this is the Enos provider, which we often define with two different sets of SSH transport configs for different Linux distros:

  ```sh
    # This default SSH user is used in RHEL, Amazon Linux, SUSE, and Leap distros
    provider "enos" "ec2_user" {
      transport = {
        ssh = {
          user             = "ec2-user"
          private_key_path = abspath(var.aws_ssh_private_key_path)
        }
      }
    }

    # This default SSH user is used in the Ubuntu distro
    provider "enos" "ubuntu" {
      transport = {
        ssh = {
          user             = "ubuntu"
          private_key_path = abspath(var.aws_ssh_private_key_path)
        }
      }
    }
  ```

  This allows us to later reference the provider with the configuration that we need, depending on what distro we're using.

  >**Note:** In this case, `"enos"`, the first identifier of the provider, is the name of the provider as it's named in the public `hashicorp-forge` repo. We then create our own unique, second identifier — `"ec2-user"` and `"ubuntu"` — to distinguish between the two.

- **`providers` block within a scenario:** This indicates which of our previously defined providers we'll be using in this particular scenario. It's defined within the scenario itself. For example:

  ```sh
  providers = [
    provider.aws.default,
    provider.enos.ec2_user,
    provider.enos.ubuntu
  ]
  ```

  These make reference to the `provider` blocks we defined in `enos-providers.hcl`. Note that they first reference the type of resources (`provider`), then the name of the provider (`aws` or `enos`), and finally, the unique name assigned to the resource in the definition.

- **`providers` block within a step:** Here, we define which provider(s) we'll use within the step itself. We sometimes make use of a `local` variable to set this value, for example in order to simplify calling the correct provider for our current Linux distro:

```sh
  locals {
    enos_provider = {
      amzn2  = provider.enos.ec2_user
      leap   = provider.enos.ec2_user
      rhel   = provider.enos.ec2_user
      sles   = provider.enos.ec2_user
      ubuntu = provider.enos.ubuntu
    }
    ...
  }
...
  step "create_vault_cluster_targets" {
    description = global.description.create_vault_cluster_targets
    module      = module.target_ec2_instances
    depends_on  = [step.create_vpc]

    providers = {
      enos = local.enos_provider[matrix.distro]
    }
    ...
  }
```

If we do not explicitly include a `providers` block within a step, the provider that is defined (in our `required_providers`) with the alias `default`, or a provider that does not have an alias, will be made available in the step.

### Variables (`enos-variables.hcl`, `enos.vars.hcl`, environment variables)

#### Variable definitions

Enos variables are defined in `variable` blocks, which we commonly group together in `enos-variables.hcl`. Variable definitions can include the `type`, `description`, `default` value, and `sensitive` boolean.

#### Setting variable values

There are several ways to set variable values for Enos:

- Default value in variable definition
- Create a `enos-local.vars.hcl` file and set the value there. You can use the `enos.vars.hcl` file as an example. Ensure that your file is gitignored so that you do not accidentally commit any sensitive variable values. Example:

```sh
  aws_region = "us-east-1"
```

- Set it as an environment variable using the prefix `ENOS_VAR_` in the terminal where you will run Enos commands. Example:

```sh
  export ENOS_VAR_aws_region=us-east-1
```

### Globals and locals (`enos-globals.hcl`, `locals` blocks)

You can also create both local and global variables.

- **Local variables:** Scoped to a single scenario; live within the `scenario` block. Common reasons to create a local variable include:
  - You want to assign a value to a variable dynamically depending on a matrix variant (`matrix.<variable>`). Matrix variants are scoped to the scenario level, so if you want to reference one, you must use a local variable, not a global one. For example:

    ```sh
    locals {
      manage_service    = matrix.artifact_type == "bundle"
    }
    ```

  - You want to organize some other values into a format that is more clear or easier to access within the scenario. For example:

    ```sh
      locals {
        enos_provider = {
          amzn2  = provider.enos.ec2_user
          leap   = provider.enos.ec2_user
          rhel   = provider.enos.ec2_user
          sles   = provider.enos.ec2_user
          ubuntu = provider.enos.ubuntu
        }
      }
    ```

- **Global variables:** Scoped to the directory they're in; we often define them in `enos-globals.hcl`. We mostly use global variables for items that will be used across multiple scenarios. In this way, instead of repeatedly defining the same variable as a `local` in each scenario, we just define it once as a `global`.

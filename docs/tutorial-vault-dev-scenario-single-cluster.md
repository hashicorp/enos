# TUTORIAL: Vault single cluster `dev` scenario

## About the `dev` scenarios

[This scenario](https://github.com/hashicorp/vault/blob/main/enos/enos-dev-scenario-single-cluster.hcl), along with the PR replication `dev` scenario, was originally created to meet a need for an Enos scenario that was easier and faster to run locally. The rest of our scenarios in the Vault repos are primarily designed to be run in CI. They perform many verifications of metadata and software behavior, require several inputs, and have large matrices. The configurations for these scenarios are automated in CI. However, when running the scenarios locally, they represent a series of manual steps and inputs that can be tedious for a user who doesn't require these verifications. *Thus, we designed these two scenarios to be more "lightweight", performing few or no verifications, using smaller matrices, and requiring less inputs.*

## When to use this scenario

We recommend using this scenario when you need to quickly spin up a real Vault cluster on live AWS infrastructure, in the configuration of your choosing (arm64 or amd64 architecture, type of Vault artifact, Consul or raft backend, Linux distro, Vault edition, seal type). This could be for reasons including:

- Bug fixes: reproduce a specific environment to test your bug fix
- Feature development: test the code in your local branch to see how it behaves, before even opening a PR / pushing to Github

Once you spin up a Vault cluster using Enos, you can then [SSH in](./troubleshooting.md#ssh-into-an-instance).

## How to run this scenario

### Prerequisites

In order to run this scenario, you'll need to perform a few prerequisite steps around Enos installation and credential setup:

1. [Install Enos](./getting-started.md#install-enos).
2. Authenticate to AWS and export the credentials to your terminal. See more details [here](./getting-started.md#authenticate-to-aws).

3. Set the `aws_ssh_private_key_path` and `aws_ssh_keypair_name` variables with your AWS key pair name and private key. See more details [here](./getting-started.md#set-your-aws-key-pair-name-and-private-key).

```sh
  # Option: set them as ENOS_VARs in your terminal:
  export ENOS_VAR_aws_ssh_private_key_path=path/to/your/private-ssh-key.pem
  export ENOS_VAR_aws_ssh_keypair_name=your-ssh-key-name

  # Option: set them in your enos-local.vars.hcl file
  aws_ssh_private_key_path=path/to/your/private-ssh-key.pem
  aws_ssh_keypair_name=your-ssh-key-name
```

4. If you're not already familiar with Enos, we recommend reviewing the [basic parts](./scenarios-and-configurations.md) of an Enos scenario and the [basics of running a scenario](./running-a-scenario.md).

### Choose your filters and set variables

You'll need to make a few decisions about how you want to configure the scenario. These decisions include which combination of [matrix](./scenarios-and-configurations.md#matrix) variants you'd like to run (and thus, the [filter](./running-a-scenario.md#filters) you'll set on your Enos command), and the [variables](./getting-started.md#set-enos-variables) required for those variants.

You can also see an overview of the scenario steps and the variables for each one by using the [`outline`](./running-a-scenario.md#outline) command.

For this scenario, the matrix looks like the following:

```sh
  matrix {
    # What architecture to use for your Vault cluster
    arch     = ["amd64", "arm64"]
    # What type of Vault artifact to use
    artifact = ["local", "deb", "rpm", "zip"]
    # What backend storage to use
    backend  = ["consul", "raft"]
    # What Linux distro to use
    distro   = ["amzn", "leap", "rhel", "sles", "ubuntu"]
    # What Vault edition to use
    edition  = ["ce", "ent", "ent.fips1402", "ent.hsm", "ent.hsm.fips1402"]
    # What Vault seal type to use
    seal     = ["awskms", "pkcs11", "shamir"]

    exclude {
      edition = ["ent.hsm", "ent.fips1402", "ent.hsm.fips1402"]
      arch    = ["arm64"]
    }

    exclude {
      artifact = ["rpm"]
      distro   = ["ubuntu"]
    }

    exclude {
      artifact = ["deb"]
      distro   = ["rhel"]
    }

    exclude {
      seal    = ["pkcs11"]
      edition = ["ce", "ent", "ent.fips1402"]
    }
  }
```

> **_Note:_** The `exclude` blocks eliminate certain combinations that are unsupported. If you try to run the scenario with an unsupported variant combination (e.g. trying to run a Red Hat `rpm` on `ubuntu`), you will get an error message indicating that `no scenarios found matching filter <your:filters>`.

> **_Note:_** This scenario supports finding and installing any released `linux/amd64` or `linux/arm64` Vault artifact as long as its version is >= 1.8.

You'll need to choose your scenario configuration options for each of the items in the matrix: `arch`, `artifact`, `backend`, `distro`, `edition`, and `seal`. Several of these variants will require input variables:

- `artifact`: This determines where you are getting your Vault artifact from, and what type of artifact you'll use.
  - `artifact:local`: This variant builds Vault from your local branch.
    - Ensure that your `edition` variant matches the repo you're building from (e.g. use `edition:ce` if building from the Vault CE repo and `edition:<an ENT edition>` if building from the Vault Enterprise repo)
  - `artifact:deb`: This variant downloads a Debian `.deb` package of Vault from Artifactory.
    - Requires:
      - `artifactory_username`: See [here](./running-a-scenario.md#4-choose-where-you-will-get-a-vault-artifact) for information on getting an Artifactory username and token.
      - `artifactory_token`
      - `vault_product_version`: What version of Vault Enos should get
    - Must use `distro:ubuntu`
  - `artifact:rpm`
    - Enos will fetch a Red Hat `.rpm` package from Artifactory
    - Requires:
      - `artifactory_username`: See [here](./running-a-scenario.md#4-choose-where-you-will-get-a-vault-artifact) for information on getting an Artifactory username and token.
      - `artifactory_token`
      - `vault_product_version`: What version of Vault Enos should get
    - Must use `distro:rhel`, `distro:amzn`, or `distro:sles`
  - `artifact:zip`
    - Enos will fetch a `.zip` bundle from releases.hashicorp.com
    - `vault_product_version`: What version of Vault Enos should get
    - Can use any distro
- `backend`
  - `backend:raft`: Uses Raft as backend storage for Vault. No further configuration required.
  - `backend:consul`: Uses Consul as backend storage for Vault.
    - `dev_consul_version`: The version of Consul to use when using Consul for storage. If your preferred Consul version differs from the default value in `enos-dev-variables.hcl`, set the value with this variable.
    - `backend_edition`: The backend edition (functionally, the Consul edition, since `backend:raft` is not affected by edition). The default value in `enos-variables.hcl` is `ce`; if you need to use Consul Enterprise, set to `ent`.
    - `backend_license_path`: If you use an enterprise edition of Consul, set this path to a valid Consul enterprise edition license.
- `edition`
  - `edition:ce`
    - No additional variables required
  - `edition:<any ENT edition>`
    - `vault_license_path`: The path to a valid Vault enterprise edition license. If you're using any ENT edition of Vault, set to the path where you have a Vault license stored.

Other vars:

There are a variety of other variables that you can set to configure your scenario. The variables specific to the dev scenarios are found in `enos-dev-variables.hcl`, while variables that are shared by the CI _and_ dev scenarios are found in `enos-variables.hcl`. Here are a few that might be of interest; for the full list, see `enos-variables.hcl`.

- `dev_build_local_ui`: Whether or not to build the web UI when using the local builder var. If the assets have already been built we'll still include them. Default is `false`; if you need to build the UI, set to `true`.
- `dev_config_mode`: The method to use when configuring Vault. When set to `env` we will configure Vault using `VAULT_` style environment variables if possible. When `file` we'll use the HCL configuration file for all configuration options. Default is `file`; otherwise, set to `env`.
- `aws_region`: What AWS region to run your instances in.
- `backend_log_level`: The server log level for the backend. Supported values include `trace`, `debug`, `info`, `warn`, `error`.
- `distro_version_<distro>`: This set of variables sets the version for the Linux distros we support.
- `vault_instance_count`: How many instances to create for the Vault cluster. Default is 3.
- `vault_log_level`: The server log level for Vault logs. Supported values (in order of detail) are trace, debug, info, warn, and err.

### Launch or run your scenario

Now that you have set your variables and selected what matrix variants you'll use, you can run the scenario using either the [`launch`](./running-a-scenario.md#launch) (the cluster will be spun up and remain running) or [`run`](./running-a-scenario.md#run) (the cluster will be spun up and then immediately torn down) command.

Example:

```sh
enos scenario launch dev_single_cluster arch:amd64 artifact:local backend:raft distro:ubuntu edition:ce seal:awskms
```

### SSH in

If you have `launch`ed your scenario and want to manually test Vault, you can get the public IP of your instance and SSH in. See instructions [here](./running-a-scenario.md#7-optional-troubleshooting-if-something-went-wrong).

### Troubleshoot

If you encounter errors, review some commonly encountered errors and their solutions [here](./troubleshooting.md).

### Destroy

In the interest of your AWS bill, remember to [`destroy`](./running-a-scenario.md#destroy) your resources when you're done.

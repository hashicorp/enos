# Vault smoke tests

Here you'll find the Vault & Vault Enterprise smoke tests. Each scenario is
designed to test different behavior against Vault clusters that have been created
with staged Vault and Vault Enterprise artifact bundles. Each test scenario is
written as Terraform and utilizes the [enos Terraform provider](https://github.com/hashicorp/enos-provider)
to dynamically configure instances, perform lifecycle events, and test for
various conditions.

## Requirements

1. Terraform v0.15.3 or higher.
1. `make`.
1. Access to an AWS account and an AWS SSH key pair in that account.
1. Access to Artifactory and an Artifactory API token.
1. Access to the `hashicorp-qti` TFC org, where the enos Terraform modules
  that power the tests reside. You can find the `vault` team token in 1Password as `hashicorp-qti HCP Token`.

## Setup

1. Use `doormat` to create temporary AWS credentials and set up the AWS credential
  chain in your preferred way.
1. Update the `atifactory_username` and `artifactory_token` variables in [terraform.tfvars](./terraform.tfvars).
  Your artifactory username will be your HashiCorp email address. If you don't
  have a token you can generate one [here](https://artifactory.hashicorp.engineering/ui/admin/artifactory/user_profile).
1. Update the `vault_enterprise_product_revision`,`vault_oss_product_revision`,
  and `vault_product_version` variables in [terraform.tfvars](./terraform.tfvars).
  The product revisions refer to the `PRODUCT_REVISION` that you used when
  staging the releases. It should usually be the git SHA of the last commit
  on the branch. The product version refers to the `PRODUCT_VERSION` that was
  set when you staged the releases. It should be the desired Vault version.
1. Update the `vault_enterprise_initial_release` and `vault_oss_initial_release`
  variables in [terraform.tfvars](./terraform.tfvars) if you're going to run
  the upgrade test scenario.
1. Make any changes to the `vault_product_editions_to_test` variable in [terraform.tfvars](./terraform.tfvars).
  By default it includes all supported editions of Vault and Vault Enterprise.
1. Update the `aws_ssh_key_pair_name` and `aws_ssh_private_key_path` variables in 
  [terraform.tfvars](./terraform.tfvars) with your AWS key pair name and the path
  to the private key on your machine. If you don't have an AWS key pair you can
  use `doormat --aws console` to login to the AWS console and generate one. Make
  sure the AWS key pair is in the same region that you've configured in [terraform.tfvars](./terraform.tfvars).
1. Update the [mirror.tfrc](./mirror.tfrc) Terraform CLI config file with the hashicorp-qti
  `vault` team token. You can find it in 1Password as `hashicorp-qti HCP Token`.
1. If you're planning to test a version of Vault Enterprise that is 1.8-rc1 or higher,
  you will also need [to get a Vault Enterprise license](https://license.hashicorp.services/)
  and write it to a local file. After you've written the file, update the
  `vault_license_path` variable in [terraform.tfvars](./terraform.tfvars) with
  the fully qualified path to it. If you're testing a prior version, set
  `vault_license_path` to `null` or comment it out.


## Run the smoke tests

All of the test scenarios are defined in a unique sub-directory that includes the
Terraform HCL that defines the test case. If you wish to override a default variable in the
test case you should set it in the [terraform.tfvars](./terraform.tfvars) file.

To execute all tests and cleanup after, run `make`.

To run an individual test:
1. Initialize it. You only need to do this once. `make test-name-init`
1. Run it. `make test-name-run`
1. Destroy it. `make test-name-destroy`

Note: sometimes transient errors occur when provisioning all of the required resources.
Most of the time it's okay to re-run the make command to continue the test.

Consult the [Makefile](./Makefile) to determine which test targets are available.

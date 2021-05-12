# Vault Enterprise smoke tests

Here you'll find the Vault Enterprise smoke tests. All of these smoke tests are
designed to create Vault clusters from staged Vault Enterprise artifacts and run
various test scenarios. All tests are written as Terraform using the the
[enos Terraform provider](https://github.com/hashicorp/enos-provider) to dynamically
configure instances, perform lifecycle events, and test for various conditions.

## Requirements

1. Terraform v0.15.0 or higher.
1. `make`.
1. Access to an AWS account and an AWS SSH key pair.
1. Access to Artifactory and an Artifactory token.
1. Access to the hashicorp-qti TFC org to access the enos Terraform modules.
  You can find the `vault` team token in 1Password as `hashicorp-qti HCP Token`.

## Setup

1. Use `doormat` to create temporary AWS credentials and set up the AWS credential
  chain in your preferred way.
1. Update [terraform.tfvars](./terraform.tfvars) with your Artifactory username
  and token. Your artifactory username will be your HashiCorp email address. If you
  don't have a token you can generate one [here](https://artifactory.hashicorp.engineering/ui/admin/artifactory/user_profile).
1. Update the `productRevision` and `productVersion` variables in [terraform.tfvars](.terraform.tfvars)
  with the git SHA of the staged build you wish to test and its corresponding
  version.
1. Update [terraform.tfvars](./terraform.tfvars) with your AWS key pair name
  and the path to the private key on your machine. If you don't have an AWS key
  pair you can use `doormat --aws console` to login to the AWS console and generate
  one. Make sure the AWS key pair is in the same region that you've configured in
  [terraform.tfvars](./terraform.tfvars).
1. Update the [mirror.tfrc](./mirror.tfrc) Terraform CLI config file with the hashicorp-qti
  `vault` team token. You can find it in 1Password as `hashicorp-qti HCP Token`.

## Run the smoke tests

All of the test scenarios are defined in their own sub-directory that includes the
Terraform HCL that defines the test case. If you wish to override a default variable in the
test case you should set it in the [terraform.tfvars](./terraform.tfvars) file.

To execute all tests and cleanup after, run `make`.

To run an individual test:
1. Initialize it. You only need to do this once. `make test-name-init`
1. Run it. `make test-name-run`
1. Destroy it. `make test-name-destroy`

Consult the [Makefile](./Makefile) to which test targets are available.

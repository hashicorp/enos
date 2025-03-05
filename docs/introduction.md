# Introduction

Enos is a tool for powering Software Quality as Code by writing Terraform-based quality requirement scenarios using a composable, modular, and declarative language.

## What problem does Enos solve?

Customers use HashiCorp products in countless different environments and combinations of platforms, runtimes, system resources, access patterns, storage backends, plugins, operating modes, and configurations. If we don't test our software in a wide range of environments on actual live infrastructure, we introduce risk that the software may not behave as expected in a customer's environment.

Enos provides a scalable, effective way for us to spin up live infrastructure upon which to test our software, across a matrix of different configurations. This allows us to have more confidence that our software will behave as expected in a wide range of environments, as well as reproduce specific environments.

Ultimately, the goals of Enos are to:

* Reduce escaped defects
* Improve the user experience of our products
* Improve business, customer, and community outcomes

## What is Enos?

Enos is a Terraform-based framework. In practice, an Enos scenario consists of one or more HCL configuration files that make use of Terraform providers and modules. These files are read and executed by the Enos CLI to spin up the specified resources and perform the specified verifications on them.

The Enos framework is made up of several components:

* **Terraform** is the engine of Enos. It powers cloud infrastructure provisioning.
* The **Domain Specific Language (DSL)** allows us to describe the scenario we want to run, including resources we want to spin up and actions or tests we want to perform upon them. Its syntax is very close to Terraform, with some differences that allow us to abstract away some complexities.
* The **Command Line Interface (CLI)** allows us to outline, validate, execute, obtain outputs from, and tear down the scenario and resources we’ve described.
* The **Enos Terraform provider** gives us access to Terraform resources and data sources that are useful for common Enos tasks like: running commands locally or on remote machines, discovering artifacts or local system information, downloading and installing artifacts, or setting up clusters of HashiCorp products. Cloud-specific Terraform providers like the AWS provider allow us to interact with resources supported by that platform.
* **Terraform modules** allow us to manage and configure product- and cloud-specific infrastructure resources as well as perform verifications of expected behavior of our software.
* The **Enos Homebrew formula** allows for easy installation of Enos on your local machine.
* The **`action-setup-enos` Github Action** allows for easy installation of Enos in Github Actions workflows.

## What can I use Enos for?

Currently, there are two primary use cases for Enos: quality testing in the CI/CD pipeline, and reproduction/testing in development or support workflows.

### CI/CD

Enos is already heavily integrated into the Vault and Boundary repos as a testing tool. At various points throughout the pipeline, we randomly select a group of Enos scenarios to run. These scenarios test various Vault features and behaviors on live infrastructure in various configurations, emulating how Vault might run in the wild.

In the Vault repos, most of the resources in the `enos` directory are designed for CI/CD purposes, including scenarios and modules that test various aspects of the software's behavior on live infrastructure.

### Development and support workflows

Enos is also an excellent tool for reproducing customer environments and for testing software in local development. For example, the Vault Support team uses it to quickly spin up a Vault cluster to emulate a customer's environment. A product engineer might also use Enos to spin up a cluster using a Vault artifact built from their local branch, allowing them to verify the software behavior on live infrastructure before they've even pushed their changes to a PR.

You can see the existing scenarios designed for this purpose in the [Vault Enos directory](https://github.com/hashicorp/vault/blob/main/enos). These scenarios are denoted with the naming convention `enos-dev-scenario-*.hcl`. Each scenario includes descriptions and instructions for use. To see a summary for each one, run `enos scenario outline <scenario-name>` from that directory.

## How can I get started with Enos?

1. [Install](./getting-started.md#install-enos) Enos.
1. [Set up](./getting-started.md#set-credentials) cloud/SSH credentials.
1. Review Enos documentation and tutorials:

  * [Scenarios and configurations](./scenarios-and-configurations.md): learn about the basic concepts and components of an Enos scenario
  * [Running a scenario](./running-a-scenario.md): learn about the most common Enos commands, and recommended workflows for running a scenario
  * [Troubleshooting](./troubleshooting.md): common Enos errors and their solutions
  * Vault [single cluster](./tutorial-vault-dev-scenario-single-cluster.md) `dev` scenario tutorial: a lightweight scenario for spinning up a single Vault cluster
  * Vault [PR replication](./tutorial-vault-dev-scenario-pr-replication.md) `dev` scenario tutorial: a lightweight scenario for spinning up two Vault clusters, with the secondary cluster configured with performance replication from the primary cluster

## Feedback and support

Enos is an ongoing project and your feedback is valuable to us! If you encounter any bugs, have ideas for feature requests, think we could improve an error message, or any other improvement to the user experience, we would love to hear from you.

We welcome PRs for bug fixes, new features, or other improvements on all of the related repos:

* [Enos](https://github.com/hashicorp/enos)
* [Enos provider](https://github.com/hashicorp-forge/terraform-provider-enos/)
* [action-setup-enos Github Action](https://github.com/hashicorp/action-setup-enos)
* [Enos documentation](https://github.com/hashicorp/engineering-handbook)

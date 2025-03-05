# Troubleshooting with Enos

So you installed Enos, set up all your credentials, and tried to run a scenario.

It failed. Now what?

This guide will give you some ideas on how to troubleshoot what went wrong.

## When and where did it fail?

You can learn something about why a scenario failed based on the Enos/Terraform action it failed on.

When you run an Enos scenario, you will see output that indicates which action is being performed: `generate`, `init`, `validate`, `plan`, `apply`, and `destroy`. These correspond to a series of Enos and Terraform commands that are being performed in the background. If your scenario fails, knowing which of these actions it failed on can help you understand what went wrong.

### `generate`

This step generates the Terraform root modules required by your scenario. Things can go wrong if it can't find or access the modules at the location specified.

This step performs the following actions:

* Initializes a working directory containing Terraform configuration files
* Finds, downloads, and installs providers according to the location specified in their `source`

Things can go wrong if it can't find or access the modules at the location specified.

### `validate`

This step verifies that your scenario and associated configurations are valid. Things can go wrong if you have syntax errors, invalid variable references, etc.

### `plan`

This step creates a Terraform execution plan from your Enos configurations. It also validates credentials for the required providers. Things can go wrong if you don't have valid credentials for a provider like AWS, or if your scenario's execution plan depends on variables that can't be known before execution, like instance IPs or data sources that request information from AWS (in this case, Terraform can't form an execution plan since it doesn't have these values yet).

### `apply`

This step carries out the steps listed in your scenario. Here, it's possible to see errors related to:

* Enos configuration issues
* The actual software or your code
* Occasional transient AWS flakiness

### `destroy`

This step tears down any infrastructure that was created as part of your scenario. Here, it's possible to see [errors related to actions performed on your infrastructure outside of Enos](#error-creating-ec2-subnet-invalidvpcidnotfound), creating a mismatch between the state Enos is aware of and the actual state of the resources.

## Check the current state of your instance(s)

### If your scenario failed at `generate`, `validate`, or `plan`:

You should review your syntax, variable dependencies, and module source locations. In these cases, Enos has not yet reached the `apply` phase, and there are no live instances to investigate.

### If your scenario failed during `apply`:

Depending on what scenario step failed, you may already have live instances whose state you can investigate. There are several ways to look at the current state, including using `enos scenario exec` to execute Terraform commands directly, and SSHing into the instance itself to review logs.

> **Tip:** When your Enos scenario fails, scroll to the top of the output to find the _first_ error. Often, there is a long tail of items that fail only because they were dependent on the original error, and are not actually broken themselves.

#### Access Terraform commands with `enos scenario exec`

The `enos scenario exec` command allows you to access Terraform commands directly. If your scenario made it to the `plan` stage or later, you can get some useful information this way. Here is an example of a workflow you can use with this command:

1. View a list of all of the scenario's resources:

  `enos scenario exec --cmd 'state list' <scenario-name> <scenario-filter>`

2. Copy the full name of the resource that failed e.g. `module.create_primary_cluster.module.install_packages.enos_remote_exec.distro_repo_setup["2"]`.

3. View the state of that particular resource:

  `enos scenario exec --cmd 'state show <resource-name> <scenario-name> <scenario-filter>`

This will show STDOUT, STDERR, and the contents of any data sources of the resource.

### SSH into an instance

If your scenario has made it into `apply` and past the successful creation of infrastructure, you can SSH into the instance to try to figure out what went wrong.

1. Get public IP of the instance you're interested in. This information is usually displayed as part of the error message:

  ```sh
  SSH Transport Config:
    user : ubuntu
    host : 54.242.91.178 # This is the public IP
    private_key : null
    private_key_path : <path>
    passphrase : null
    passphrase_path : null
  ```

  You can also obtain this information from the scenario `output`:

  `enos scenario output <scenario-name> <scenario-filter>`

2. SSH into the instance, using `ubuntu` for the username when using Ubuntu Linux, or `ec2-user` for all other supported Linux distros.

  `ssh -i ~/path/to/your/private-key.pem <linux-user>@<instance-public-ip>`

3. Check Vault logs. Note that Enos automatically adds `vault` to the `PATH` for you and configures the `VAULT_ADDR` and `VAULT_TOKEN` with the root token.

  `sudo journalctl -u vault`

## Common errors and solutions

### Error: Retrieving AWS account details

```sh
Error: exit status 1

Error: Retrieving AWS account details: validating provider credentials: retrieving caller identity from STS: operation error STS: GetCallerIdentity, https response error StatusCode: 403, RequestID: 9cf3cb73-ff1a-4596-9973-3254cff857fb, api error ExpiredToken: The security token included in the request is expired

  with provider["registry.terraform.io/hashicorp/aws"],
  on scenario.tf line 16, in provider "aws":
  16: provider "aws" {

Error: Search Failed
```

Enos encountered a permissions error when trying to access the AWS provider. It's a common mistake — you may have forgotten to authenticate to AWS and export your AWS credentials to your terminal. Follow the instructions [here](./getting-started.md#authenticate-to-aws).

### Error: creating EC2 VPC: VpcLimitExceeded

When you `launch` or `run` your scenario, you get an error like the following:

```sh
Error: creating EC2 VPC: VpcLimitExceeded: The maximum number of VPCs has been reached.
    status code: 400, request id: 0b24af94-b828-4920-8452-3893087cfb28

  with module.create_vpc.aws_vpc.enos_vpc,
  on .terraform/modules/create_vpc/vpc.tf line 1, in resource "aws_vpc" "enos_vpc":
   1: resource "aws_vpc" "enos_vpc" {
```

Each AWS account has a limit on the number of VPCs that it can have in a region at any given time. If you or your teammates have run several scenarios, and/or have not cleaned up unused AWS resources from past scenarios, it’s possible you could hit this limit.

As a temporary solution, you can:

* Destroy any of your own unused resources using `enos scenario destroy`.
* Manually delete resources in the AWS console. Before you do this, you may need to check with your teammates to understand which resources are in use and which are not.
* Try running your scenario in another AWS region. VPC limits are per region so you may still have capacity in another region.

If this is a frequently recurring issue, we recommend requesting an increase on the VPC limit for your AWS account and region:

1. [Authenticate to AWS](./getting-started.md#authenticate-to-aws)
1. Click the arrow icon to enter the AWS console for the relevant account.
1. Follow the instructions to [request a quota increase](https://docs.aws.amazon.com/servicequotas/latest/userguide/request-quota-increase.html).

### Error: creating EC2 Subnet: InvalidVpcID.NotFound

```sh
Error: exit status 1

 Error: creating EC2 Subnet: InvalidVpcID.NotFound: The vpc ID 'vpc-04f681422e7eb4dbe' does not exist
     status code: 400, request id: 292be92d-c894-4bae-992c-e847555507df

   with module.create_base_infra.aws_subnet.enos_subnet["0"],
   on .terraform/modules/create_base_infra/vpc.tf line 15, in resource "aws_subnet" "enos_subnet":
   15: resource "aws_subnet" "enos_subnet" {
```

This error can occur in the following circumstances:

* You spin up a scenario.
* You leave it running and do not `destroy` it.
* The AWS instances get terminated outside of Enos, through an automated or manual resource cleanup via the AWS console or CLI.
* Later, you attempt to perform an Enos action on the scenario, but get an error message indicating that the resource can't be found.

Enos (via Terraform) only knows about the current state of a scenario's resources _according to what actions Enos itself has performed_. If actions have been performed upon those resources outside of Enos — for example, through AWS directly — Enos does not know about this, and therefore believes those resources should still exist.

To resolve this, you need to start a fresh run of your Enos scenario by removing the automatically generated Terraform files, including the state file. Each run of a unique scenario variant gets its own subdirectory under the `.enos` directory, which includes the Terraform files that Enos has created in order to run your scenario. If you don't need to preserve any state files from any other scenario variant runs, you can simply delete the entire `.enos` directory. If you want to preserve the state files from any other runs, you'll need to first determine which is the subdirectory that corresponds to the run that needs a fresh start. Here is one way to identify the subdirectory of your current scenario:

  1. `cd` into the `enos/.enos/` directory.
  1. Run `ls lah` to see the timestamps of when each directory was most recently changed (in most cases, the scenario in question is the most recently changed one).

Once you have deleted the subdirectory, you can try your Enos command again.

### Execution Error — `expected` vs `got` for Vault version/edition/revision/build date

During `apply`, you get an error like the following:

```sh
Error: Execution Error
│
│   with module.verify_vault_version.enos_remote_exec.verify_cli_version["2"],
│   on ../../modules/vault_verify_version/main.tf line 61, in resource "enos_remote_exec" "verify_cli_version":
│   61: resource "enos_remote_exec" "verify_cli_version" {
│
│ failed to execute commands due to: running script:
│ [/Users/your-user/repos/vault/enos/modules/vault_verify_version/scripts/verify-cli-version.sh]
│ failed, due to: 1 error occurred:
│       * executing script: Process exited with status 1: 
│ The Vault cluster did not match the expected version:
│ --- expected
│ +++ got
│ @@ -1 +1 @@
│ -Vault v1.18.2+ent (e0f4178b7592891b63bc7e82b843b3f16d3fd8a8), built
│ 2024-06-04T09:37:10Z
│ +Vault v1.18.2+ent (b0e4bcd70f2e0ed85eef7e016998f3e008752034), built
│ 2024-11-20T11:25:07Z
```

**If this error occurs while running locally:**

The `vault_verify_version` module is used in CI to verify the version, edition, revision SHA, and build date of the artifact that we are using. If you are running one of these scenarios locally, you'll need to set these variables.

> **Note:** This verification step has been removed in our `dev`-specific scenarios to make it simpler to run these locally, so you don't need to set these variables for those scenarios.

The `expected` data includes the values that you set, and therefore what the verification step is expecting to see from the artifact it's checking. The `got` data is what the verification step has actually received from the artifact. This error means that there is a mismatch between the two sets of data, and that you have told it to expect different metadata than what the artifact actually is. In the error above, we have set the Vault version/edition correctly, but the revision and build date don't match. Set these variables to match the `got` values:

```sh
export ENOS_VAR_vault_revision=b0e4bcd70f2e0ed85eef7e016998f3e008752034
export ENOS_VAR_vault_build_date=2024-11-20T11:25:07Z
```

### Error: creating EC2 Instance: AWS Marketplace subscription error

During `apply`, you get an error like the following:

```sh
Error: exit status 1

Error: creating EC2 Instance: operation error EC2: RunInstances, https response error StatusCode: 401, RequestID: 0a16885e-6ce5-4cf7-9e3e-e3c72cd6ddca, api error OptInRequired: In order to use this AWS Marketplace product you need to accept terms and subscribe. To do so please visit https://aws.amazon.com/marketplace/pp?sku=147aomaaws9zx4er41jp3mozy

  with module.create_vault_cluster_targets.aws_instance.targets["2"],
  on ../../modules/target_ec2_instances/main.tf line 242, in resource "aws_instance" "targets":
  242: resource "aws_instance" "targets" {
```

Some Vault scenario matrices include the `distro:leap` variant, which uses the openSUSE Leap Linux distro. In order to use this variant, for which Enos will obtain a Leap AWS AMI, you must first accept the subscription for those AMIs, using the AWS account you wish to use. To verify your subscription, authenticate to your AWS account via Doormat, and visit the following links:

* [arm64 AMI](https://aws.amazon.com/marketplace/server/procurement?productId=a516e959-df54-4035-bb1a-63599b7a6df9)
* [amd64 AMI](https://aws.amazon.com/marketplace/server/procurement?productId=5535c495-72d4-4355-b169-54ffa874f849)

### Error: reading EC2 AMIs: operation error EC2: DescribeImages, https response error StatusCode: 403

During `plan`, you get an error like the following:

```sh
  Error: exit status 1

  Error: reading EC2 AMIs: operation error EC2: DescribeImages, https response error StatusCode: 403, RequestID: e4557444-a53e-4cc6-8e69-550571d9a72a, api error UnauthorizedOperation: You are not authorized to perform this operation. User: arn:aws:sts::734500048898:assumed-role/qt_dev-developer/your-user@hashicorp.com is not authorized to perform: ec2:DescribeImages with an explicit deny in a session policy

    # Note: this error will likely happen with all of the module.ec2_info.data.aws_ami.*
    # resources, as these attempt to get AMI information for several Linux distros
    # and architectures.
    with module.ec2_info.data.aws_ami.amzn_2["x86_64"],
│   on ../../modules/ec2_info/main.tf line 65, in data "aws_ami" "amzn_2":
│   65: data "aws_ami" "amzn_2" {
```

This error can occur if you authenticated to Doormat and then changed IP addresses — for example, switching wifi networks. To resolve it, [re-authenticate](./getting-started.md#authenticate-to-aws) with Doormat using the `-f` flag to force a fresh set of credentials.

### Error: executable file not found in $PATH

During `apply`, you get an error like the following:

```sh
  Error: Execution Error

  with module.build_or_find_vault_artifact.enos_local_exec.build,
  on ../../modules/build_local/main.tf line 55, in resource "enos_local_exec" "build":
  55: resource "enos_local_exec" "build" {

  failed to execute commands due to: running script:
  [/Users/your-user/repos/vault/enos/modules/build_local/scripts/build.sh]
  failed, due to: failed to execute command due to: exit status 2

  output:
  + npm install --global yarn

  changed 1 package in 311ms
  + export CGO_ENABLED=0
  + CGO_ENABLED=0
  ++ git rev-parse --show-toplevel
  + root_dir=/Users/your-user/repos/vault
  + pushd /Users/your-user/repos/vault
  + '[' -n false ']'
  + '[' false = true ']'
  + make ci-build
  builtin/logical/pki/path_config_acme.go:420: running "enumer": exec:
  "enumer": executable file not found in $PATH
  builtin/logical/pki/pki_backend/common.go:29: running "enumer": exec:
  "enumer": executable file not found in $PATH
  command/operator_generate_root.go:26: running "enumer": exec: "enumer":
  executable file not found in $PATH
  command/agent/exec/exec.go:29: running "enumer": exec: "enumer": executable
  file not found in $PATH
  command/agentproxyshared/cache/api_proxy.go:19: running "enumer": exec:
  "enumer": executable file not found in $PATH
  command/healthcheck/healthcheck.go:255: running "enumer": exec: "enumer":
  executable file not found in $PATH
  helper/testhelpers/testhelpers.go:29: running "enumer": exec: "enumer":
  executable file not found in $PATH
  internalshared/configutil/kms.go:41: running "enumer": exec: "enumer":
  executable file not found in $PATH
  vault/core.go:3091: running "enumer": exec: "enumer": executable file not
  found in $PATH
  vault/quotas/quotas.go:33: running "enumer": exec: "enumer": executable file
  not found in $PATH
  make: *** [ci-build] Error 1
```

You might get this error if using the `artifact:local` / `artifact_source:local` matrix variant. This matrix variant builds Vault from your local branch in order to spin up a cluster using your current changes. The `executable file not found in $PATH` error usually indicates you might be missing one of the tools used to build Vault — in this case, `enumer`. To resolve this, from the main Vault directory, run the following to make sure you have all the tools you'll need:

  `make tools`

If that does not fully resolve the error, try running:

  `make bootstrap`

### Error: no scenarios found matching filter <your:filters>

Before any Enos operations, your scenario immediately fails out, ending in an error that looks something like:

```sh
  Error: no scenarios found matching filter 'dev_pr_replication arch:arm64 artifact:local distro:ubuntu edition:ce primary_backend:raft primary_seal:shamir secondary_backend:raft secondary_seal:awskms'
```

First, ensure that you are in the `/enos` directory (or wherever your Enos scenarios live). If you run Enos from a directory that does not contain any valid Enos scenarios, it won't find any, and thus will fail.

Second, there is a chance that there is some kind of typo in your Enos filter. To make sure you're typing out a valid filter that matches the matrix and possible variants, use the `enos scenario list` function to get Enos to print them out. For most scenarios, running `enos scenario list <scenario-name>` will print a _very_ long list of possible combinations, so one troubleshooting method is to start adding in filters one by one until you get down to a more manageable list, which you can then copy and paste from, or identify your typo.

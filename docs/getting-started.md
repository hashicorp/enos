# Getting started with Enos: installation and setup

For most macOS users, we recommend installing Enos via the Homebrew formula, using the instructions below.

If you prefer not to use Homebrew, Enos binaries are also available as [Github Releases](https://github.com/hashicorp/enos/releases/).

## Install Homebrew

Install Homebrew if you have not done so previously. To do this, run the following command, which explains what it does and then pauses before it does it.

>**Note:** This command is for macOS or Linux shells. For alternate installation options for Homebrew, refer to the Homebrew [documentation](https://docs.brew.sh/Installation).

```sh
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

## Install Enos

Install Enos via Homebrew:

> **Note:** The Enos Homebrew formula includes a Terraform dependency. If you prefer not to use the Homebrew installation of Terraform, you can add the `--ignore-dependencies` flag to the above command. Note that Enos requires Terraform version 1.2.0 or above.

```sh
brew tap hashicorp/tap && brew update && brew install hashicorp/tap/enos
```

## Set credentials

In order to run most Enos scenarios, you must first:

1. Authenticate to AWS
1. Set your AWS keypair name and private key

### Authenticate to AWS

HashiCorp developers should [use Doormat](https://eng-handbook.hashicorp.services/internal-tools/enos/getting-started/#authenticate-to-aws-with-doormat) to authenticate to AWS and export the AWS credentials to the terminal.

### Set your AWS key pair name and private key

If you already have an AWS public key on your AWS account and a private key saved locally, you can skip to the [set Enos variables](#set-enos-variables) section. Otherwise, follow the below instructions for creating and saving an AWS keypair.

#### About AWS key pairs

From the AWS key pair [documentation](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-key-pairs.html):

> A key pair, consisting of a public key and a private key, is a set of security credentials that you use to prove your identity when connecting to an Amazon EC2 instance. Amazon EC2 stores the public key on your instance, and you store the private key. For Linux instances, the private key allows you to securely SSH into your instance.

#### AWS key pairs for Enos

The Enos provider requires a path to a locally stored private key that has a matching public key on your AWS EC2 instance. The provider uses this private key to connect via SSH to the instance and perform actions upon it.

The steps for using an AWS key pair in Enos are:

* Generate the key pair, if you or your team does not already have one
* Store the private key locally
* Store the public key in AWS
* Set the relevant Enos variables (commonly `aws_ssh_private_key_path` and `aws_ssh_keypair_name`)

> **Note:** Key pairs are created for a specific AWS region (e.g. “us-west-1”). If you plan to run your Enos scenario using more than one AWS region, you need to [add the key pair to each AWS region you wish to use](#add-an-existing-key-pair-to-additional-aws-regions).

#### Use an existing AWS key pair

Your team may have an existing AWS key pair that they use with Enos. If this is the case, you do not need to generate a new one. You need:

* A copy of the existing private key file
* The name of the key pair in AWS. If you don’t have this, or want to verify it, you can check it using the following steps:

1. Log into the AWS console in your browser.

3. Type “key pairs” into the search bar. In the results, under the **Features** section, click **Key pairs**.

4. Find the name of the key pair that matches your private key.

Once you have a copy of the private key file downloaded locally and the name of the key pair in AWS, you can proceed with [setting the relevant variables](#set-enos-variables) in your Enos scenario.

#### Generate an AWS key pair

If you or your team do not yet have an AWS key pair that you will use with Enos, you need to generate one. We’ll recommend two ways to do this: via  AWS or `ssh-keygen`.

##### Generate a key pair using AWS

You can generate a key pair using the AWS CLI. The public key will be stored in AWS, and the private key will be saved locally. Instructions for this are below.

1. Install the AWS CLI using the instructions [here](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html).

2. Use the [create-key-pair](https://docs.aws.amazon.com/cli/latest/reference/ec2/create-key-pair.html) command as follows. It will generate the key pair, add the public key to AWS in the region you specify, and to save the private key to a local `.pem` file.

  ```sh
  $   aws ec2 create-key-pair \
  --key-name my-key-pair \
  --key-type rsa \
  --key-format pem \
  --query "KeyMaterial" \
  --region "your-aws-region" \
  --output text > path/to/my-key-pair.pem
  ```

  * For `--key-name`, specify a name for the public key.
  * For `--key-type`, specify either `rsa` or `ed25519`. If you do not include the `--key-type` parameter, an `rsa` key is created by default. Note that `ED25519` keys are not supported for Windows instances.
  * For `--key-format`, specify either `pem` or `ppk`. If you do not include the `--key-format` parameter, a `pem` file is created by default.
  * `--query "KeyMaterial"` prints the private key material to the output.
  * `--output text > my-key-pair.pem` saves the private key material in a file with the specified extension and relative path. The extension can be either `.pem` or `.ppk`. The private key can have a name that's different from the public key name, but for ease of use, we recommend using the same name.
  * For `--region`, specify the AWS region you wish to use (e.g. `“us-west-1”`).

3. Use the following command to set the permissions of your private key file so that only you can read it.
  
  ```sh
  chmod 400 my-key-pair.pem
  ```

4. In your browser, check the **Key Pairs** page in the AWS console to verify that your new key pair is present there.

  * Go to [doormat.hashicorp.services](https://doormat.hashicorp.services/).

  * Click the arrow icon under the **Console** column for the AWS account you are using. This authenticates you to AWS and opens a new tab with the AWS console.

  * Type “key pairs” into the search bar, and click the **Key pairs** page under the **Features** section of the results.

5. Proceed to setting the relevant variables in your Enos scenario.

##### Generate a key pair using ssh-keygen

You can also generate a key pair using a third-party tool like `ssh-keygen`. `ssh-keygen` is a tool for creating new authentication key pairs for SSH. Use the below command to create a new public key and private key, both of which will be stored locally on your machine.

1. Use the following command to create a new public key and private key:
  
  ```sh
  ssh-keygen -f your-key-name -t rsa -b 4096 -N ""
  ```

  * `your-key-name` will be used to name the newly created public key file (`your-key-name.pub`) and private key file (`your-key-name`). You can also use a relative path here, e.g. `support/your-key-name`.
  * `-t rsa` sets the key type to `rsa`. `ED25519` keys are also an option, though `ED25519` keys are not supported for Windows instances.
  * `-b 4096` sets the number of bits in the key to 4096 (recommended)
  * `-N ""` sets an empty passphrase (sufficient for most users)

  > **Note:** If you prefer to use a passphrase on your SSH transport, you can do so by setting the `passphrase` or `passphrase_path` arguments on your Enos provider block.

2. Follow the instructions to import your public key into AWS.

#### Add an existing key pair to additional AWS regions

If you have an existing key pair and want to use it across multiple AWS regions, you must add it manually to each one.

1. If you have the private key stored locally but not the public key, you need to use the private key to generate the public key. This would be the case if you used the AWS CLI to generate the key pair. Run the following command to generate the public key from your private key:

  ```sh
  ssh-keygen -y -f your-key-name.pem > your-key-name.pub
  ```

  * `-y`  reads a private OpenSSH format file (like a `.pem` file) and prints an OpenSSH public key to `stdout`.
  * `-f` sets an input key file.
  * `your-key-name.pem` is the name of the private key file you have stored locally. It may or may not have a `.pem` file extension.
  * `your-key-name.pub` is the name that will be given to the public key file.

2. In your browser, navigate to the **Key Pairs** page in the AWS console.

  * Go to [doormat.hashicorp.services](https://doormat.hashicorp.services/).
  * Click the arrow icon under the **Console** column for the AWS account you are using. This authenticates you to AWS and opens a new tab with the AWS console.
  * Type “key pairs” into the search bar, and click the **Key pairs** page under the **Features** section of the results.

3. On the top right of the page, click the region dropdown and select your desired region.

4. Click the **Actions** dropdown menu, then click **Import key pair**.

5. Enter your key name.

6. Provide your public key in one of the following ways:

  * Click **Browse** and select your public key file.
  * On your command line, print the contents of your public key file with `cat your-key-name.pub`, copy them, and paste them into the text box.

7. Click **Import key pair**.

8. Verify that your public key now appears on the list of key pairs for your desired region.

## Set Enos variables

Most Enos scenarios require you to set a few variables. Common examples of these variables include:

* `aws_ssh_private_key_path`
* `aws_ssh_keypair_name`
* `<product>_license_path` when working with enterprise versions
* `artifactory_username` when using artifacts from Artifactory
* `artifactory_token` when using artifacts from Artifactory

There are several ways to set these variables:

### Set as environment variable

Set a variable as an environment variable in the terminal where you want to run Enos commands, using:

  ```sh
  export ENOS_VAR_your-var=<your_var-value>
  ```

**Tip:** To check what Enos environment variables you have set, run `printenv | grep ENOS`.

### Set in `enos.vars.hcl`

You can also set variables in a `enos*.vars.hcl` file. Some users find this convenient because it's easier to see what values you have set, compared to setting env vars in the terminal; the values also stay set if you start a new terminal session. If you choose this option, we recommend creating your own `enos-local.vars.hcl` file and setting the values there. You can use the `enos.vars.hcl` file as an example for formatting. Ensure that your file is gitignored so that you do not accidentally commit any sensitive variable values. Example:

Set a variable in your `enos.vars.hcl` file, by uncommenting the relevant line and replacing the placeholder value with your own token.
  
  ```hcl
  your-var = "your-var-value"
  ```

> **Note:** If you choose to set any sensitive values, like tokens, in the `enos.vars.hcl` file, and you’ll be committing your files to Github, make sure that your `.gitignore` file includes `enos.vars.hcl`.

## Update Enos

### Upgrade Enos via Homebrew

If you already have Enos installed, but want to update to the latest version, follow these instructions:

1. Check your current Enos version.

  ```sh
  enos version
  ```

2. Check for newer versions of your Homebrew-installed programs.

  ```sh
  brew update
  ```

If a newer version of Enos is available, it will be listed under `Outdated Formulae`.

3. Upgrade Enos.

  ```sh
  brew upgrade enos
  ```

### Upgrade Enos from a version before 0.0.18

This is a one-time case when upgrading from an earlier version of Enos to 0.0.18 or higher.

The [Enos Homebrew formula](https://github.com/hashicorp/homebrew-tap/blob/master/Formula/enos.rb) includes Terraform as a dependency. In Enos version 0.0.18, we changed the Homebrew tap that we get Terraform from. Previously, we used the community Homebrew tap; we now use the official HashiCorp Terraform Homebrew tap in order to ensure we are only using released artifacts.

This means that when you try to `brew upgrade` from an earlier version of Enos to 0.0.18 or higher, you may get the following message:

```sh
==> Installing dependencies for hashicorp/internal/enos: hashicorp/tap/terraform
Error: terraform is already installed from homebrew/core!
Please `brew uninstall terraform` first."
```

Follow these steps to resolve this:

1. Uninstall Terraform via Homebrew. The `--ignore-dependencies` flag is required, as Terraform is a dependency of Enos and therefore Homebrew will not let you uninstall it without ignoring the dependency.

  ```sh
  brew uninstall --ignore-dependencies terraform
  ```

2. Check for newer versions of your Homebrew-installed programs.

  ```sh
  brew update
  ```

3. Upgrade Enos. This will update your Enos version and download Terraform from the new (correct) tap.

  ```sh
  brew upgrade enos
  ```

## Enos for Github Actions

To use Enos in a Github Actions workflow, you can use the `action-setup-enos` Github Action. Visit the [repo](https://github.com/hashicorp/action-setup-enos) for full documentation.

You can find examples of `action-setup-enos` being used in Github Actions workflows in [Vault](https://github.com/hashicorp/vault/blob/2db5d6aa54036651f8ad289e14f95929cd39197e/.github/workflows/test-run-enos-scenario-matrix.yml#L55) repos and in [Boundary](https://github.com/hashicorp/boundary/blob/e4fbc0281af917de2c7c62e36fe9c6d5584fbdd4/.github/workflows/enos-fmt.yml#L25) repos.

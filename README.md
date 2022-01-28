# enos

Enos is the forthcoming QTI CLI application. For now, it mostly includes [reference
Terraform workspaces](/workspaces/enos-modules-dev) that utilize the `enos-provider`
and `terraform-enos-*` Terraform modules.

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

### Validate Workflow

The `validate` workflow is a re-usable GitHub workflow that is called by `PR_build` workflow, when a PR is created against the `main` branch and is also called by the `build` workflow after a PR is merged to `main` branch. This workflow runs Lint, Unit and Acceptance tests. The Acceptance tests are run on `linux/amd64` artifacts created by the caller workflows (`PR_build` or `build`).

### PR Build Workflow

The `PR_build` workflow is run when a PR is created against the `main` branch.  This workflow creates and uploads `linux/amd64` artifact. It then calls the `validate` workflow which downloads this artifact and runs Lint, Unit, and Acceptances tests on it. 

### Build Workflow
The `build` workflow is run after PR merge to `main` and only if `version.go` is updated. The `build` workflow creates build artifacts for `Linux` and `Darwin` on `amd64` and `arm64` architectures. It also creates `rpm`, `deb` packages, and `Docker` images. All created artifacts are uploaded to GH workflow. It then calls the `validate` workflow which downloads the `linux/amd64` artifact and runs Lint, Unit, and Acceptance tests on it.

### CRT Release
The `ci.hcl` is responsible for configuring the CRT workflow orchestration app. The orchestration app will read this configuration and trigger common workflows from the CRT repo. These workflows are responsible for uploading the release artifacts to Artifactory, notarizing macOS binaries, signing binaries and packages with the appropriate HashiCorp certificates, security scanning, and binary verification. The `build` workflow is a required prerequisite as it is responsible for building the release artifacts and generating the required metadata.

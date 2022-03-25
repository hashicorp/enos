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
Enos is made available as a Github release, via the `create_release` workflow. This workflow is manually triggered via the Github UI. It has three required inputs: the git SHA of the artifacts in Artifactory to be used in this release; the version number to be used in this release; and the Artifactory channel from which to download the release assets. It also has one optional input: a `pre-release` checkbox. If `pre-release` is selected, the release will be marked on the Github UI with a `pre-release` label and its tag will take the form `v<version>-pre+<first 5 characters of SHA>` i.e. `v0.0.1-pre+bd25a`. Regular release tags will take the form `v<version>` i.e. `v.0.0.1`.

To create the release, the workflow downloads the release assets from Artifactory, which have been previously notarized, signed, scanned, and verified by CRT workflows. Then, it creates a Github release and uploads these artifacts as the release assets. It automatically generates release notes, which are organized into the following categories via the four designated `changelog/` labels. At least one of these labels must be applied to each PR.
- **New Features 🎉** category includes PRs with the label `changelog/feat`
- **Bug Fixes 🐛** category includes PRs with the label `changelog/bug`
- **Other Changes 😎** category includes PRs with the label `changelog/other`
- PRs with the label `changelog/none` will be excluded from release notes.

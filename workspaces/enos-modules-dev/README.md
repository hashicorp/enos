# enos-modules-dev

This workspace creates Vault cluster with a Consul cluster backend that
services both state and service discovery. In order to utilize this provider
you must build and install the `enos-provider` [from source](https://github.com/hashicorp/enos-provider#build-from-source)

Future versions of the enos Terraform modules will allow installation of the
the provider from [the network mirror](https://github.com/hashicorp/enos-provider#network-mirror)

## Example `terraform.tfvars`
--- 
```hcl
aws_region            = "us-east-1"
aws_availability_zone = "us-east-1a"
project_name          = "qti-enos"
environment           = "dev"
ssh_aws_keypair       = "qti-aws-keypair"
common_tags           = {
  "Project Name" : "qti-enos",
  "Environment" : "dev" 
}
```

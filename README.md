# enos-modules
Enos modules are Terraform modules for quality infrastructure

Creates Vault and Consul clusters with TF Cloud backend
## Example `terraform.tfvars` for Root directory
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

# enos-modules
Enos modules are Terraform modules for quality infrastructure

Creates Vault and Consul clusters with TF Cloud backend
## Example `terraform.tfvars` for Root directory
--- 
```aws_region   = "us-east-1"
project_name = "qti-enos"
common_tags = {
  "Project Name" : "qti-enos",
  "Environment" : "dev" 
}
environment     = "dev"
ssh_aws_keypair = "qti-aws-keypair"```
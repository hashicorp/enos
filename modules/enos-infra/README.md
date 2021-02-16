# enos-infra
The enos_infra module is a module that creates a base infrastructure required by enos_vault_cluster and enos_consul_cluster, abstracting away the provider-specific details into outputs for use across scenarios

# Example Usage
```
module "enos_infra" {
  source       = "./modules/enos-infra/aws"
  project_name = var.project_name
  environment  = var.environment
  common_tags  = var.common_tags
}

data "aws_vpc" "infra" {
  id = var.vpc_id
}

data "aws_subnet_ids" "infra" {
  vpc_id = var.vpc_id
}

data "aws_subnet" "infra" {
  for_each = data.aws_subnet_ids.infra.ids
  id       = each.value
}


locals {
  infra_subnet_blocks = [for s in data.aws_subnet.infra : s.cidr_block]
}

resource "aws_instance" "vault" {
  # Second-newest LTS release
  ami           = module.enos_infra.ubuntu_ami_id
  instance_type = "t3.micro"

  key_name               = var.ssh_key_name
  # Lists, easy to use with `count.index`
  subnet_id              = local.infra_subnet_blocks[count.index]
```  
# Inputs

# Outputs
## Networking
* `vpc_id` - AWS 
* `vpc_cidr` - CIDR for the entire VPC
* `vpc_subnet_ids` - Map of IDs of the different subnets created in each availability zone
## OS
* `ubuntu_ami_id` - AMI ID of Ubuntu LTS (currently 18.04)
## Infra
* `availability_zone_names` - List of all AZs in the region
* `account_id` - AWS Account ID
## Secrets
* `kms_key_arn` - ARN of key used to encrypt secrets
* `kms_key_alias` - Alias used for above key
* `vault_license` - License file stored in KMS, encrypted with `kms_key_id`
* `consul_license` - License file stored in KMS, encrypted with `kms_key_id`
* `enos_aws_keypair` - Enos AWS KeyPair used to ssh to enos AWS instances
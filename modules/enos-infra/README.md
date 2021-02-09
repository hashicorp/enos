# enos-infra
The enos_infra module is a module that creates a base infrastructure required by enos_vault_cluster and enos_consul_cluster, abstracting away the provider-specific details into outputs for use across scenarios

# Example Usage
```
module "enos_infra" {
  source = "github.com/hashicorp/enos-modules/modules/enos_infra/aws"
}

resource "aws_instance" "vault" {
  # Second-newest LTS release
  ami           = module.enos_infra.ubuntu_ami_id
  instance_type = "t3.micro"

  key_name               = var.ssh_key_name
  # Lists, easy to use with `count.index`
  subnet_id              = module.enos_infra.vpc_subnet_ids[0]
  availability_zone      = module.enos_infra.availability_zone_names[0]
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
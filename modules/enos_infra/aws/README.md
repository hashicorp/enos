# enos-infra
The enos_infra module is a module that creates a base infrastructure required by enos_vault_cluster and enos_consul_cluster using the “aws” and “enos” providers.

This module returns the enos_aws_keypair that can be used to ssh to ENOS AWS instances which defaults to qti.  To run this module using a custom key-pair,
pass the variable "ssh_pub_key" with the value set to your public ssh key.
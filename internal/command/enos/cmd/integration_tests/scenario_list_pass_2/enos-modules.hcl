module "consul" {
  source = "hashicorp/consul/aws"
}

module "vault" {
  source = "hashicorp/vault/aws"
}

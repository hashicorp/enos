aws_region               = "us-east-1"
aws_availability_zone    = "us-east-1a"
aws_ssh_key_pair_name    = "<your-aws-ssh-key-pair-name>"
aws_ssh_private_key_path = "</fully/qualified/path/to/your/ssh/private/key.pem>"

artifactory_username = "<your-user>@hashicorp.com"
artifactory_token    = "<your-artifactory-token>"

vault_install_dir    = "/opt/vault/bin"
vault_instance_count = 3
# You must set a license path if you intend to test ent and ent.hsm of Vault >= 1.8-rc1
# Otherwise, comment this line out or set it to null
vault_license_path = "/fully/qualified/path/to/your/license.lic"

# The git SHA of the staged vault-enterprise release
vault_enterprise_product_revision = "2034d39d50597566d6e86eec08cea55affccfdcc"
# The git SHA of the staged vault OSS release
vault_oss_product_revision = "ad4b2494f7cd301169c9096b9ba314367282a887"
# The version of vault we're testing
vault_product_version = "1.8.0-rc1"
# Which editions we want to smoke test
vault_product_editions_to_test = ["oss", "ent", "ent.hsm", "prem", "prem.hsm", "pro"]

# The initial versions to install for the upgrade test
vault_enterprise_initial_release = {
  edition = "ent"
  version = "1.7.3"
}
vault_oss_initial_release = {
  edition = "oss"
  version = "1.7.0"
}

# Consul cluster
consul_license_path = null # only required for "ent" edition
consul_release = {
  version = "1.9.5"
  edition = "oss"
}

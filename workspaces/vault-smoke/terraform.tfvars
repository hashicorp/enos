aws_region                = "us-east-1"
aws_availability_zone     = "us-east-1a"
aws_ssh_key_pair_name    = "<your-aws-ssh-key-pair-name>"
aws_ssh_private_key_path = "</path/to/your/ssh/private/key.pem>"

artifactory_username = "<your-artifactory-username>"
artifactory_token    = "<your-artifactory-token>"

vault_install_dir    = "/opt/vault/bin"
vault_instance_count = 3
vault_license_path   = null # Required to test some editions of vault >= 1.8

# The git SHA of the staged vault-enterprise release
vault_enterprise_product_revision = "8ffca2568597ad7b2860ca4fa6bbb436a4445efe"
# The git SHA of the staged vault release
vault_oss_product_revision        = "534a12ac6fa226cc3c63698067d6708e5f2a2770"
# The version of vault we're staging
vault_product_version             = "1.5.9"
# Which editions we want to smoke test
vault_product_editions_to_test    = ["oss", "ent", "ent.hsm", "prem", "prem.hsm", "pro"]

# The initial versions to install for the upgrade test
vault_enterprise_initial_release = {
    edition = "ent"
    version = "1.5.8"
}
vault_oss_initial_release = {
    edition = "oss"
    version = "1.5.8"
}

# Consul cluster
consul_license_path = null # only required for "ent" edition
consul_release = {
  version = "1.9.5"
  edition = "oss"
}
